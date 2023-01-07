package database

import (
	"database/sql"
	"os"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Setup() {
	var err error
	DB, err = sql.Open("postgres", os.Getenv("PROJMGMT_DATABASE_URL"))

	if err != nil {
		panic(err)
	}

	err = DB.Ping()
	if err != nil {
		panic(err)
	}

	var boardColumnTypeExists bool
	DB.QueryRow("select exists (select 1 from pg_type where typname = 'board_column')").Scan(&boardColumnTypeExists)
	if !boardColumnTypeExists {
		DB.Exec(`create type board_column as (id uuid, name text, "order" int)`)
	}

	_, err = DB.Exec(`
		BEGIN;

			CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
			CREATE EXTENSION IF NOT EXISTS pg_trgm;
			CREATE EXTENSION IF NOT EXISTS ltree;

			CREATE TABLE IF NOT EXISTS users (
				id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
				username varchar(128) UNIQUE,
				password varchar(128),
				authorized boolean NOT NULL DEFAULT true
			);

			CREATE TABLE IF NOT EXISTS sessions (
				id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
				uid uuid NOT NULL,
				expiry timestamptz NOT NULL DEFAULT (now() + '01:00:00'::interval),
				CONSTRAINT sessions_uid_fkey FOREIGN KEY (uid)
					REFERENCES users (id) MATCH SIMPLE
					ON UPDATE NO ACTION
					ON DELETE CASCADE
			);
			CREATE INDEX IF NOT EXISTS sessions_expiry ON sessions USING btree (expiry ASC);

			CREATE TABLE IF NOT EXISTS wrong_login (
				ip character varying(50) NOT NULL,
				"time" timestamptz NOT NULL DEFAULT now()
			);
			CREATE INDEX IF NOT EXISTS wrong_login_time ON wrong_login USING btree ("time" ASC);
			CREATE INDEX IF NOT EXISTS wrong_login_ip ON wrong_login (ip);

			CREATE TABLE IF NOT EXISTS departments (
				id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
				internal_id varchar(30) UNIQUE NOT NULL,
				name varchar(200) NOT NULL,
				tsv tsvector
			);
			CREATE INDEX IF NOT EXISTS departments_tsv ON departments USING gin (tsv);


			CREATE TABLE IF NOT EXISTS user_details (
				id uuid UNIQUE NOT NULL,
				invitation_mail_key uuid DEFAULT gen_random_uuid(),
				staff_id varchar(40) UNIQUE NOT NULL,
				first_name varchar(128) NOT NULL,
				last_name varchar(128) NOT NULL,
				title varchar(128) NOT NULL,
				department_id uuid NOT NULL DEFAULT uuid_nil(),
				supervisor_id uuid NOT NULL DEFAULT uuid_nil(),
				user_right integer NOT NULL,
				email varchar(128) UNIQUE NOT NULL,
				avatar text,
				last_invite_dt timestamptz,
				date_registered_dt timestamptz,
				last_active_dt timestamptz,
				tsv tsvector,
				max_child_task_level integer NOT NULL DEFAULT 1,
				visitor boolean NOT NULL DEFAULT false,
				CONSTRAINT fk_department_id FOREIGN KEY (department_id)
					REFERENCES departments (id) MATCH SIMPLE
					ON UPDATE NO ACTION
					ON DELETE SET DEFAULT,
				CONSTRAINT fk_supervisor_id FOREIGN KEY (supervisor_id)
					REFERENCES users (id) MATCH SIMPLE
					ON UPDATE NO ACTION
					ON DELETE SET DEFAULT,
				CONSTRAINT fk_uid FOREIGN KEY (id)
					REFERENCES users (id) MATCH SIMPLE
					ON UPDATE NO ACTION
					ON DELETE CASCADE
			);
			CREATE INDEX IF NOT EXISTS user_details_tsv ON user_details USING gin (tsv);

			CREATE TABLE IF NOT EXISTS files (
				id varchar(200) PRIMARY KEY,
				name text NOT NULL,
				size bigint NOT NULL
			);

			CREATE TABLE IF NOT EXISTS ws_message_content (
				id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
				action_type varchar(100) NOT NULL,
				payload jsonb NOT NULL,
				dt timestamptz NOT NULL DEFAULT now(),
				to_all_recipients boolean NOT NULL DEFAULT false
			);

			CREATE TABLE IF NOT EXISTS ws_message_to (
				message_id uuid NOT NULL,
				uid uuid NOT NULL,
				CONSTRAINT fk_message_id FOREIGN KEY (message_id)
					REFERENCES ws_message_content (id) MATCH SIMPLE
					ON UPDATE NO ACTION
					ON DELETE CASCADE
			);

			CREATE TABLE IF NOT EXISTS chatrooms (
				id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
				room_name varchar(100),
				avatar text,
				tsv tsvector,
				tsv_w_position tsvector,
				deleted boolean NOT NULL DEFAULT false
			);
			CREATE INDEX IF NOT EXISTS chatrooms_tsv ON chatrooms USING gin (tsv);
			CREATE INDEX IF NOT EXISTS chatrooms_tsv_w_position ON chatrooms USING gin (tsv_w_position);

			CREATE TABLE IF NOT EXISTS chatrooms_users (
				rid uuid NOT NULL,
				uid uuid NOT NULL,
				last_seen timestamptz NOT NULL DEFAULT now(),
				in_users_list boolean NOT NULL DEFAULT true,
				admin boolean DEFAULT false,
				pinned boolean DEFAULT false,
				mark_as_read integer DEFAULT 0,
				CONSTRAINT chatrooms_users_rid FOREIGN KEY (rid)
					REFERENCES chatrooms (id) MATCH SIMPLE
					ON UPDATE NO ACTION
					ON DELETE NO ACTION,
				CONSTRAINT chatrooms_users_uid FOREIGN KEY (uid)
					REFERENCES users (id) MATCH SIMPLE
					ON UPDATE NO ACTION
					ON DELETE CASCADE
			);
			CREATE INDEX IF NOT EXISTS chatroom_users_rid ON chatrooms_users USING btree (rid ASC);
			CREATE UNIQUE INDEX IF NOT EXISTS chatrooms_users_rid_uid ON chatrooms_users USING btree (rid ASC, uid ASC);
			CREATE INDEX IF NOT EXISTS chatrooms_users_uid_in_users_list ON chatrooms_users USING btree (uid ASC, in_users_list ASC);

			CREATE TABLE IF NOT EXISTS chat_messages (
				id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
				rid uuid NOT NULL,
				content text,
				sender_id uuid NOT NULL,
				dt timestamptz NOT NULL DEFAULT now(),
				reply_msg_id uuid,
				reply_msg text,
				edit_time timestamptz,
				files character varying[],
				reply_msg_sender uuid,
				CONSTRAINT chat_messages_reply_msg_id FOREIGN KEY (reply_msg_id)
					REFERENCES chat_messages (id) MATCH SIMPLE
					ON UPDATE NO ACTION
					ON DELETE SET NULL,
				CONSTRAINT chat_messages_reply_msg_sender FOREIGN KEY (reply_msg_sender)
					REFERENCES users (id) MATCH SIMPLE
					ON UPDATE NO ACTION
					ON DELETE SET NULL,
				CONSTRAINT chat_messages_rid FOREIGN KEY (rid)
					REFERENCES chatrooms (id) MATCH SIMPLE
					ON UPDATE NO ACTION
					ON DELETE NO ACTION,
				CONSTRAINT chat_messages_sender_id FOREIGN KEY (sender_id)
					REFERENCES users (id) MATCH SIMPLE
					ON UPDATE NO ACTION
					ON DELETE CASCADE
			);
			CREATE INDEX IF NOT EXISTS chat_messages_rid_dt_sender_id ON chat_messages USING btree (rid ASC, dt ASC, sender_id ASC);

			CREATE TABLE IF NOT EXISTS task_custom_field_type (
				id varchar(100) PRIMARY KEY,
				type_name varchar(100) NOT NULL,
				default_value jsonb,
				list_view boolean NOT NULL DEFAULT true,
				task_details_sidebar boolean NOT NULL DEFAULT true,
				in_add_task boolean NOT NULL DEFAULT true,
				custom_field boolean NOT NULL DEFAULT true,
				edit_in_list_view boolean NOT NULL DEFAULT true
			);

			CREATE TABLE IF NOT EXISTS task_approval_list (
				id smallint PRIMARY KEY,
				name varchar(100) NOT NULL
			);

			CREATE TABLE IF NOT EXISTS tasks (
				id uuid PRIMARY KEY,
				name varchar(100) NOT NULL,
				description text,
				create_dt timestamptz NOT NULL DEFAULT now(),
				start_dt timestamptz,
				deadline_dt timestamptz,
				owner uuid NOT NULL,
				supervisors uuid[],
				participants uuid[],
				viewers uuid[],
				parents ltree NOT NULL,
				hourly_rate double precision NOT NULL DEFAULT 0,
				approval smallint NOT NULL DEFAULT 25,
				track_time boolean NOT NULL DEFAULT false,
				files text[],
				is_group_task boolean NOT NULL DEFAULT false,
				public_file_ids character varying[],
				deleted boolean NOT NULL DEFAULT false,
				delete_dt timestamptz,
				tsv tsvector NOT NULL,
				assignee uuid,
				CONSTRAINT tasks_approval_id FOREIGN KEY (approval)
					REFERENCES task_approval_list (id) MATCH SIMPLE
					ON UPDATE NO ACTION
					ON DELETE NO ACTION,
				CONSTRAINT tasks_owner FOREIGN KEY (owner)
					REFERENCES users (id) MATCH SIMPLE
					ON UPDATE NO ACTION
					ON DELETE CASCADE,
				CONSTRAINT tasks_assignee FOREIGN KEY (assignee)
					REFERENCES users (id) MATCH SIMPLE
					ON UPDATE NO ACTION
					ON DELETE SET NULL
			);
			CREATE INDEX IF NOT EXISTS tasks_tsv ON tasks USING gin (tsv);

			CREATE TABLE IF NOT EXISTS task_approval_record (
				id uuid PRIMARY KEY,
				uid uuid NOT NULL,
				task_id uuid NOT NULL,
				status smallint NOT NULL,
				dt timestamptz NOT NULL,
				CONSTRAINT task_approval_record_uid FOREIGN KEY (uid)
					REFERENCES users (id) MATCH SIMPLE
					ON UPDATE NO ACTION
					ON DELETE CASCADE,
				CONSTRAINT task_approval_record_taskid FOREIGN KEY (task_id)
					REFERENCES tasks (id) MATCH SIMPLE
					ON UPDATE NO ACTION
					ON DELETE CASCADE
			);

			CREATE TABLE IF NOT EXISTS task_comments (
				id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
				task_id uuid NOT NULL,
				sender uuid NOT NULL,
				content text,
				dt timestamptz NOT NULL DEFAULT now(),
				files character varying[],
				reply_comment_id uuid,
				reply_comment text,
				reply_comment_sender uuid,
				edit_time timestamptz,
				public_file_ids text[],
				deleted boolean NOT NULL DEFAULT false,
				delete_time timestamptz,
				CONSTRAINT task_comments_task_id FOREIGN KEY (task_id)
					REFERENCES tasks (id) MATCH SIMPLE
					ON UPDATE NO ACTION
					ON DELETE CASCADE,
				CONSTRAINT task_comments_sender FOREIGN KEY (sender)
					REFERENCES users (id) MATCH SIMPLE
					ON UPDATE NO ACTION
					ON DELETE CASCADE,
				CONSTRAINT task_comments_reply_comment_id FOREIGN KEY (reply_comment_id)
					REFERENCES task_comments (id) MATCH SIMPLE
					ON UPDATE NO ACTION
					ON DELETE SET NULL,
				CONSTRAINT task_comments_reply_comment_sender FOREIGN KEY (reply_comment_sender)
					REFERENCES users (id) MATCH SIMPLE
					ON UPDATE NO ACTION
					ON DELETE SET NULL
			);

			CREATE TABLE IF NOT EXISTS task_record (
				id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
				task_id uuid NOT NULL,
				requester uuid NOT NULL,
				action varchar(30) NOT NULL,
				approval integer,
				dt timestamptz NOT NULL DEFAULT now(),
				add_personnel uuid[],
				remove_personnel uuid[],
				CONSTRAINT task_record_task_id FOREIGN KEY (task_id)
					REFERENCES tasks (id) MATCH SIMPLE
					ON UPDATE NO ACTION
					ON DELETE CASCADE,
				CONSTRAINT task_record_requester FOREIGN KEY (requester)
					REFERENCES users (id) MATCH SIMPLE
					ON UPDATE NO ACTION
					ON DELETE CASCADE
			);

			CREATE TABLE IF NOT EXISTS task_time_track (
				id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
				task_id uuid NOT NULL,
				uid uuid,
				start_dt timestamptz NOT NULL DEFAULT now(),
				end_dt timestamptz,
				CONSTRAINT task_time_track_task_id FOREIGN KEY (task_id)
					REFERENCES tasks (id) MATCH SIMPLE
					ON UPDATE NO ACTION
					ON DELETE CASCADE,
				CONSTRAINT task_time_track_uid FOREIGN KEY (uid)
					REFERENCES users (id) MATCH SIMPLE
					ON UPDATE NO ACTION
					ON DELETE SET NULL
			);

			CREATE TABLE IF NOT EXISTS task_custom_user_fields (
				id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
				uid uuid NOT NULL,
				field_type varchar(100),
				details jsonb,
				field_name varchar(100),
				CONSTRAINT task_custom_user_fields_uid FOREIGN KEY (uid)
					REFERENCES users (id) MATCH SIMPLE
					ON UPDATE NO ACTION
					ON DELETE CASCADE
			);

			ALTER TABLE task_custom_user_fields DROP CONSTRAINT IF EXISTS task_custom_user_fields_uid_type;
			ALTER TABLE task_custom_user_fields ADD CONSTRAINT task_custom_user_fields_uid_type UNIQUE (uid,field_type);

			CREATE TABLE IF NOT EXISTS task_custom_user_field_values (
				uid uuid NOT NULL,
				task_id uuid NOT NULL,
				"values" jsonb NOT NULL,
				CONSTRAINT task_custom_user_field_values_task_id FOREIGN KEY (task_id)
					REFERENCES tasks (id) MATCH SIMPLE
					ON UPDATE NO ACTION
					ON DELETE CASCADE,
				CONSTRAINT task_custom_user_field_values_uid FOREIGN KEY (uid)
					REFERENCES users (id) MATCH SIMPLE
					ON UPDATE NO ACTION
					ON DELETE CASCADE
			);

			INSERT INTO task_approval_list (id,name) VALUES
				(0,'rejected'),
				(25,'not submitted'),
				(50,'changes requested'),
				(75,'approval pending'),
				(100,'approved')
			ON CONFLICT (id) DO NOTHING;

			INSERT INTO task_custom_field_type (
				id,
				type_name,
				list_view,
				task_details_sidebar,
				in_add_task,
				custom_field,
				edit_in_list_view
			) VALUES (
				'short_text',
				'Short Text',
				true,
				true,
				true,
				true,
				true
			),
			(
				'long_text',
				'Formatted Text',
				false,
				true,
				true,
				true,
				false
			),
			(
				'people',
				'People',
				true,
				true,
				true,
				true,
				true
			),
			(
				'number',
				'Number',
				true,
				true,
				true,
				true,
				true
			),
			(
				'tags',
				'Tags',
				true,
				true,
				true,
				true,
				true
			),
			(
				'link',
				'Link',
				true,
				true,
				true,
				true,
				true
			),
			(
				'dropdown',
				'Dropdown',
				true,
				true,
				true,
				true,
				false
			),
			(
				'checkbox',
				'Checkbox',
				true,
				true,
				true,
				true,
				false
			),
			(
				'date',
				'Date',
				true,
				true,
				true,
				true,
				true
			),
			(
				'single_person',
				'Single Person',
				true,
				true,
				true,
				true,
				true
			),
			(
				'files',
				'Files',
				false,
				true,
				false,
				false,
				false
			),
			(
				'approval',
				'Approval',
				true,
				true,
				false,
				false,
				false
			),
			(
				'assignee',
				'Assignee',
				true,
				true,
				true,
				false,
				true
			),
			(
				'child_status',
				'Child Tasks Status',
				true,
				false,
				false,
				false,
				false
			),
			(
				'parents',
				'Parents',
				true,
				true,
				true,
				false,
				true
			),
			(
				'timer',
				'Timer',
				true,
				true,
				false,
				false,
				false
			)
			ON CONFLICT (id) DO NOTHING;

			INSERT INTO task_custom_field_type (
				id,
				type_name,
				default_value,
				list_view,
				task_details_sidebar,
				in_add_task,
				custom_field,
				edit_in_list_view
			) VALUES
			(
				'board_column',
				'Board Column',
				'{"default": "3fb03582-8a8e-4290-aa87-b2d1a8455bbd", "options": [{"id": "3fb03582-8a8e-4290-aa87-b2d1a8455bbd", "name": "To Do", "order": 0}, {"id": "189cf03d-cf81-457c-a7ab-e1f02bbdce03", "name": "In Progress", "order": 1}, {"id": "cf7d0582-90e1-48d9-8c6d-644447a0b04e", "name": "Done", "order": 2}]}',
				false,
				true,
				false,
				false,
				false
			),
			(
				'order_in_board_column',
				'Order in Board Column',
				'{"default": 0}',
				false,
				false,
				false,
				false,
				false
			)
			ON CONFLICT (id) DO NOTHING;

			INSERT INTO users (id) VALUES ('00000000-0000-0000-0000-000000000000') ON CONFLICT (id) DO NOTHING;

			INSERT INTO departments (
				id,
				internal_id,
				name
			) VALUES (
				'00000000-0000-0000-0000-000000000000',
				'',
				''
			) 
			ON CONFLICT (id) DO NOTHING;

			INSERT INTO user_details (
				id,
				staff_id,
				first_name,
				last_name,
				title,
				department_id,
				supervisor_id,
				user_right,
				email,
				max_child_task_level,
				visitor
			) VALUES (
				'00000000-0000-0000-0000-000000000000',
				'00000000-0000-0000-0000-000000000000',
				'(No',
				'Supervisor)',
				'',
				'00000000-0000-0000-0000-000000000000',
				'00000000-0000-0000-0000-000000000000',
				0,
				'',
				1,
				false
			)
			ON CONFLICT (id) DO NOTHING;
		END;
	`)
	if err != nil {
		panic(err)
	}

	_, err = DB.Exec(`
		INSERT INTO users (
			id,
			username,
			password
		) VALUES (
			$1,
			$2,
			$3
		)
		ON CONFLICT (id) DO NOTHING;
	`,
		os.Getenv("PROJMGMT_DEMO_USER"),
		os.Getenv("PROJMGMT_DEMO_USERNAME"),
		os.Getenv("PROJMGMT_DEMO_USER_PASSWORD"),
	)
	if err != nil {
		panic(err)
	}

	_, err = DB.Exec(`
		INSERT INTO user_details (
			id,
			invitation_mail_key,
			staff_id,
			first_name,
			last_name,
			title,
			department_id,
			supervisor_id,
			user_right,
			email,
			avatar,
			date_registered_dt,
			last_active_dt,
			tsv,
			max_child_task_level,
			visitor
		) VALUES (
			$1,
			NULL,
			gen_random_uuid()::text,
			'Cindy',
			'Ho',
			'',
			'00000000-0000-0000-0000-000000000000',
			'00000000-0000-0000-0000-000000000000',
			15,
			$2,
			$3,
			now(),
			now(),
			to_tsvector('Cindy Ho'),
			5,
			false
		)
		ON CONFLICT (id) DO NOTHING;
	`,
		os.Getenv("PROJMGMT_DEMO_USER"),
		os.Getenv("PROJMGMT_DEMO_USER_EMAIL"),
		os.Getenv("PROJMGMT_DEMO_USER_AVATAR"),
	)
	if err != nil {
		panic(err)
	}

	_, err = DB.Exec(`
		WITH source AS (
			SELECT id, $1::uuid as uid, type_name, default_value FROM task_custom_field_type WHERE default_value IS NOT NULL
		)
		INSERT INTO
			task_custom_user_fields
			(uid,field_type,details,field_name)
		SELECT
			uid, id, default_value, type_name
		FROM
			source
		ON CONFLICT ON CONSTRAINT task_custom_user_fields_uid_type DO NOTHING;
	`,
		os.Getenv("PROJMGMT_DEMO_USER"),
	)
	if err != nil {
		panic(err)
	}
}

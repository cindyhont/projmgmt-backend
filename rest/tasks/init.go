package tasks

import (
	"encoding/json"
	"fmt"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/lib/pq"
)

type Approval struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type ApprovalEntity struct {
	IDs      []int            `json:"ids"`
	Entities map[int]Approval `json:"entities"`
}

type TaskRecord struct {
	ID              string   `json:"id"`
	TaskID          string   `json:"taskID"`
	Requester       string   `json:"requester"`
	Action          string   `json:"action"`
	Approval        int      `json:"approval"`
	AddPersonnel    []string `json:"addPersonnel"`
	RemovePersonnel []string `json:"removePersonnel"`
	DateTime        int64    `json:"dt"`
}

type TaskRecordEntity struct {
	IDs      []string              `json:"ids"`
	Entities map[string]TaskRecord `json:"entities"`
}

type CustomFieldType struct {
	ID                 string `json:"id"`
	TypeName           string `json:"typeName"`
	ListView           bool   `json:"listView"`
	TaskDetailsSidebar bool   `json:"taskDetailsSidebar"`
	InAddTask          bool   `json:"inAddTask"`
	CustomField        bool   `json:"customField"`
	EditInListView     bool   `json:"editInListView"`
}

type CustomFieldTypeEntity struct {
	IDs      []string                   `json:"ids"`
	Entities map[string]CustomFieldType `json:"entities"`
}

type UserCustomFields struct {
	ID        string                 `json:"id"`
	FieldType string                 `json:"fieldType"`
	Details   map[string]interface{} `json:"details"`
	FieldName string                 `json:"fieldName"`
}

type UserCustomFieldsEntity struct {
	IDs      []string                    `json:"ids"`
	Entities map[string]UserCustomFields `json:"entities"`
}

type Comment struct {
	ID             string   `json:"id"`
	TaskID         string   `json:"taskID"`
	Content        string   `json:"content"`
	Sender         string   `json:"sender"`
	DateTime       int64    `json:"dt"`
	ReplyMsgID     string   `json:"replyMsgID"`
	ReplyMsg       string   `json:"replyMsg"`
	ReplyMsgSender string   `json:"replyMsgSender"`
	EditDateTime   int64    `json:"editDt"`
	FileIDs        []string `json:"fileIDs"`
	Sent           bool     `json:"sent"`
	Deleted        bool     `json:"deleted"`
	DeleteDateTime int64    `json:"deleteDT"`
}

type CommentEntity struct {
	IDs      []string           `json:"ids"`
	Entities map[string]Comment `json:"entities"`
}

type TimeRecord struct {
	ID      string `json:"id"`
	TaskID  string `json:"taskID"`
	UserID  string `json:"uid"`
	StartDT int64  `json:"start"`
	EndDT   int64  `json:"end"`
}

type TimeRecordEntity struct {
	IDs      []string              `json:"ids"`
	Entities map[string]TimeRecord `json:"entities"`
}

type Task struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	CreateDT      int64    `json:"createDT"`
	StartDT       int64    `json:"startDT"`
	DeadlineDT    int64    `json:"deadlineDT"`
	Owner         string   `json:"owner"`
	IsGroupTask   bool     `json:"isGroupTask"`
	Supervisors   []string `json:"supervisors"`
	Participants  []string `json:"participants"`
	Viewers       []string `json:"viewers"`
	Parents       []string `json:"parents"`
	TrackTime     bool     `json:"trackTime"`
	HourlyRate    float32  `json:"hourlyRate"`
	FileIDs       []string `json:"fileIDs"`
	FilesToDelete []string `json:"filesToDelete"`
	Sent          bool     `json:"sent"`
	Approval      int      `json:"approval"`
	Assignee      string   `json:"assignee"`
}

type TaskEntity struct {
	IDs      []string                          `json:"ids"`
	Entities map[string]map[string]interface{} `json:"entities"`
}

func GetInitTasksEntity(uid string) (ids *[]string, myTaskIDs *[]string, entities *map[string]Task, userIDs *[]string, fileIDs *[]string) {
	_ids := make([]string, 0)
	_myTaskIDs := make([]string, 0)
	_entities := make(map[string]Task)
	_userIDs := make([]string, 0)
	_fileIDs := make([]string, 0)

	/*
		rows, err := database.DB.Query(`
			WITH querystr AS (
				SELECT
					'*{0,1}.' || REPLACE(STRING_AGG(id::text,'|'),'-','') || '.*{0,5}' AS q
				FROM
					tasks
				WHERE
					deleted = FALSE
				AND
				(
					owner = $1
					OR
					is_group_task
					AND
					(
						assignee = $1
						OR
						array_position(supervisors,$1) is not null
						OR
						array_position(participants,$1) is not null
						OR
						array_position(viewers,$1) is not null
					)
				)
			)
			SELECT
				id,
				name,
				coalesce(description,''),
				floor(extract(epoch from create_dt) * 1000) as create_dt,
				coalesce(floor(extract(epoch from start_dt) * 1000),0) as start_dt,
				coalesce(floor(extract(epoch from deadline_dt) * 1000),0) as deadline_dt,
				owner,
				coalesce(supervisors,array[]::uuid[]) as supervisors,
				coalesce(participants,array[]::uuid[]) as participants,
				coalesce(viewers,array[]::uuid[]) as viewers,
				string_to_array(ltree2text(parents),'.')::uuid[],
				hourly_rate,
				track_time,
				coalesce(files,array[]::varchar[]) as files,
				is_group_task,
				approval,
				assignee
			FROM
				tasks,
				querystr
			WHERE
				parents ~ querystr.q::lquery
		`, uid)
		if err != nil {
			return &_ids, &_entities, &_userIDs, &_fileIDs
		}
	*/

	rows, err := database.DB.Query(`
		SELECT
			id,
			name,
			coalesce(description,''),
			floor(extract(epoch from create_dt) * 1000) as create_dt,
			coalesce(floor(extract(epoch from start_dt) * 1000),0) as start_dt,
			coalesce(floor(extract(epoch from deadline_dt) * 1000),0) as deadline_dt,
			owner,
			coalesce(supervisors,array[]::uuid[]) as supervisors,
			coalesce(participants,array[]::uuid[]) as participants,
			coalesce(viewers,array[]::uuid[]) as viewers,
			string_to_array(ltree2text(parents),'.')::uuid[],
			hourly_rate,
			track_time,
			coalesce(files,array[]::varchar[]) as files,
			is_group_task,
			approval,
			assignee
		FROM
			tasks
		WHERE
			deleted = FALSE
		AND
		(
			owner = $1
			OR
			is_group_task
			AND
			(
				assignee = $1
				OR
				array_position(supervisors,$1) is not null
				OR 
				array_position(participants,$1) is not null
				OR 
				array_position(viewers,$1) is not null
			)
		)
	`, uid)
	if err != nil {
		fmt.Println(err)
		return &_ids, &_myTaskIDs, &_entities, &_userIDs, &_fileIDs
	}

	defer rows.Close()

	for rows.Next() {
		var t Task
		rows.Scan(
			&t.ID,
			&t.Name,
			&t.Description,
			&t.CreateDT,
			&t.StartDT,
			&t.DeadlineDT,
			&t.Owner,
			pq.Array(&t.Supervisors),
			pq.Array(&t.Participants),
			pq.Array(&t.Viewers),
			pq.Array(&t.Parents),
			&t.HourlyRate,
			&t.TrackTime,
			pq.Array(&t.FileIDs),
			&t.IsGroupTask,
			&t.Approval,
			&t.Assignee,
		)
		t.Sent = true
		t.FilesToDelete = make([]string, 0)

		_ids = append(_ids, t.ID)
		_myTaskIDs = append(_myTaskIDs, t.ID)
		_userIDs = append(_userIDs, t.Owner)
		if len(t.Supervisors) != 0 {
			_userIDs = append(_userIDs, t.Supervisors...)
		}
		if len(t.Participants) != 0 {
			_userIDs = append(_userIDs, t.Participants...)
		}
		if len(t.Viewers) != 0 {
			_userIDs = append(_userIDs, t.Viewers...)
		}
		if len(t.FileIDs) != 0 {
			_fileIDs = append(_fileIDs, t.FileIDs...)
		}
		_entities[t.ID] = t
	}

	// fetch parents and children
	rowsC, errC := database.DB.Query(`
		WITH source AS (
			SELECT
				id,
				parents
			FROM
				tasks
			WHERE
				deleted = FALSE
			AND
			(
				owner = $1
				OR
				is_group_task
				AND
				(
					assignee = $1
					OR
					array_position(supervisors,$1) is not null
					OR
					array_position(participants,$1) is not null
					OR
					array_position(viewers,$1) is not null
				)
			)
		),
		source_ids AS (
			SELECT id FROM source
			UNION
			SELECT 
				DISTINCT ltree2text(subpath(parents,nlevel(parents)-2,1))::uuid
			FROM
				source
			WHERE
				nlevel(parents) > 1
		),
		querystr AS (
			SELECT
				'*{0,1}.' || REPLACE(STRING_AGG(DISTINCT id::text,'|'),'-','') || '.*{0,5}' AS q
			FROM
				source_ids
		),
		task_ids as (
			SELECT
				id
			FROM
				tasks,
				querystr
			WHERE
				parents ~ querystr.q::lquery
			except
			SELECT
				id
			FROM
				tasks
			WHERE
				deleted = FALSE
			AND
			(
				owner = $1
				OR
				is_group_task
				AND
				(
					assignee = $1
					OR
					array_position(supervisors,$1) is not null
					OR
					array_position(participants,$1) is not null
					OR
					array_position(viewers,$1) is not null
				)
			)
		)
		select 
			T.id,
			T.name,
			coalesce(T.description,''),
			floor(extract(epoch from T.create_dt) * 1000) as create_dt,
			coalesce(floor(extract(epoch from T.start_dt) * 1000),0) as start_dt,
			coalesce(floor(extract(epoch from T.deadline_dt) * 1000),0) as deadline_dt,
			T.owner,
			coalesce(T.supervisors,array[]::uuid[]) as supervisors,
			coalesce(T.participants,array[]::uuid[]) as participants,
			coalesce(T.viewers,array[]::uuid[]) as viewers,
			string_to_array(ltree2text(T.parents),'.')::uuid[],
			T.hourly_rate,
			T.track_time,
			coalesce(T.files,array[]::varchar[]) as files,
			T.is_group_task,
			T.approval,
			T.assignee
		from 
			tasks T 
		inner join 
			task_ids TI 
		on 
			T.id = TI.id
	`, uid)
	if errC != nil {
		return &_ids, &_myTaskIDs, &_entities, &_userIDs, &_fileIDs
	}

	defer rowsC.Close()

	for rowsC.Next() {
		var t Task
		rowsC.Scan(
			&t.ID,
			&t.Name,
			&t.Description,
			&t.CreateDT,
			&t.StartDT,
			&t.DeadlineDT,
			&t.Owner,
			pq.Array(&t.Supervisors),
			pq.Array(&t.Participants),
			pq.Array(&t.Viewers),
			pq.Array(&t.Parents),
			&t.HourlyRate,
			&t.TrackTime,
			pq.Array(&t.FileIDs),
			&t.IsGroupTask,
			&t.Approval,
			&t.Assignee,
		)
		t.Sent = true
		t.FilesToDelete = make([]string, 0)

		_ids = append(_ids, t.ID)
		_userIDs = append(_userIDs, t.Owner)
		_entities[t.ID] = t
	}

	return &_ids, &_myTaskIDs, &_entities, &_userIDs, &_fileIDs
}

func GetApprovalList() (*[]int, *map[int]Approval) {
	ids := make([]int, 0)
	entities := make(map[int]Approval)

	rows, err := database.DB.Query(`
		SELECT
			id,
			name
		FROM
			task_approval_list
	`)
	if err != nil {
		fmt.Println(err)
		return &ids, &entities
	}

	defer rows.Close()

	for rows.Next() {
		var a Approval
		rows.Scan(&a.ID, &a.Name)
		ids = append(ids, a.ID)
		entities[a.ID] = a
	}

	return &ids, &entities
}

func GetCustomFieldTypes() (*[]string, *map[string]CustomFieldType) {
	ids := make([]string, 0)
	entities := make(map[string]CustomFieldType)

	rows, err := database.DB.Query(`
		SELECT
			id,
			type_name,
			list_view,
			task_details_sidebar,
			in_add_task,
			custom_field,
			edit_in_list_view
		FROM
			task_custom_field_type
	`)
	if err != nil {
		fmt.Println(err)
		return &ids, &entities
	}

	defer rows.Close()

	for rows.Next() {
		var f CustomFieldType
		rows.Scan(
			&f.ID,
			&f.TypeName,
			&f.ListView,
			&f.TaskDetailsSidebar,
			&f.InAddTask,
			&f.CustomField,
			&f.EditInListView,
		)
		ids = append(ids, f.ID)
		entities[f.ID] = f
	}
	return &ids, &entities
}

func GetUserCustomFields(uid string) *[]UserCustomFields {
	arr := make([]UserCustomFields, 0)

	rows, err := database.DB.Query(`
		SELECT
			id,
			field_type,
			field_name,
			details
		FROM
			task_custom_user_fields
		WHERE
			uid = $1
	`, uid)
	if err != nil {
		fmt.Println(err)
		return &arr
	}

	defer rows.Close()

	for rows.Next() {
		var f UserCustomFields
		var s string
		rows.Scan(
			&f.ID,
			&f.FieldType,
			&f.FieldName,
			&s,
		)
		json.Unmarshal([]byte(s), &f.Details)
		arr = append(arr, f)
	}

	return &arr
}

func GetTimeRecords(taskIDs *[]string) *[]TimeRecord {
	result := make([]TimeRecord, 0)

	rows, err := database.DB.Query(`
		SELECT
			id,
			task_id,
			uid,
			coalesce(floor(extract(epoch from start_dt) * 1000),0) as start_dt,
			coalesce(floor(extract(epoch from end_dt) * 1000),0) as end_dt
		FROM
			task_time_track
		WHERE
			task_id = ANY($1)
	`, pq.Array(taskIDs))
	if err != nil {
		return &result
	}

	for rows.Next() {
		var t TimeRecord
		rows.Scan(&t.ID, &t.TaskID, &t.UserID, &t.StartDT, &t.EndDT)
		result = append(result, t)
	}
	return &result
}

func GetTimeRecordsEntity(taskIDs *[]string) (ids *[]string, entities *map[string]TimeRecord, userIDs *[]string) {
	_ids := make([]string, 0)
	_entities := make(map[string]TimeRecord)
	_userIDs := make([]string, 0)

	arr := *GetTimeRecords(taskIDs)

	for _, t := range arr {
		_ids = append(_ids, t.ID)
		_userIDs = append(_userIDs, t.UserID)
		_entities[t.ID] = t
	}
	return &_ids, &_entities, &_userIDs
}

func GetTaskRecords(taskIDs *[]string) *[]TaskRecord {
	result := make([]TaskRecord, 0)

	rows, err := database.DB.Query(`
		SELECT
			id,
			task_id,
			requester,
			action,
			coalesce(approval,0) as approval,
			floor(extract(epoch from dt) * 1000) as dt,
			coalesce(add_personnel,array[]::uuid[]) as add_personnel,
			coalesce(remove_personnel,array[]::uuid[]) as remove_personnel
		FROM
			task_record
		WHERE
			task_id = ANY($1)
	`, pq.Array(*taskIDs))
	if err != nil {
		return &result
	}

	defer rows.Close()

	for rows.Next() {
		var r TaskRecord
		rows.Scan(
			&r.ID,
			&r.TaskID,
			&r.Requester,
			&r.Action,
			&r.Approval,
			&r.DateTime,
			pq.Array(&r.AddPersonnel),
			pq.Array(&r.RemovePersonnel),
		)
		result = append(result, r)
	}
	return &result
}

func GetTaskRecordsEntity(taskIDs *[]string) (ids *[]string, entities *map[string]TaskRecord, userIDs *[]string) {
	_ids := make([]string, 0)
	_entities := make(map[string]TaskRecord)
	_userIDs := make([]string, 0)

	arr := *GetTaskRecords(taskIDs)
	for _, r := range arr {
		_ids = append(_ids, r.ID)
		_userIDs = append(_userIDs, r.Requester)
		if len(r.AddPersonnel) != 0 {
			_userIDs = append(_userIDs, r.AddPersonnel...)
		}
		if len(r.RemovePersonnel) != 0 {
			_userIDs = append(_userIDs, r.RemovePersonnel...)
		}
		_entities[r.ID] = r
	}

	return &_ids, &_entities, &_userIDs
}

func GetComments(taskIDs *[]string) *[]Comment {
	result := make([]Comment, 0)
	rows, err := database.DB.Query(`
		SELECT
			id,
			task_id,
			sender,
			coalesce(content,''),
			floor(extract(epoch from dt) * 1000) as dt,
			coalesce(files,array[]::varchar[]) as file_ids,
			coalesce(reply_comment_id::varchar,'') as reply_comment_id,
			coalesce(reply_comment,'') as reply_comment,
			coalesce(reply_comment_sender::varchar,'') as reply_comment_sender,
			coalesce(floor(extract(epoch from edit_time) * 1000),0) as edit_time,
			deleted,
			coalesce(floor(extract(epoch from delete_time) * 1000),0) as deleted_time
		FROM
			task_comments
		WHERE 
			task_id = ANY($1)
	`, pq.Array(*taskIDs))
	if err != nil {
		return &result
	}

	defer rows.Close()

	for rows.Next() {
		var c Comment
		rows.Scan(
			&c.ID,
			&c.TaskID,
			&c.Sender,
			&c.Content,
			&c.DateTime,
			pq.Array(&c.FileIDs),
			&c.ReplyMsgID,
			&c.ReplyMsg,
			&c.ReplyMsgSender,
			&c.EditDateTime,
			&c.Deleted,
			&c.DeleteDateTime,
		)
		c.Sent = true
		result = append(result, c)
	}
	return &result
}

func GetCommentsEntity(taskIDs *[]string) (ids *[]string, entities *map[string]Comment, userIDs *[]string, fileIDs *[]string) {
	_ids := make([]string, 0)
	_entities := make(map[string]Comment)
	_userIDs := make([]string, 0)
	_fileIDs := make([]string, 0)

	arr := *GetComments(taskIDs)
	for _, c := range arr {
		_ids = append(_ids, c.ID)
		_userIDs = append(_userIDs, c.Sender)
		if c.ReplyMsgSender != "" {
			_userIDs = append(_userIDs, c.ReplyMsgSender)
		}
		if len(c.FileIDs) != 0 {
			_fileIDs = append(_fileIDs, c.FileIDs...)
		}
		_entities[c.ID] = c
	}
	return &_ids, &_entities, &_userIDs, &_fileIDs
}

func GetUserCustomFieldValues(uid string) (bool, *map[string]map[string]interface{}) {
	entities := make(map[string]map[string]interface{})

	rows, err := database.DB.Query(`
		SELECT
			task_id,
			values
		FROM
			task_custom_user_field_values
		WHERE
			uid = $1
	`, uid)
	if err != nil {
		fmt.Println(err)
		return false, &entities
	}

	defer rows.Close()

	for rows.Next() {
		var id, s string
		var values map[string]interface{}
		rows.Scan(
			&id,
			&s,
		)
		json.Unmarshal([]byte(s), &values)
		entities[id] = values
	}

	return true, &entities
}

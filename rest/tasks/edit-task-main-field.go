package tasks

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/cindyhont/projmgmt-backend/rest/common"
	"github.com/cindyhont/projmgmt-backend/websocket"
	"github.com/julienschmidt/httprouter"
	"github.com/lib/pq"
)

type editFieldRequest struct {
	TaskID     string      `json:"taskID"`
	Field      string      `json:"field"`
	Value      interface{} `json:"value"`
	TaskRecord TaskRecord  `json:"taskRecord,omitempty"`
}

func editTaskMainField(
	w http.ResponseWriter,
	r *http.Request,
	p httprouter.Params,
	s *model.Session,
	signedIn bool,
	uid string,
) {
	if !signedIn {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	data := struct {
		Success      bool     `json:"success"`
		WsRequestIDs []string `json:"wsids"`
	}{
		Success:      false,
		WsRequestIDs: make([]string, 0),
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	var req editFieldRequest

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	existingUserIDs := *getTaskUserIDs(req.TaskID)

	rv := reflect.ValueOf(req.Value)

	if strings.HasSuffix(req.Field, "_dt") {
		_, err = database.DB.Exec(fmt.Sprintf("UPDATE tasks SET %s = $1 WHERE id = $2", req.Field), timeOrNil(int64(req.Value.(float64))), req.TaskID)
		if err != nil {
			json.NewEncoder(w).Encode(data)
			return
		}
	} else if rv.Kind() == reflect.Slice {
		var arrValue []interface{}
		length := rv.Len()
		for i := 0; i < length; i++ {
			arrValue = append(arrValue, rv.Index(i).Interface())
		}

		_, err = database.DB.Exec(fmt.Sprintf("UPDATE tasks SET %s = $1 WHERE id = $2", req.Field), pq.Array(arrValue), req.TaskID)
		if err != nil {
			fmt.Println(err)
			json.NewEncoder(w).Encode(data)
			return
		}
	} else if req.Field == "name" {
		_, err = database.DB.Exec("UPDATE tasks SET name = $1, tsv = to_tsvector($2) WHERE id = $3", req.Value, req.Value, req.TaskID)
		if err != nil {
			json.NewEncoder(w).Encode(data)
			return
		}
	} else {
		_, err = database.DB.Exec(fmt.Sprintf("UPDATE tasks SET %s = $1 WHERE id = $2", req.Field), req.Value, req.TaskID)
		if err != nil {
			json.NewEncoder(w).Encode(data)
			return
		}
	}

	if req.TaskRecord.DateTime != 0 {
		database.DB.Exec(
			"INSERT INTO task_record (id,task_id,requester,action,approval,dt,add_personnel,remove_personnel) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)",
			req.TaskRecord.ID,
			req.TaskRecord.TaskID,
			req.TaskRecord.Requester,
			req.TaskRecord.Action,
			approvalOrNil(&req.TaskRecord),
			time.UnixMilli(req.TaskRecord.DateTime),
			arrayOrNil(&req.TaskRecord.AddPersonnel),
			arrayOrNil(&req.TaskRecord.RemovePersonnel),
		)
	}

	userIDs := *getTaskUserIDs(req.TaskID)

	newUserIDs := *getExtraStringsFromSlices(&userIDs, &existingUserIDs)

	if len(newUserIDs) != 0 {
		// update board column for returning users, and for new users set task to default column
		for _, userID := range newUserIDs {
			var exists bool
			database.DB.QueryRow(
				"SELECT EXISTS (SELECT 1 FROM task_custom_user_field_values WHERE uid = $1 AND task_id = $2)",
				userID,
				req.TaskID,
			).Scan(&exists)

			if exists {
				_, err = database.DB.Exec(`
					WITH get_current_board_column_options AS (
						SELECT
							(jsonb_populate_record(null::board_column,jsonb_array_elements(details->'options'))).*
						FROM
							task_custom_user_fields
						WHERE
							uid = $1
						AND
							field_type = 'board_column'
					),
					get_board_column_field AS (
						SELECT
							id::text,
							details->>'default' as default_v
						FROM
							task_custom_user_fields
						WHERE
							uid = $1
						AND
							field_type = 'board_column'
					),
					get_order_in_board_column_field AS (
						SELECT
							id::text
						FROM
							task_custom_user_fields
						WHERE
							uid = $1
						AND
							field_type = 'order_in_board_column'
					),
					get_current_board_column AS (
						SELECT
							V.values->>CF.id as curr_col
						FROM
							task_custom_user_field_values V,
							get_board_column_field CF
						WHERE
							uid = $1
						AND
							task_id = $2
					),
					get_board_column_still_exists AS (
						SELECT EXISTS (
							SELECT 1
							FROM
								get_current_board_column_options OP,
								get_current_board_column CC
							WHERE
								OP.id = CC.curr_col::uuid
						) as curr_col_exists
					),
					if_curr_col_not_exists AS (
						SELECT
							V.task_id,
							CF.default_v as col,
							row_number() OVER (order by V.values->>OF.id) as seq,
							false as col_exists
						FROM
							task_custom_user_field_values V,
							get_order_in_board_column_field OF,
							get_board_column_field CF
						WHERE
							V.uid = $1
						AND
							V.values->>CF.id = CF.default_v
						UNION
						SELECT $2, CF.default_v, 0, false FROM get_board_column_field CF LIMIT 1
					),
					if_curr_col_exists AS (
						SELECT
							V.task_id,
							CC.curr_col as col,
							row_number() OVER (order by V.values->>OF.id) as seq,
							true as col_exists
						FROM
							task_custom_user_field_values V,
							get_order_in_board_column_field OF,
							get_current_board_column CC,
							get_board_column_field CF
						WHERE
							V.uid = $1
						AND
							V.values->>CF.id = CC.curr_col
						AND
							V.task_id <> $2
						UNION
						SELECT $2, CC.curr_col, 0, true FROM get_current_board_column CC LIMIT 1
					),
					final_result AS (
						SELECT task_id, col, seq, col_exists FROM if_curr_col_not_exists
						UNION
						SELECT task_id, col, seq, col_exists FROM if_curr_col_exists
					)
					UPDATE
						task_custom_user_field_values
					SET
						values[CF.id] = to_jsonb(FR.col),
						values[OF.id] = to_jsonb(FR.seq)
					FROM
						task_custom_user_field_values V,
						get_board_column_field CF,
						get_order_in_board_column_field OF,
						final_result FR,
						get_board_column_still_exists COL_EXISTS
					WHERE
						FR.col_exists = COL_EXISTS.curr_col_exists
					AND
						V.uid = $1
					AND
						V.task_id = FR.task_id
				`, uid, req.TaskID)
				if err != nil {
					fmt.Println("returning user: ", err)
				}
			} else {
				database.DB.Exec(`
					insert into task_custom_user_field_values
					select
						$1 as uid,
						$2 as task_id,
						jsonb_object_agg(id, details['default']) as values
					from task_custom_user_fields where uid = $1
				`, userID, req.TaskID)

				_, err = database.DB.Exec(`
					WITH get_board_column_field AS (
						SELECT 
							id::text,
							details->>'default' as default_v
						FROM
							task_custom_user_fields
						WHERE 
							uid = $1
						AND
							field_type = 'board_column'
					),
					get_order_in_board_column_field AS (
						SELECT
							id::text
						FROM
							task_custom_user_fields
						WHERE 
							uid = $1
						AND
							field_type = 'order_in_board_column'
					),
					final_result AS (
						SELECT
							V.task_id,
							row_number() OVER (order by V.values->>OF.id) as seq
						FROM
							task_custom_user_field_values V,
							get_order_in_board_column_field OF,
							get_board_column_field CF
						WHERE
							V.uid = $1
						AND
							V.task_id <> $2
						AND
							V.values->>CF.id = CF.default_v
					)
					UPDATE
						task_custom_user_field_values
					SET
						values[OF.id] = to_jsonb(FR.seq)
					FROM
						task_custom_user_field_values V,
						get_order_in_board_column_field OF,
						final_result FR
					WHERE
						V.uid = $1
					AND
						V.task_id = FR.task_id
				`, uid, req.TaskID)
				if err != nil {
					fmt.Println("new user: ", err)
				}
			}
		}
	}

	wsMessage := websocket.Response{
		Type: "tasks_edit-main-field",
		Payload: map[string]interface{}{
			"taskID":     req.TaskID,
			"field":      req.Field,
			"value":      req.Value,
			"taskRecord": req.TaskRecord,
		},
	}

	allUsers := make([]string, 0)
	allUsers = append(allUsers, userIDs...)
	allUsers = append(allUsers, existingUserIDs...)

	data.WsRequestIDs = append(data.WsRequestIDs, websocket.SaveWsMessageInDB(&wsMessage, common.UniqueStringFromSlice(&allUsers)))

	if req.Field == "name" || req.Field == "approval" || req.Field == "owner" {
		parentChildTaskUserIDs := *getParentChildTasksUserIDs(req.TaskID)
		if len(parentChildTaskUserIDs) != 0 {
			wsMessage := websocket.Response{
				Type: "tasks_parent-child-task",
				Payload: map[string]interface{}{
					"taskID": req.TaskID,
					"field":  req.Field,
					"value":  req.Value,
				},
			}
			data.WsRequestIDs = append(data.WsRequestIDs, websocket.SaveWsMessageInDB(&wsMessage, &parentChildTaskUserIDs))
		}
	}

	data.Success = true
	json.NewEncoder(w).Encode(data)
}

func approvalOrNil(r *TaskRecord) interface{} {
	if r.Action == "approval" {
		return r.Approval
	} else {
		return nil
	}
}

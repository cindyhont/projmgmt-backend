package tasks

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/instantcomm"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/julienschmidt/httprouter"
)

func taskMovedInBoard(
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
		Success     bool   `json:"success"`
		WsRequestID string `json:"wsid"`
	}{
		Success:     false,
		WsRequestID: "",
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	var req struct {
		TaskID         string `json:"taskID"`
		NewColumnID    string `json:"newColumnID"`
		NewIdxInColumn int    `json:"newIdxInColumn"`
	}

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	_, err = database.DB.Exec(`
		with board_column_field_sq as (
			select 
				id::text 
			from 
				task_custom_user_fields 
			where 
				uid = $1 
			and 
				field_type='board_column' 
			limit 1
		),
		order_in_board_column_field_sq as (
			select 
				id::text 
			from 
				task_custom_user_fields 
			where 
				uid = $1 
			and 
				field_type='order_in_board_column' 
			limit 1
		),
		old_task_column_sq as (
			select 
				V.values->>CF.id as id, 
				V.values->>CF.id = $3 as new_column_is_same
			from 
				task_custom_user_field_values V, 
				board_column_field_sq CF
			where 
				V.task_id = $2
		),
		old_column_temp_result as (
			select 
				V.task_id,
				V.values->>CF.id as col_id,
				row_number() over (order by V.values->>OF.id) - 1 as temp_seq
			from
				task_custom_user_field_values V,
				order_in_board_column_field_sq OF,
				board_column_field_sq CF,
				old_task_column_sq oldTC
			where
				V.values->>CF.id = oldTC.id
			and 
				V.values ? OF.id
			and
				V.task_id <> $2
			order by 
				V.values->>OF.id
		),
		same_column_final_result as (
			select 
				task_id, 
				col_id,
				temp_seq + case when temp_seq < $4 then 0 else 1 end as seq,
				true as same_column
			from 
			  	old_column_temp_result
			union
			select 
				$2, 
				V.values->>CF.id, 
				$4,
				true
			from 
				task_custom_user_field_values V,
				board_column_field_sq CF,
				old_task_column_sq oldTC
			where
				V.values->>CF.id = oldTC.id
		),
		new_column_temp_result as (
			select 
				V.task_id,
				$3 as col_id,
				row_number() over (order by V.values->>OF.id) - 1 as temp_seq
			from 
				task_custom_user_field_values V,
				order_in_board_column_field_sq OF,
				board_column_field_sq CF
			where
			  	V.values->>CF.id = $3
			and
			  	V.uid = $1
			and 
			  	V.values ? OF.id
			order by 
			  	V.values->>OF.id
		),
		new_column_final_result as (
			select 
				task_id,
				col_id,
				temp_seq + case when temp_seq < $4 then 0 else 1 end as seq,
				false as same_column
			from 
			  	new_column_temp_result
			union
			select 
				$2, 
				$3, 
				$4,
				false
			union
			select 
				task_id,
				col_id,
				temp_seq,
				false
			from 
			  	old_column_temp_result
		),
		final_result as (
			select 
				task_id,
				col_id,
				seq,
				same_column
			from 
				same_column_final_result
			union
			select 
				task_id,
				col_id,
				seq,
				same_column
			from
				new_column_final_result
			order by
				same_column
		)
		update
			task_custom_user_field_values FV
		set
			values[CF.id] = to_jsonb(FR.col_id),
			values[OF.id] = to_jsonb(FR.seq)
		from
			board_column_field_sq CF,
			order_in_board_column_field_sq OF,
			final_result FR,
			old_task_column_sq oldTC
		where
			FR.same_column = oldTC.new_column_is_same
		and
			uid = $1
		and
			FV.task_id = FR.task_id
	`, uid, req.TaskID, req.NewColumnID, req.NewIdxInColumn)
	if err != nil {
		fmt.Println(err)
	}

	wsMessage := instantcomm.Response{
		Type: "tasks_task-moved-in-board",
		Payload: map[string]interface{}{
			"taskID":         req.TaskID,
			"newColumnID":    req.NewColumnID,
			"newIdxInColumn": req.NewIdxInColumn,
			"active":         false,
		},
	}

	data.WsRequestID = instantcomm.SaveWsMessageInDB(&wsMessage, &[]string{uid})
	data.Success = true

	json.NewEncoder(w).Encode(data)
}

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

func deleteBoardColumn(
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

	var req struct {
		ColumnIdToDelete string `json:"boardColumnIdToDelete"`
		Action           string `json:"action"`
		NewDefault       string `json:"newDefault"`
	}

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	var boardColumnFieldID string
	database.DB.QueryRow("SELECT id FROM task_custom_user_fields WHERE uid=$1 AND field_type='board_column' limit 1", uid).Scan(&boardColumnFieldID)

	_, err = database.DB.Exec(`
		WITH get_column_task_count AS (
			select 
				count(1) as count
			from 
				task_custom_user_field_values
			where 
				values[$1]=$2
		),
		get_order_in_board_column_field_id AS (
			select 
				id::text
			from 
				task_custom_user_fields
			where
				uid = $4
			and
				field_type = 'order_in_board_column'
		),
		get_old_board_column_seq AS (
			select
				V.task_id,
				row_number() over (order by V.values->>OF.id) - 1 + COUNT.count as seq
			from 
				task_custom_user_field_values V,
				get_order_in_board_column_field_id OF,
				get_column_task_count COUNT
			where
				V.values[$1]=$3
			and
				V.uid = $4
			order by V.values->>OF.id
		)
		UPDATE 
			task_custom_user_field_values
		SET 
			values[$1]=$2,
			values[OF.id]=to_jsonb(SEQ.seq)
		FROM
			get_old_board_column_seq SEQ,
			get_order_in_board_column_field_id OF
		where 
			values[$1]=$3
		and
			uid = $4
		`,
		boardColumnFieldID,
		addQmark(req.Action),
		addQmark(req.ColumnIdToDelete),
		uid,
	)
	if err != nil {
		fmt.Println("a:", err)
	}

	if len(req.NewDefault) != 0 {
		database.DB.Exec("UPDATE task_custom_user_fields SET details['default']=$1 WHERE id=$2", addQmark(req.NewDefault), boardColumnFieldID)
	}

	_, err = database.DB.Exec(`
		with sqa as (
			select
				(jsonb_populate_record(null::board_column,jsonb_array_elements(details->'options'))).*
			from
				task_custom_user_fields
			where
				id=$1
		),
		sqb as (
			select
				id,
			name,
				row_number() over (order by "order" asc) -1 as order
			from
				sqa
			where
				id<>$2
		),
		sqc as (
			select
				jsonb_agg(jsonb_build_object('id',id,'name',name,'order',"order")) as _result
			from
				sqb
		)
		update
			task_custom_user_fields
		set
			details['options'] = _result
		from
			sqc
		where
			id=$1
	`, boardColumnFieldID, req.ColumnIdToDelete)
	if err != nil {
		fmt.Println("b:", err)
	}

	wsMessage := instantcomm.Response{
		Type: "tasks_delete-board-column",
		Payload: map[string]interface{}{
			"boardColumnIdToDelete": req.ColumnIdToDelete,
			"action":                req.Action,
			"newDefault":            req.NewDefault,
		},
	}
	data.WsRequestIDs = append(data.WsRequestIDs, instantcomm.SaveWsMessageInDB(&wsMessage, &[]string{uid}))

	data.Success = true
	json.NewEncoder(w).Encode(data)
}

func addQmark(e string) string {
	return fmt.Sprintf("%q", e)
}

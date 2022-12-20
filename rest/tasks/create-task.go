package tasks

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/instantcomm"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/cindyhont/projmgmt-backend/rest/common"
	"github.com/cindyhont/projmgmt-backend/rest/googleservice"
	"github.com/julienschmidt/httprouter"
	"github.com/lib/pq"
)

func addTask(
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

	type parent struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Owner    string `json:"owner"`
		Approval int    `json:"approval"`
	}

	data := struct {
		Success      bool     `json:"success"`
		WsRequestIDs []string `json:"wsids"`
		Parents      []string `json:"parents"`
		Parent       parent   `json:"parent,omitempty"`
	}{
		Success:      false,
		WsRequestIDs: make([]string, 0),
		Parents:      make([]string, 0),
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	var req struct {
		Task          Task                   `json:"task"`
		ExtraFieldObj map[string]interface{} `json:"extraFieldObj"`
		PublicFileIDs []string               `json:"publicFileIDs"`
		Files         []googleservice.File   `json:"files"`
	}

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	if len(req.Files) != 0 {
		googleservice.AddFiles(&req.Files)
	}

	var boardColumnFieldID string //orderInBoardColumnFieldID
	database.DB.QueryRow("SELECT id FROM task_custom_user_fields WHERE uid = $1 AND field_type = 'board_column'", uid).Scan(&boardColumnFieldID)
	// database.DB.QueryRow("SELECT id FROM task_custom_user_fields WHERE uid = $1 AND field_type = 'order_in_board_column'", uid).Scan(&orderInBoardColumnFieldID)

	// move all items in default board column lower, so that the new task goes to top of column
	boardColumnID := req.ExtraFieldObj[boardColumnFieldID].(string)
	_, err = database.DB.Exec(`
		with get_order_in_board_column as (
			select
				id::text
			from 
				task_custom_user_fields
			where
				uid = $1
			and 
				field_type = 'order_in_board_column'
		),
		get_column_seq as (
			select
				V.task_id,
				row_number() over (order by V.values->>CF.id) as seq
			from
				task_custom_user_field_values V,
				get_order_in_board_column CF
			where
				V.uid = $1
			and
				V.values[$2] = $3
		)
		update
			task_custom_user_field_values
		set
			values[CF.id] = to_jsonb(SEQ.seq)
		from
			get_order_in_board_column CF,
			get_column_seq SEQ,
			task_custom_user_field_values V
		where
			SEQ.task_id = V.task_id
		and
			V.uid = $1
		and
			V.values[$2] = $3
	`, uid, boardColumnFieldID, addQmark(boardColumnID))
	if err != nil {
		fmt.Println(err)
		json.NewEncoder(w).Encode(data)
	}

	parentID := cleanParent(&req.Task.Parents)

	// insert task
	err = database.DB.QueryRow(`
		with task_possible_parents as (
			select
				parents || replace($2,'-','')::ltree as lt
			from
				tasks
			where
				id = case when $1='' then uuid_nil() else $1::uuid end
			and
				deleted = false
		)
		insert into tasks (
			parents,
			id,
			name,
			description,
			create_dt,
			start_dt,
			deadline_dt,
			is_group_task,
			owner,
			supervisors,
			participants,
			viewers,
			hourly_rate,
			track_time,
			files,
			public_file_ids,
			tsv,
			assignee,
			approval
		)
		select
			coalesce((select lt from task_possible_parents),replace($2,'-','')::ltree),
			$2::uuid,
			$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,to_tsvector($17),$18,$19
		returning
			string_to_array(ltree2text(parents),'.')::uuid[]
	`,
		parentID,
		req.Task.ID,
		req.Task.Name,
		stringOrNil(req.Task.Description),
		time.UnixMilli(req.Task.CreateDT),
		timeOrNil(req.Task.StartDT),
		timeOrNil(req.Task.DeadlineDT),
		req.Task.IsGroupTask,
		req.Task.Owner,
		arrayOrNil(&req.Task.Supervisors),
		arrayOrNil(&req.Task.Participants),
		arrayOrNil(&req.Task.Viewers),
		req.Task.HourlyRate,
		req.Task.TrackTime,
		arrayOrNil(&req.Task.FileIDs),
		arrayOrNil(&req.PublicFileIDs),
		req.Task.Name,
		req.Task.Assignee,
		req.Task.Approval,
	).Scan(pq.Array(&data.Parents))
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	var taskMap map[string]interface{}
	taskMapBytes, _ := json.Marshal(req.Task)
	json.Unmarshal(taskMapBytes, &taskMap)

	taskMap["parents"] = data.Parents

	// if task has parent, get parent details
	if parentID != "" {
		database.DB.QueryRow("SELECT id, name, owner, approval FROM tasks WHERE id = $1", parentID).Scan(
			&data.Parent.ID,
			&data.Parent.Name,
			&data.Parent.Owner,
			&data.Parent.Approval,
		)
	}

	extraFieldInBytes, _ := json.Marshal(req.ExtraFieldObj)

	_, err = database.DB.Exec(`
		INSERT INTO task_custom_user_field_values (uid, task_id, values)
		VALUES ($1,$2,$3)
	`, uid, req.Task.ID, string(extraFieldInBytes))
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	data.Success = true

	_userIDs := []string{req.Task.Owner, uid}
	_userIDs = append(_userIDs, req.Task.Supervisors...)
	_userIDs = append(_userIDs, req.Task.Participants...)
	_userIDs = append(_userIDs, req.Task.Viewers...)

	userIDs := *common.UniqueStringFromSlice(&_userIDs)

	for _, userID := range userIDs {
		if userID == uid {
			wsMessage := instantcomm.Response{
				Type: "tasks_add-task",
				Payload: map[string]interface{}{
					"task":          taskMap,
					"extraFieldObj": req.ExtraFieldObj,
					"files":         req.Files,
				},
			}
			data.WsRequestIDs = append(data.WsRequestIDs, instantcomm.SaveWsMessageInDB(&wsMessage, &[]string{userID}))
		} else {
			_, err = database.DB.Exec(`
				with get_order_in_board_column as (
					select
						id::text
					from 
						task_custom_user_fields
					where
						uid = $1
					and 
						field_type = 'order_in_board_column'
				),
				get_board_column_field as (
					select
						id::text,
						details->>'default' as default_v
					from 
						task_custom_user_fields
					where
						uid = $1
					and 
						field_type = 'board_column'
				),
				result_seq as (
					select 
						V.task_id,
						row_number() over (order by V.values->>OF.id) as seq
					from
						task_custom_user_fields V,
						get_order_in_board_column OF,
						get_board_column_field CF
					where
						V.uid = $1
					and
						V.values->>CF.id = CF.default_v
				)
				update
					task_custom_user_fields V
				set
					V.values->>OF.id = SEQ.seq
				from 
					get_order_in_board_column OF,
					result_seq SEQ,
					get_board_column_field CF
				where
					V.uid = $1
				and
					V.task_id = SEQ.task_id
				and
					V.values->>CF.id = CF.default_v
			`, userID)
			if err != nil {
				fmt.Println(err)
			}

			var s string
			database.DB.QueryRow(`
				insert into task_custom_user_field_values
				select 
					$1 as uid, 
					$2 as task_id, 
					jsonb_object_agg(id, details['default']) as values 
				from task_custom_user_fields where uid = $1
				returning values
			`, userID, req.Task.ID).Scan(&s)

			var extraFieldObj map[string]interface{}
			json.Unmarshal([]byte(s), &extraFieldObj)

			wsMessage := instantcomm.Response{
				Type: "tasks_add-task",
				Payload: map[string]interface{}{
					"task":          taskMap,
					"extraFieldObj": extraFieldObj,
					"files":         req.Files,
				},
			}
			data.WsRequestIDs = append(data.WsRequestIDs, instantcomm.SaveWsMessageInDB(&wsMessage, &[]string{userID}))
		}
	}

	parentChildTaskUserIDs := *getParentChildTasksUserIDs(req.Task.ID)
	if len(parentChildTaskUserIDs) != 0 {
		wsMessage := instantcomm.Response{
			Type:    "tasks_new-parent-child-task",
			Payload: taskMap,
		}
		data.WsRequestIDs = append(data.WsRequestIDs, instantcomm.SaveWsMessageInDB(&wsMessage, &parentChildTaskUserIDs))
	}

	json.NewEncoder(w).Encode(data)
}

func cleanParent(arr *[]string) string {
	if len(*arr) > 1 {
		return (*arr)[0]
	} else {
		return ""
	}
}

func stringOrNil(s string) interface{} {
	if len(s) != 0 {
		return s
	} else {
		return nil
	}
}

func arrayOrNil(a *[]string) interface{} {
	if len(*a) != 0 {
		return pq.Array(*a)
	} else {
		return nil
	}
}

func timeOrNil(a int64) interface{} {
	if a != 0 {
		return time.UnixMilli(a)
	} else {
		return nil
	}
}

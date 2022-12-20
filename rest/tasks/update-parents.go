package tasks

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/instantcomm"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/cindyhont/projmgmt-backend/rest/common"
	"github.com/julienschmidt/httprouter"
	"github.com/lib/pq"
)

func updateParents(
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
		Success     bool     `json:"success"`
		WsRequestID string   `json:"wsid"`
		Parents     []string `json:"parents"`
		Parent      parent   `json:"parent,omitempty"`
	}{
		Success:     false,
		WsRequestID: "",
		Parents:     make([]string, 0),
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	var req struct {
		TaskID string `json:"taskID"`
		Parent string `json:"parent"`
	}

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	userIDs := *getParentChildTasksUserIDs(req.TaskID)

	_, err = database.DB.Exec(`
		with original_ltree as (
			select parents,nlevel(parents) as lvl from tasks where id = $1
		),
		source_list as (
			select
				id,
				nlevel(T.parents) as lvl,
				index(T.parents,OL.parents)+OL.lvl as startidx
			from 
				tasks T,
				original_ltree OL
			where
				T.parents ~ concat('*.', OL.parents, '.*')::lquery
		),
		new_value as (
			select coalesce(
				(
					select 
						parents || replace($1::text,'-','')::ltree
					from 
						tasks 
					where 
						id = case when $2='' then uuid_nil() else $2::uuid end
				),
				replace($2,'-','')::ltree
			) as lt
		)
		update
			tasks
		set 
			parents = case when lvl = startidx then NV.lt else NV.lt || subpath(parents,startidx) end
		from 
			source_list SRC,
			new_value NV
		where
			SRC.id = tasks.id
	`, req.TaskID, req.Parent)
	if err != nil {
		fmt.Println(err)
		json.NewEncoder(w).Encode(data)
		return
	}

	err = database.DB.QueryRow("select string_to_array(ltree2text(parents),'.')::uuid[] from tasks where id = $1", req.TaskID).Scan(pq.Array(&data.Parents))
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	if req.Parent != "" {
		database.DB.QueryRow("SELECT id, name, owner, approval FROM tasks WHERE id = $1", req.Parent).Scan(
			&data.Parent.ID,
			&data.Parent.Name,
			&data.Parent.Owner,
			&data.Parent.Approval,
		)
	}

	userIDs = append(userIDs, *getParentChildTasksUserIDs(req.TaskID)...)
	userIDs = append(userIDs, *getTaskUserIDs(req.TaskID)...)

	wsMessage := instantcomm.Response{
		Type: "tasks_update-parents",
		Payload: map[string]interface{}{
			"taskID":  req.TaskID,
			"parents": data.Parents,
		},
	}

	data.WsRequestID = instantcomm.SaveWsMessageInDB(&wsMessage, common.UniqueStringFromSlice(&userIDs))
	data.Success = true
	json.NewEncoder(w).Encode(data)
}

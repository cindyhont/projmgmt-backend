package tasks

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/instantcomm"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/julienschmidt/httprouter"
)

func editTaskExtraField(
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

	var req editFieldRequest

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	_, err = database.DB.Exec(
		"UPDATE task_custom_user_field_values SET values[$1] = $2 WHERE uid=$3 and task_id=$4",
		req.Field,
		req.Value,
		uid,
		req.TaskID,
	)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	wsMessage := instantcomm.Response{
		Type: "tasks_edit-extra-field",
		Payload: map[string]interface{}{
			"taskID": req.TaskID,
			"field":  req.Field,
			"value":  req.Value,
		},
	}
	data.WsRequestID = instantcomm.SaveWsMessageInDB(&wsMessage, &[]string{uid})
	data.Success = true
	json.NewEncoder(w).Encode(data)
}

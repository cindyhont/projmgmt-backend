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

func deleteCustomField(
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
		FieldID string `json:"fieldID"`
	}

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	database.DB.Exec("DELETE FROM task_custom_user_fields WHERE id = $1", req.FieldID)
	database.DB.Exec("UPDATE task_custom_user_field_values SET values = values - $1 WHERE uid = $2", req.FieldID, uid)

	wsMessage := instantcomm.Response{
		Type: "tasks_delete-custom-field",
		Payload: map[string]interface{}{
			"fieldID": req.FieldID,
		},
	}
	data.WsRequestID = instantcomm.SaveWsMessageInDB(&wsMessage, &[]string{uid})
	data.Success = true
	json.NewEncoder(w).Encode(data)
}

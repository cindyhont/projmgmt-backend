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

func editCustomFieldConfig(
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
		ID        string                 `json:"id"`
		FieldName string                 `json:"fieldName"`
		Details   map[string]interface{} `json:"details"`
	}

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	bytesDetails, err := json.Marshal(req.Details)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	_, err = database.DB.Exec("UPDATE task_custom_user_fields SET field_name=$1, details=$2 WHERE id=$3", req.FieldName, string(bytesDetails), req.ID)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	wsMessage := instantcomm.Response{
		Type: "tasks_edit-custom-field-config",
		Payload: map[string]interface{}{
			"id":        req.ID,
			"fieldName": req.FieldName,
			"details":   req.Details,
		},
	}
	data.WsRequestID = instantcomm.SaveWsMessageInDB(&wsMessage, &[]string{uid})
	data.Success = true
	json.NewEncoder(w).Encode(data)
}

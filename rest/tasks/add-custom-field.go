package tasks

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/instantcomm"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/julienschmidt/httprouter"
)

func addCustomField(
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
		FieldType string                 `json:"fieldType"`
		Details   map[string]interface{} `json:"details"`
	}

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	database.DB.Exec(
		"UPDATE task_custom_user_field_values SET values[$1] = $2 WHERE uid=$3",
		req.ID,
		handleDefaultValueType(req.Details["default"]),
		uid,
	)

	bytesDetails, _ := json.Marshal(req.Details)
	database.DB.Exec(
		"INSERT INTO task_custom_user_fields (id,uid,field_type,details,field_name) VALUES ($1,$2,$3,$4,$5)",
		req.ID,
		uid,
		req.FieldType,
		string(bytesDetails),
		req.FieldName,
	)

	var wsPayload map[string]interface{}
	json.Unmarshal(body, &wsPayload)

	wsMessage := instantcomm.Response{
		Type:    "tasks_add-custom-field",
		Payload: wsPayload,
	}
	data.WsRequestID = instantcomm.SaveWsMessageInDB(&wsMessage, &[]string{uid})
	data.Success = true
	json.NewEncoder(w).Encode(data)
}

func handleDefaultValueType(e interface{}) interface{} {
	switch v := reflect.ValueOf(e); v.Kind() {
	case reflect.String:
		return fmt.Sprintf("%q", e)
	case reflect.Array, reflect.Slice, reflect.Map:
		bytes, _ := json.Marshal(e)
		return string(bytes)
	default:
		return fmt.Sprintf("%v", e)
	}
}

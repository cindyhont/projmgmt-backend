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

func deleteTask(
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
		TaskID string `json:"taskID"`
	}

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	existingUserIDs := getTaskUserIDs(req.TaskID)

	database.DB.Exec(`
		UPDATE
			tasks
		SET
			deleted = TRUE,
			delete_dt = now()
		WHERE
			id = $1
	`, req.TaskID)

	wsMessage := instantcomm.Response{
		Type: "tasks_delete-task",
		Payload: map[string]interface{}{
			"taskID": req.TaskID,
		},
	}

	data.WsRequestID = instantcomm.SaveWsMessageInDB(&wsMessage, existingUserIDs)
	data.Success = true
	json.NewEncoder(w).Encode(data)
}

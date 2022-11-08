package tasks

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/cindyhont/projmgmt-backend/websocket"
	"github.com/julienschmidt/httprouter"
)

func deleteComment(
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
		ID       string `json:"id"`
		TaskID   string `json:"taskID"`
		DateTime int64  `json:"time"`
	}

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	database.DB.Exec("UPDATE task_comments SET deleted = TRUE, delete_time = $1 WHERE id = $2", time.UnixMilli(req.DateTime), req.ID)

	wsMessage := websocket.Response{
		Type: "tasks_delete-comment",
		Payload: map[string]interface{}{
			"id":   req.ID,
			"time": req.DateTime,
		},
	}
	data.WsRequestID = websocket.SaveWsMessageInDB(&wsMessage, getTaskUserIDs(req.TaskID))
	data.Success = true
	json.NewEncoder(w).Encode(data)
}

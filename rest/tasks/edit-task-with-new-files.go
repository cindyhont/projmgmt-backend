package tasks

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/cindyhont/projmgmt-backend/rest/googleservice"
	"github.com/cindyhont/projmgmt-backend/websocket"
	"github.com/julienschmidt/httprouter"
	"github.com/lib/pq"
)

func editTaskWithNewFiles(
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
		LongTextMap   map[string]string    `json:"longTextMap"`
		PublicFileIDs []string             `json:"publicFileIDs"`
		PrivateFiles  []googleservice.File `json:"privateFiles"`
		TaskID        string               `json:"taskID"`
	}

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	for k, v := range req.LongTextMap {
		if k == "description" {
			database.DB.Exec("UPDATE tasks SET description = $1 WHERE id = $2", v, req.TaskID)
		} else {
			database.DB.Exec(
				"UPDATE task_custom_user_field_values SET values[$1] = $2 WHERE uid = $3 AND task_id = $4",
				k,
				fmt.Sprintf("%q", v),
				uid,
				req.TaskID,
			)
		}
	}

	if len(req.PrivateFiles) != 0 {
		fileIDs := make([]string, 0)
		for _, file := range req.PrivateFiles {
			fileIDs = append(fileIDs, file.ID)
		}
		database.DB.Exec(
			"UPDATE tasks SET files = CASE WHEN NULL THEN $1 ELSE array_cat(files,$1) END WHERE id = $2",
			pq.Array(fileIDs),
			req.TaskID,
		)
		googleservice.AddFiles(&req.PrivateFiles)
	}

	if len(req.PublicFileIDs) != 0 {
		database.DB.Exec(
			"UPDATE tasks SET public_file_ids = CASE WHEN NULL THEN $1 ELSE array_cat(public_file_ids,$1) END WHERE id = $2",
			pq.Array(req.PublicFileIDs),
			req.TaskID,
		)
	}

	wsMessage := websocket.Response{
		Type: "tasks_edit-task-with-new-files",
		Payload: map[string]interface{}{
			"taskID":       req.TaskID,
			"longTextMap":  req.LongTextMap,
			"privateFiles": req.PrivateFiles,
			"uid":          uid,
		},
	}
	data.WsRequestID = websocket.SaveWsMessageInDB(&wsMessage, getTaskUserIDs(req.TaskID))
	data.Success = true
	json.NewEncoder(w).Encode(data)
}

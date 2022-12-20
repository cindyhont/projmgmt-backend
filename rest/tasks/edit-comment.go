package tasks

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/instantcomm"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/cindyhont/projmgmt-backend/rest/googleservice"
	"github.com/julienschmidt/httprouter"
	"github.com/lib/pq"
)

func editComment(
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
		CommentID        string               `json:"id"`
		TaskID           string               `json:"taskID"`
		EditDateTime     int64                `json:"editDT"`
		Content          string               `json:"content"`
		NewPublicFileIDs []string             `json:"newPublicFileIDs"`
		PrivateFileIDs   []string             `json:"privateFileIDs"`
		NewFiles         []googleservice.File `json:"newFiles"`
	}

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	if len(req.NewFiles) != 0 {
		googleservice.AddFiles(&req.NewFiles)
	}

	if len(req.NewPublicFileIDs) != 0 {
		database.DB.Exec(
			"UPDATE task_comments SET public_file_ids = CASE WHEN NULL THEN $1 ELSE array_cat(public_file_ids,$1) END WHERE id = $2",
			pq.Array(req.NewPublicFileIDs),
			req.CommentID,
		)
	}

	database.DB.Exec(
		"UPDATE task_comments SET content=$1, edit_time=$2, files=$3 WHERE id=$4",
		req.Content,
		time.UnixMilli(req.EditDateTime),
		arrayOrNil(&req.PrivateFileIDs),
		req.CommentID,
	)

	wsMessage := instantcomm.Response{
		Type: "tasks_edit-comment",
		Payload: map[string]interface{}{
			"id":             req.CommentID,
			"editDt":         req.EditDateTime,
			"content":        req.Content,
			"privateFileIDs": req.PrivateFileIDs,
			"newFiles":       req.NewFiles,
		},
	}

	data.WsRequestID = instantcomm.SaveWsMessageInDB(&wsMessage, getTaskUserIDs(req.TaskID))
	data.Success = true
	json.NewEncoder(w).Encode(data)
}

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
)

func addComment(
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
		Comment       Comment              `json:"comment"`
		Files         []googleservice.File `json:"files"`
		PublicFileIDs []string             `json:"publicFileIDs"`
	}

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	if len(req.Files) != 0 {
		googleservice.AddFiles(&req.Files)
	}

	database.DB.Exec(
		`
			INSERT INTO
				task_comments
				(
					id,
					task_id,
					sender,
					content,
					dt,
					files,
					reply_comment_id,
					reply_comment,
					reply_comment_sender,
					public_file_ids
				)
			VALUES
				($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		`,
		req.Comment.ID,
		req.Comment.TaskID,
		req.Comment.Sender,
		req.Comment.Content,
		time.UnixMilli(req.Comment.DateTime),
		arrayOrNil(&req.Comment.FileIDs),
		stringOrNil(req.Comment.ReplyMsgID),
		stringOrNil(req.Comment.ReplyMsg),
		stringOrNil(req.Comment.ReplyMsgSender),
		arrayOrNil(&req.PublicFileIDs),
	)

	wsMessage := instantcomm.Response{
		Type: "tasks_add-comment",
		Payload: map[string]interface{}{
			"comment": req.Comment,
			"files":   req.Files,
		},
	}
	data.WsRequestID = instantcomm.SaveWsMessageInDB(&wsMessage, getTaskUserIDs(req.Comment.TaskID))
	data.Success = true
	json.NewEncoder(w).Encode(data)
}

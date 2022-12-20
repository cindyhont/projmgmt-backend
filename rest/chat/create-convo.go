package chat

import (
	"encoding/json"
	"fmt"
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

type convo struct {
	ConvoID    string   `json:"id"`
	Content    string   `json:"content"`
	ReplyMsgID string   `json:"replyMsgID"`
	FileIDs    []string `json:"fileIDs"`
}

func createConvo(
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
		RoomID string               `json:"rid"`
		Convo  convo                `json:"convo"`
		Files  []googleservice.File `json:"files"`
	}

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	if len(req.Files) != 0 {
		err = googleservice.AddFiles(&req.Files)
		if err != nil {
			json.NewEncoder(w).Encode(data)
			return
		}
	}

	data.Success = createConvoInternal(&req.Convo, uid, req.RoomID)
	if !data.Success {
		json.NewEncoder(w).Encode(data)
		return
	}

	userIDs := getChatRoomUserIDs(req.RoomID)

	var replyMsgContent, replyMsgSender string
	if req.Convo.ReplyMsgID != "" {
		if err = database.DB.QueryRow(`
			SELECT content, sender_id FROM chat_messages WHERE id = $1
		`, req.Convo.ReplyMsgID).Scan(
			&replyMsgContent,
			&replyMsgSender,
		); err != nil {
			replyMsgContent = ""
			replyMsgSender = ""
		}
	}

	wsMessage := instantcomm.Response{
		Type: "chat_new-convo",
		Payload: map[string]interface{}{
			"id":             req.Convo.ConvoID,
			"sender":         uid,
			"roomID":         req.RoomID,
			"content":        req.Convo.Content,
			"replyMsgID":     req.Convo.ReplyMsgID,
			"replyMsg":       replyMsgContent,
			"replyMsgSender": replyMsgSender,
			"fileIDs":        req.Convo.FileIDs,
			"dt":             time.Now().UnixMilli(),
			"files":          req.Files,
		},
	}

	data.WsRequestID = instantcomm.SaveWsMessageInDB(&wsMessage, userIDs)

	json.NewEncoder(w).Encode(data)
}

func createConvoInternal(c *convo, uid string, roomID string) bool {
	baseQuery := `
		INSERT INTO
			chat_messages (
				id,
				rid,
				content,
				sender_id%s
			)
		VALUES ($1, $2, $3, $4%s)
	`

	var variables []interface{}
	variables = append(variables, c.ConvoID)
	variables = append(variables, roomID)
	variables = append(variables, c.Content)
	variables = append(variables, uid)

	var variableStr, positionStr string
	pos := 4

	if len(c.FileIDs) != 0 {
		pos++
		variableStr += ", files"
		positionStr += fmt.Sprintf(", $%d", pos)
		variables = append(variables, pq.Array(c.FileIDs))
	}
	if c.ReplyMsgID != "" {
		pos++
		variableStr += ", reply_msg_id,reply_msg,reply_msg_sender"
		positionStr += fmt.Sprintf(`
			, $%d, 
			(select content from chat_messages where id = $%d),
			(select sender_id from chat_messages where id = $%d)
		`, pos, pos, pos)
		variables = append(variables, c.ReplyMsgID)
	}

	query := fmt.Sprintf(baseQuery, variableStr, positionStr)

	if _, err := database.DB.Exec(query, variables...); err != nil {
		fmt.Println(err)
		return false
	}

	return true
}

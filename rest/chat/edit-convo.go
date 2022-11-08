package chat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/cindyhont/projmgmt-backend/websocket"
	"github.com/julienschmidt/httprouter"
)

func editConvo(
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
		ConvoID string `json:"id"`
		Content string `json:"content"`
	}

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	if _, err = database.DB.Exec("UPDATE chat_messages SET content = $1, edit_time = NOW() WHERE id = $2", req.Content, req.ConvoID); err != nil {
		fmt.Println(err)
		json.NewEncoder(w).Encode(data)
		return
	}

	var roomid string
	database.DB.QueryRow("SELECT rid FROM chat_messages WHERE id = $1", req.ConvoID).Scan(&roomid)

	userIDs := getChatRoomUserIDs(roomid)

	wsMessage := websocket.Response{
		Type: "chat_edit-convo",
		Payload: map[string]interface{}{
			"roomid":  roomid,
			"content": req.Content,
			"convoid": req.ConvoID,
			"editdt":  time.Now().UnixMilli(),
		},
	}

	data.WsRequestID = websocket.SaveWsMessageInDB(&wsMessage, userIDs)

	data.Success = true
	json.NewEncoder(w).Encode(data)
}

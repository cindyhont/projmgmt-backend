package chat

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/cindyhont/projmgmt-backend/websocket"
	"github.com/julienschmidt/httprouter"
)

func updateLastSeen(
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

	rid := p.ByName("room-id")

	data := struct {
		Success     bool   `json:"success"`
		WsRequestID string `json:"wsid"`
	}{
		Success:     false,
		WsRequestID: "",
	}

	if _, err := database.DB.Exec("UPDATE chatrooms_users SET last_seen = default WHERE rid = $1 AND uid = $2", rid, uid); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	err := updateMarkAsReadInternal(rid, 0, uid)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	userIDs := getChatRoomUserIDs(rid)

	wsMessage := websocket.Response{
		Type: "chat_lastseen",
		Payload: map[string]interface{}{
			"uid":      uid,
			"roomid":   rid,
			"lastseen": time.Now().UnixMilli(),
		},
	}
	data.WsRequestID = websocket.SaveWsMessageInDB(&wsMessage, userIDs)

	data.Success = true
	json.NewEncoder(w).Encode(data)
}

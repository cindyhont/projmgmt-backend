package chat

import (
	"encoding/json"
	"net/http"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/instantcomm"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/julienschmidt/httprouter"
)

func createRoomNoConvo(
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
		RoomID      string `json:"rid"`
		Success     bool   `json:"success"`
		WsRequestID string `json:"wsid"`
	}{
		RoomID:      "",
		Success:     false,
		WsRequestID: "",
	}

	roommateID := p.ByName("roommate-id")
	users := []string{roommateID, uid}
	data.RoomID = getRoomIdByUserIDsInternal(users)

	if data.RoomID == "" {
		database.DB.QueryRow("INSERT INTO chatrooms (id) VALUES (default) RETURNING id").Scan(&data.RoomID)
	}

	data.Success = bulkImportRoomUsers(users, "", data.RoomID)
	if !data.Success {
		json.NewEncoder(w).Encode(data)
		return
	}

	userIDs := []string{uid, roommateID}

	wsMessage := instantcomm.Response{
		Type: "chat_new-room-no-convo",
		Payload: map[string]interface{}{
			"id":    data.RoomID,
			"users": userIDs,
		},
	}

	data.WsRequestID = instantcomm.SaveWsMessageInDB(&wsMessage, &userIDs)

	json.NewEncoder(w).Encode(data)
}

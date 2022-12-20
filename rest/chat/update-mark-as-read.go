package chat

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/instantcomm"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/julienschmidt/httprouter"
)

func updateMarkAsRead(
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

	type updateMarkAsReadRequest struct {
		RoomID     string `json:"rid"`
		MarkAsRead int    `json:"markAsRead"`
	}

	var req updateMarkAsReadRequest

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	err = updateMarkAsReadInternal(req.RoomID, req.MarkAsRead, uid)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	wsMessage := instantcomm.Response{
		Type: "chat_markasread",
		Payload: map[string]interface{}{
			"roomid":     req.RoomID,
			"markasread": req.MarkAsRead,
		},
	}

	data.WsRequestID = instantcomm.SaveWsMessageInDB(&wsMessage, &[]string{uid})

	data.Success = true
	json.NewEncoder(w).Encode(data)
}

func updateMarkAsReadInternal(roomID string, markAsRead int, uid string) error {
	_, err := database.DB.Exec("UPDATE chatrooms_users SET mark_as_read = $1 WHERE rid = $2 AND uid = $3", markAsRead, roomID, uid)
	if err != nil {
		return err
	}
	return nil
}

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

func updatePinned(
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

	type setPinnedRequest struct {
		RoomID string `json:"rid"`
		Pinned bool   `json:"pinned"`
	}

	var req setPinnedRequest

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	_, err = database.DB.Exec("UPDATE chatrooms_users SET pinned = $1 WHERE rid = $2 AND uid = $3", req.Pinned, req.RoomID, uid)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	wsMessage := instantcomm.Response{
		Type: "chat_pinned",
		Payload: map[string]interface{}{
			"roomid": req.RoomID,
			"pinned": req.Pinned,
		},
	}

	data.WsRequestID = instantcomm.SaveWsMessageInDB(&wsMessage, &[]string{uid})
	data.Success = true
	json.NewEncoder(w).Encode(data)
}

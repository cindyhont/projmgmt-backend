package chat

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/julienschmidt/httprouter"
)

func fetchSpecificRooms(
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

	data := fetchRoomsDetailsResponse{
		Rooms: make([]Room, 0),
		Users: make([]model.UserDetails, 0),
		Files: make([]model.File, 0),
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	var req struct {
		RoomIDs []string `json:"roomIDs"`
	}

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	data = *getDetailsOfRooms(&req.RoomIDs, uid)
	json.NewEncoder(w).Encode(data)
}

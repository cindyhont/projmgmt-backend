package chat

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/julienschmidt/httprouter"
	"github.com/lib/pq"
)

type fetchRoomsDetailsResponse struct {
	Rooms []Room              `json:"rooms"`
	Users []model.UserDetails `json:"users"`
	Files []model.File        `json:"files"`
}

func fetchMoreRooms(
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

	roomIDs := *getIDs(uid, &req.RoomIDs)
	if len(roomIDs) == 0 {
		json.NewEncoder(w).Encode(data)
		return
	}

	data = *getDetailsOfRooms(&roomIDs, uid)

	json.NewEncoder(w).Encode(data)
}

func getDetailsOfRooms(roomIDs *[]string, uid string) *fetchRoomsDetailsResponse {
	data := fetchRoomsDetailsResponse{
		Rooms: make([]Room, 0),
		Users: make([]model.UserDetails, 0),
		Files: make([]model.File, 0),
	}

	fileIDs := make([]string, 0)
	userIDs := make([]string, 0)
	for _, roomID := range *roomIDs {
		room, _fileIDs, _possibleUserIDs := GetInitRoomInfo(roomID, uid)
		fileIDs = append(fileIDs, *_fileIDs...)
		userIDs = append(userIDs, room.User.IDs...)
		userIDs = append(userIDs, *_possibleUserIDs...)
		data.Rooms = append(data.Rooms, *room)
	}

	data.Users = *getMoreUserDetails(&userIDs)

	if len(fileIDs) != 0 {
		data.Files = *getMoreFileDetails(&fileIDs)
	}
	return &data
}

func getMoreFileDetails(fileIDs *[]string) *[]model.File {
	files := make([]model.File, 0)

	rows, err := database.DB.Query(`
		select
			id,
			name,
			size
		from
			files
		where
			id = any($1)
	`, pq.Array(*fileIDs))
	if err != nil {
		return &files
	}
	defer rows.Close()

	for rows.Next() {
		var f model.File
		rows.Scan(
			&f.ID,
			&f.Name,
			&f.Size,
		)
		f.Downloading = false
		f.Progress = 0
		f.Url = ""
		files = append(files, f)
	}
	return &files
}

func getMoreUserDetails(userIDs *[]string) *[]model.UserDetails {
	users := make([]model.UserDetails, 0)

	rows, err := database.DB.Query(`
		select
			id,
			first_name,
			last_name,
			coalesce(avatar,'') as avatar
		from
			user_details
		where
			id = any($1)
	`, pq.Array(userIDs))
	if err != nil {
		return &users
	}

	defer rows.Close()

	for rows.Next() {
		var u model.UserDetails
		rows.Scan(
			&u.ID,
			&u.FirstName,
			&u.LastName,
			&u.Avatar,
		)
		users = append(users, u)
	}
	return &users
}

func getIDs(uid string, roomIDsToSkip *[]string) *[]string {
	result := make([]string, 0)

	rows, err := database.DB.Query(`
		select 
			CU.rid
		from 
			chatrooms_users CU
		inner join
			chat_messages CM
		on 
			CU.rid = CM.rid
		where
			CU.uid = $1
		and 
			CU.in_users_list
		and 
			CU.rid <> all($2)
		group by
			CU.rid
		order by
			max(CM.dt) desc
		limit 10 offset 0
	`, uid, pq.Array(*roomIDsToSkip))
	if err != nil {
		return &result
	}

	defer rows.Close()

	for rows.Next() {
		var s string
		rows.Scan(&s)
		result = append(result, s)
	}
	return &result
}

package chat

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/instantcomm"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/julienschmidt/httprouter"
	"github.com/lib/pq"
)

type createRoomRequest struct {
	RoommateID string `json:"roommateID"`
	Convo      convo  `json:"convo"`
}

func createRoomWithFirstConvo(
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
		RoomID       string `json:"rid"`
		RoomSuccess  bool   `json:"roomSuccess"`
		ConvoSuccess bool   `json:"convoSuccess"`
		WsRequestID  string `json:"wsid"`
	}{
		RoomID:       "",
		RoomSuccess:  false,
		ConvoSuccess: false,
		WsRequestID:  "",
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	var req createRoomRequest

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	users := []string{req.RoommateID, uid}
	data.RoomID = getRoomIdByUserIDsInternal(users)

	if data.RoomID == "" {
		database.DB.QueryRow("INSERT INTO chatrooms (id) VALUES (default) RETURNING id").Scan(&data.RoomID)
	}

	chanUserImportSuccess := make(chan bool, 1)
	chanConvoImportSuccess := make(chan bool, 1)

	go bulkImportRoomUsersGoroutine(users, "", data.RoomID, chanUserImportSuccess)
	go importConvo(&req.Convo, uid, data.RoomID, chanConvoImportSuccess)

	data.RoomSuccess = <-chanUserImportSuccess
	data.ConvoSuccess = <-chanConvoImportSuccess

	if !data.RoomSuccess || !data.ConvoSuccess {
		json.NewEncoder(w).Encode(data)
		return
	}

	wsMessage := instantcomm.Response{
		Type: "chat_new-room-w-convo",
		Payload: map[string]interface{}{
			"roomID":  data.RoomID,
			"users":   users,
			"convoID": req.Convo.ConvoID,
			"content": req.Convo.Content,
			"sender":  uid,
			"dt":      time.Now().UnixMilli(),
		},
	}

	data.WsRequestID = instantcomm.SaveWsMessageInDB(&wsMessage, &users)

	json.NewEncoder(w).Encode(data)
}

func getRoomIdByUserIDsInternal(uids []string) string {

	var roomID string
	err := database.DB.QueryRow(`
		SELECT
			rid
		FROM
			(
				SELECT
					CU.rid,
					ARRAY_AGG(CU.uid) as uids,
					C.room_name AS room_name
				FROM
					chatrooms_users CU
				inner join
				  chatrooms C
				on
					CU.rid = C.id
				where 
					C.room_name is null
				GROUP BY
					rid
			) AS TEMP
		WHERE 
			uids <@ $1
		AND
			C.room_name IS NULL
	`, pq.Array(uids)).Scan(&roomID)
	if err != nil {
		return roomID
	}

	return roomID
}

func importConvo(c *convo, uid string, roomID string, channel chan<- bool) {
	channel <- createConvoInternal(c, uid, roomID)
}

func bulkImportRoomUsersGoroutine(uids []string, adminUID string, roomID string, channel chan<- bool) {
	channel <- bulkImportRoomUsers(uids, adminUID, roomID)
}

func bulkImportRoomUsers(uids []string, adminUID string, roomID string) bool {
	txn, err := database.DB.Begin()
	if err != nil {
		return false
	}

	var stmt *sql.Stmt
	if stmt, err = txn.Prepare(pq.CopyIn("chatrooms_users", "rid", "uid", "admin")); err != nil {
		return false
	}

	for _, uid := range uids {
		if _, err = stmt.Exec(roomID, uid, false); err != nil {
			return false
		}
	}

	if adminUID != "" {
		if _, err = stmt.Exec(roomID, adminUID, true); err != nil {
			return false
		}
	}

	if _, err = stmt.Exec(); err != nil {
		return false
	}

	if err = stmt.Close(); err != nil {
		return false
	}

	if err = txn.Commit(); err != nil {
		return false
	}

	return true
}

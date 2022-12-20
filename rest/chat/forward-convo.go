package chat

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/instantcomm"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/julienschmidt/httprouter"
	"github.com/lib/pq"
)

type forwardUser struct {
	UserID  string `json:"id"`
	ConvoID string `json:"cid"`
}

type forwardRoom struct {
	RoomID  string `json:"id"`
	UserID  string `json:"uid"`
	ConvoID string `json:"cid"`
}

func forwardConvo(
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
		Rooms        []forwardRoom `json:"rooms"`
		Success      bool          `json:"success"`
		Content      string        `json:"content"`
		FileIDs      []string      `json:"fileIDs"`
		WsRequestIDs []string      `json:"wsids"`
	}{
		Rooms:        make([]forwardRoom, 0),
		Success:      false,
		Content:      "",
		FileIDs:      make([]string, 0),
		WsRequestIDs: make([]string, 0),
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	var req struct {
		Rooms   []forwardRoom `json:"rooms"`
		Users   []forwardUser `json:"users"`
		ConvoID string        `json:"convoID"`
	}

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	err = database.DB.QueryRow("SELECT content, files FROM chat_messages WHERE id = $1", req.ConvoID).Scan(&data.Content, (*pq.StringArray)(&data.FileIDs))
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	if len(req.Rooms) != 0 {
		data.Rooms = append(data.Rooms, req.Rooms...)
	}
	if len(req.Users) != 0 {
		tempNewRooms, success := createRoomForUsers(&req.Users, uid)
		newRooms := *tempNewRooms
		if success {
			data.Rooms = append(data.Rooms, newRooms...)
		} else {
			json.NewEncoder(w).Encode(data)
			return
		}
	}

	var txn *sql.Tx
	if txn, err = database.DB.Begin(); err != nil {
		fmt.Println(err)
		json.NewEncoder(w).Encode(data)
		return
	}

	var stmt *sql.Stmt
	if stmt, err = txn.Prepare(pq.CopyIn("chat_messages", "id", "rid", "content", "sender_id")); err != nil {
		fmt.Println(err)
		json.NewEncoder(w).Encode(data)
		return
	}

	for _, room := range data.Rooms {
		if _, err = stmt.Exec(room.ConvoID, room.RoomID, data.Content, uid); err != nil {
			fmt.Println(err)
			json.NewEncoder(w).Encode(data)
			return
		}
	}

	if _, err = stmt.Exec(); err != nil {
		fmt.Println(err)
		json.NewEncoder(w).Encode(data)
		return
	}

	if err = stmt.Close(); err != nil {
		fmt.Println(err)
		json.NewEncoder(w).Encode(data)
		return
	}

	if err = txn.Commit(); err != nil {
		fmt.Println(err)
		json.NewEncoder(w).Encode(data)
		return
	}

	now := time.Now().UnixMilli()

	for _, room := range data.Rooms {
		rows, err := database.DB.Query("SELECT uid FROM chatrooms_users WHERE rid = $1", room.RoomID)
		if err != nil {
			continue
		}

		defer rows.Close()

		userIDs := make([]string, 0)

		for rows.Next() {
			var s string
			rows.Scan(&s)
			userIDs = append(userIDs, s)
		}

		wsMessage := instantcomm.Response{
			Type: "chat_new-convo",
			Payload: map[string]interface{}{
				"id":             room.ConvoID,
				"sender":         uid,
				"roomID":         room.RoomID,
				"content":        data.Content,
				"replyMsgID":     "",
				"replyMsg":       "",
				"replyMsgSender": "",
				"fileIDs":        data.FileIDs,
				"dt":             now,
			},
		}

		data.WsRequestIDs = append(data.WsRequestIDs, instantcomm.SaveWsMessageInDB(&wsMessage, &userIDs))
	}

	data.Success = true
	json.NewEncoder(w).Encode(data)
}

func createRoomForUsers(users *[]forwardUser, uid string) (*[]forwardRoom, bool) {
	convoUserMap := map[string]string{}
	userRoomMap := map[string]string{}
	rooms := make([]forwardRoom, 0)

	queryStr := make([]string, 0)
	for _, u := range *users {
		convoUserMap[u.ConvoID] = u.UserID
		queryStr = append(queryStr, "(default)")
	}

	rows, err := database.DB.Query(fmt.Sprintf("INSERT INTO chatrooms (id) VALUES %s RETURNING id", strings.Join(queryStr, ",")))
	if err != nil {
		fmt.Println(err)
		return &rooms, false
	}

	defer rows.Close()

	var i int = 0
	userColl := *users

	for rows.Next() {
		var s string
		rows.Scan(&s)
		userRoomMap[userColl[i].UserID] = s
		i++
	}

	for convo, user := range convoUserMap {
		// convoRoomMap[convo] = userRoomMap[user]
		rooms = append(rooms, forwardRoom{RoomID: userRoomMap[user], ConvoID: convo, UserID: user})
	}

	var txn *sql.Tx
	if txn, err = database.DB.Begin(); err != nil {
		fmt.Println(err)
		return &rooms, false
	}

	var stmt *sql.Stmt
	if stmt, err = txn.Prepare(pq.CopyIn("chatrooms_users", "rid", "uid")); err != nil {
		fmt.Println(err)
		return &rooms, false
	}

	for user, room := range userRoomMap {
		if _, err = stmt.Exec(room, user); err != nil {
			fmt.Println(err)
			return &rooms, false
		}
		if _, err = stmt.Exec(room, uid); err != nil {
			fmt.Println(err)
			return &rooms, false
		}
	}

	if _, err = stmt.Exec(); err != nil {
		fmt.Println(err)
		return &rooms, false
	}

	if err = stmt.Close(); err != nil {
		fmt.Println(err)
		return &rooms, false
	}

	if err = txn.Commit(); err != nil {
		fmt.Println(err)
		return &rooms, false
	}

	return &rooms, true
}

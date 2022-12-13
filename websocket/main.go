package websocket

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/router"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/julienschmidt/httprouter"
)

var users = map[string]map[*net.Conn]bool{}

// var newOnlineUser string

func runWS(res http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	// uid := getUserID(req)
	// if uid == "" {
	// 	res.WriteHeader(http.StatusInternalServerError)
	// 	return
	// }

	var uid string
	fmt.Println("ws-origin: ", req.Header.Get("Origin"))

	myConn, _, _, err := ws.UpgradeHTTP(req, res)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	/*
		if _, keyAlreadyExists := users[uid]; keyAlreadyExists {
			users[uid][&myConn] = true
		} else {
			var user = make(map[*net.Conn]bool)
			user[&myConn] = true
			users[uid] = user
		}
	*/

	go func() {
		defer closeConnection(&myConn)

		var (
			r       = wsutil.NewReader(myConn, ws.StateServerSide)
			decoder = json.NewDecoder(r)
		)

		for {
			hdr, err := r.NextFrame()
			if err != nil {
				CleanOldWsRecords()
				return
			}

			if hdr.OpCode == ws.OpClose {
				if uid != "" && len(users[uid]) == 1 {
					go announceUserStatus(uid, false)
				}
			}

			var req request
			if err := decoder.Decode(&req); err != nil {
				CleanOldWsRecords()
				return
			}

			if req.Request == "" && len(req.Requests) == 0 {
				CleanOldWsRecords()
				return
			}

			if req.Request == "online-users" && req.UserID != "" {
				if checkUserExists(req.UserID) {
					uid = req.UserID
					if _, keyAlreadyExists := users[uid]; keyAlreadyExists {
						users[uid][&myConn] = true
					} else {
						var user = make(map[*net.Conn]bool)
						user[&myConn] = true
						users[uid] = user
					}
					go sendOnlineUserList(&myConn, uid)
					go announceUserStatus(uid, true)
				}
			} else if req.Request == "chat_typing" && uid != "" {
				updateChatRoomTyping(req.ChatRoomID, uid, req.Typing, &myConn)
			} else if req.Request != "" {
				go dispatchMsgFromDB(&myConn, req.Request)
			} else if len(req.Requests) != 0 {
				for _, wsid := range req.Requests {
					go dispatchMsgFromDB(&myConn, wsid)
				}
			}
			CleanOldWsRecords()
		}
	}()
}

func toSelectedUsers(userIDs *[]string, res *Response, myConn *net.Conn) {
	for _, uid := range *userIDs {
		if _, userIsOnline := users[uid]; userIsOnline {
			connMap := users[uid]
			for conn := range connMap {
				if conn == myConn {
					continue
				}
				if err := dispatchIndividualMessage(conn, res); err != nil {
					fmt.Println(err)
					return
				}
			}
		}
	}
}

func updateChatRoomTyping(chatroomID string, uid string, typing bool, myConn *net.Conn) {
	userIDs := make([]string, 0)

	rows, err := database.DB.Query("SELECT uid FROM chatrooms_users WHERE rid = $1", chatroomID)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var s string
		rows.Scan(&s)
		if s != uid {
			userIDs = append(userIDs, s)
		}
	}

	res := Response{
		Type: "chat_typing",
		Payload: map[string]interface{}{
			"roomid": chatroomID,
			"uid":    uid,
			"typing": typing,
		},
	}

	toSelectedUsers(&userIDs, &res, myConn)
}

func sendOnlineUserList(myConn *net.Conn, userID string) {
	userIDs := make([]string, 0)
	for uid := range users {
		userIDs = append(userIDs, uid)
	}
	if len(userIDs) == 0 {
		return
	}

	res := Response{
		Type: "online-users",
		Payload: map[string]interface{}{
			"ids": userIDs,
		},
	}

	w := wsutil.NewWriter(*myConn, ws.StateServerSide, ws.OpText)
	e := json.NewEncoder(w)
	e.Encode(&res)

	if err := w.Flush(); err != nil {
		fmt.Println(err)
		return
	}
}

func dispatchIndividualMessage(conn *net.Conn, res *Response) error {
	w := wsutil.NewWriter(*conn, ws.StateServerSide, ws.OpText)
	e := json.NewEncoder(w)
	e.Encode(res)

	if err := w.Flush(); err != nil {
		return err
	}
	return nil
}

func dispatchMsgFromDB(myConn *net.Conn, reqID string) {
	res := getReqestContent(reqID)

	if res.ToAllRecipients {
		for _, connMap := range users {
			for conn := range connMap {
				if conn == myConn {
					continue
				}
				if err := dispatchIndividualMessage(conn, res); err != nil {
					fmt.Println(err)
					return
				}
			}
		}
	} else {
		userIDs := getReqestReceivers(reqID)
		toSelectedUsers(userIDs, res, myConn)
	}
}

func getReqestContent(reqID string) *Response {
	var res Response
	var s string

	database.DB.QueryRow(`
		SELECT
			action_type,
			payload,
			floor(extract(epoch from dt) * 1000) as dt,
			to_all_recipients
		FROM
			ws_message_content
		WHERE
			id = $1
	`, reqID).Scan(&res.Type, &s, &res.DateTime, &res.ToAllRecipients)
	json.Unmarshal([]byte(s), &res.Payload)

	return &res
}

func getReqestReceivers(reqID string) *[]string {
	userIDs := make([]string, 0)

	rows, err := database.DB.Query("SELECT uid FROM ws_message_to WHERE message_id = $1", reqID)
	if err != nil {
		fmt.Println(err)
		return &userIDs
	}

	defer rows.Close()

	for rows.Next() {
		var s string
		rows.Scan(&s)
		userIDs = append(userIDs, s)
	}

	return &userIDs
}

func announceUserStatus(uid string, online bool) {
	res := Response{
		Type: "user-status",
		Payload: map[string]interface{}{
			"id":     uid,
			"online": online,
		},
	}

	for user, connMap := range users {
		if user == uid {
			continue
		}
		for conn := range connMap {
			if err := dispatchIndividualMessage(conn, &res); err != nil {
				fmt.Println(err)
				return
			}
		}
	}
}

func closeConnection(myConn *net.Conn) {
	for user, connMap := range users {
		for conn := range connMap {
			if conn == myConn {
				delete(users[user], conn)
				if len(users[user]) == 0 {
					delete(users, user)
				}
			}
		}
	}
	(*myConn).Close()
}

func RunWS() {
	router.Router.GET("/ws", runWS)
}

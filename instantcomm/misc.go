package instantcomm

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

func dispatchResponseFromOtherServer(res *Response) {
	if res.ToAllRecipients {
		for _, connMap := range wsUsers {
			for conn := range connMap {
				if err := dispatchIndividualMessage(conn, res); err != nil {
					fmt.Println(err)
				}
			}
		}
	} else if len(res.UserIDs) != 0 {
		for _, uid := range res.UserIDs {
			if connMap, userIsOnline := wsUsers[uid]; userIsOnline {
				for conn := range connMap {
					if err := dispatchIndividualMessage(conn, res); err != nil {
						fmt.Println(err)
					}
				}
			}
		}
	}
}

func dispatchToServers(res *Response) {
	b, err := json.Marshal(*res)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = wsutil.WriteClientMessage(pubsubConn, ws.OpText, b)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func toSelectedUsers(userIDs *[]string, res *Response, myConn *net.Conn) {
	for _, uid := range *userIDs {
		if _, userIsOnline := wsUsers[uid]; userIsOnline {
			connMap := wsUsers[uid]
			for conn := range connMap {
				if myConn != nil && conn == myConn {
					continue
				}
				if err := dispatchIndividualMessage(conn, res); err != nil {
					fmt.Println(err)
				}
			}
		}
	}

	res.UserIDs = *userIDs
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
		UserIDs:         userIDs,
		ToAllRecipients: false,
	}

	toSelectedUsers(&userIDs, &res, myConn)
}

func sendOnlineUserList(myConn *net.Conn, userID string) {
	userIDs := make([]string, 0)
	userIdMap := map[string]bool{}

	for otherServerUID := range otherServersUserCount {
		userIdMap[otherServerUID] = true
	}

	for userid := range wsUsers {
		userIdMap[userid] = true
	}

	for uid := range userIdMap {
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

func toAllRecipients(res *Response, myConn *net.Conn) {
	for _, connMap := range wsUsers {
		for conn := range connMap {
			if myConn != nil && conn == myConn {
				continue
			}
			if err := dispatchIndividualMessage(conn, res); err != nil {
				fmt.Println(err)
			}
		}
	}
}

func dispatchMsgFromDB(myConn *net.Conn, reqID string) {
	res := getReqestContent(reqID)

	if res.ToAllRecipients {
		dispatchToServers(res)
		toAllRecipients(res, myConn)
	} else {
		userIDs := getReqestReceivers(reqID)
		res.UserIDs = *userIDs
		dispatchToServers(res)
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
		ToAllRecipients: true,
	}

	dispatchToServers(&res)

	for user, connMap := range wsUsers {
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
	for user, connMap := range wsUsers {
		for conn := range connMap {
			if conn == myConn {
				delete(wsUsers[user], conn)
				if len(wsUsers[user]) == 0 {
					delete(wsUsers, user)
				}
			}
		}
	}
	(*myConn).Close()
}

package instantcomm

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/julienschmidt/httprouter"
)

var pubsubConn net.Conn

func runWS(res http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	var uid string

	if !originOK(req) {
		return
	}

	myConn, _, _, err := ws.UpgradeHTTP(req, res)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

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
				if uid != "" && len(wsUsers[uid]) == 1 {
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
					if _, keyAlreadyExists := wsUsers[uid]; keyAlreadyExists {
						wsUsers[uid][&myConn] = true
					} else {
						var user = make(map[*net.Conn]bool)
						user[&myConn] = true
						wsUsers[uid] = user
					}
					go sendOnlineUserList(&myConn, uid)
					go announceUserStatus(uid, true)
				} else {
					return
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

func connectWebsocketAsClient() {
	var err error
	pubsubConn, _, _, err = ws.DefaultDialer.Dial(context.Background(), os.Getenv("PROJMGMT_PUBSUB_WS_URL"))

	if err != nil {
		fmt.Println(err)
		return
	}

	defer pubsubConn.Close()

	var forever chan struct{}

	go func() {
		for {
			resBytes, _, err := wsutil.ReadServerData(pubsubConn)
			if err != nil {
				fmt.Println(err)
				return
			}

			var res Response
			err = json.Unmarshal(resBytes, &res)
			if err != nil {
				continue
			}

			if res.Type == "server-disconnect" {
				offlineUsers := make([]string, 0)

				for uid, thisServerUserCount := range res.OtherServersUserCount {
					if count, exists := otherServersUserCount[uid]; exists {
						if count > thisServerUserCount {
							otherServersUserCount[uid] = count - thisServerUserCount
						} else {
							offlineUsers = append(offlineUsers, uid)
							delete(otherServersUserCount, uid)
						}
					}
				}

				if len(offlineUsers) != 0 {
					newResponse := Response{
						Type:            "server-disconnect",
						ToAllRecipients: true,
						UserIDs:         offlineUsers,
					}

					dispatchResponseFromOtherServer(&newResponse)
				}
				continue
			} else if res.Type == "online-users" {
				uid := res.Payload["id"].(string)
				online := res.Payload["online"].(bool)

				if count, exists := otherServersUserCount[uid]; exists {
					if online {
						otherServersUserCount[uid] = count + 1
					} else {
						if count > 1 {
							otherServersUserCount[uid] = count - 1
						} else {
							delete(otherServersUserCount, uid)
						}
					}
				} else if online {
					otherServersUserCount[uid] = 1
				}
			}

			if res.Type != "" {
				dispatchResponseFromOtherServer(&res)
			}
		}
	}()

	<-forever
	fmt.Println("pubsub conn ended")
}

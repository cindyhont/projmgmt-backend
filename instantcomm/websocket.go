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

	// hide for testing
	// if !originOK(req) {
	// 	return
	// }

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

			// test start
			if req.Request != "" {
				b, _ := json.Marshal(req)
				err = wsutil.WriteClientMessage(pubsubConn, ws.OpText, b)
				if err != nil {
					fmt.Println(err)
					return
				}
			}
			// test end

			// hide for testing
			/*
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
			*/
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
			msg, _, err := wsutil.ReadServerData(pubsubConn)
			if err != nil {
				fmt.Println(err)
				return
			} else {
				fmt.Println(string(msg))
			}
		}
	}()

	<-forever
	fmt.Println("pubsub conn ended")
}

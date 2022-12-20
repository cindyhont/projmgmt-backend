package instantcomm

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/julienschmidt/httprouter"
)

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

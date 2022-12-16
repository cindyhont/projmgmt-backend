package login

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/cindyhont/projmgmt-backend/rest/common"
	"github.com/cindyhont/projmgmt-backend/usermgmt"
	"github.com/julienschmidt/httprouter"
)

func login(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var err error

	// delete expired sessions
	var wg sync.WaitGroup
	wg.Add(1)
	go usermgmt.DeleteExpiredSessionsConcurrent(&wg)

	var user model.User
	var body []byte
	if body, err = io.ReadAll(r.Body); err != nil {
		fmt.Println("a:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err = json.Unmarshal(body, &user); err != nil {
		fmt.Println("b:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if common.TooLongTooShort(user.Username, 6, 128) || common.TooLongTooShort(user.Password, 6, 128) {
		common.SendResponse(w, &model.Response{Session: nil, Data: usermgmt.LoginFailedResponse(r)})
		return
	}

	var uid, pwdHash string
	if err = database.DB.QueryRow("SELECT id, password FROM users WHERE username = $1 AND authorized = true", user.Username).Scan(&uid, &pwdHash); err != nil {
		common.SendResponse(w, &model.Response{Session: nil, Data: usermgmt.LoginFailedResponse(r)})
		return
	}

	pwdMatch, _ := usermgmt.ComparePassword(user.Password, pwdHash)
	if pwdMatch {
		var sessionID string
		var expiry time.Time
		database.DB.QueryRow("INSERT INTO sessions (uid) VALUES ($1) RETURNING id, expiry", uid).Scan(&sessionID, &expiry)
		common.SetSessionCookie(w, sessionID, expiry)
		common.SendResponse(w, &model.Response{Session: nil, Data: map[string]bool{"success": true}})
		return
	} else {
		common.SendResponse(w, &model.Response{Session: nil, Data: usermgmt.LoginFailedResponse(r)})
	}
	wg.Wait()
}

package newuser

import (
	"net/http"
	"sync"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/cindyhont/projmgmt-backend/rest/common"
	"github.com/cindyhont/projmgmt-backend/usermgmt"
	"github.com/julienschmidt/httprouter"
)

func newUserByInvitation(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// delete expired sessions
	var wg sync.WaitGroup
	wg.Add(2)
	go usermgmt.DeleteExpiredSessionsConcurrent(&wg)

	inviteKey := r.Header.Get("inviteKey")

	data := struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}{
		Success: false,
		Message: "",
	}

	var uid string
	err := database.DB.QueryRow("SELECT id FROM user_details WHERE invitation_mail_key = $1", inviteKey).Scan(&uid)
	if err != nil {
		common.SendResponse(w, &model.Response{
			Session: nil,
			Data:    data,
		})
		return
	}

	go setInvitationKeyToNull(&wg, uid)

	user, err := getUser(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if invalidInput(user) {
		data.Message = "Invalid username or password."
		common.SendResponse(w, &model.Response{Session: nil, Data: data})
		return
	}

	// hash password
	newPwd, err := usermgmt.GeneratePassword(user.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err := database.DB.Exec("UPDATE users SET username = $1, password = $2 WHERE id = $3", user.Username, newPwd, uid); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// add session in database
	sessionID, expiryMS, err := addSessinToDB(uid)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	common.SetSessionCookie(w, sessionID, expiryMS)
	data.Success = true
	common.SendResponse(w, &model.Response{Session: nil, Data: data})

	wg.Wait()
}

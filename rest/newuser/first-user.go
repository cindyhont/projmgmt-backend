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

func firstUser(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// delete expired sessions
	var wg sync.WaitGroup
	wg.Add(1)
	go usermgmt.DeleteExpiredSessionsConcurrent(&wg)

	data := struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}{
		Success: false,
		Message: "",
	}

	// check if other user exists already in the database, return if other users exists
	var exists bool
	if err := database.DB.QueryRow("select exists (select 1 from users)").Scan(&exists); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if exists {
		data.Message = "There are other users already."
		common.SendResponse(w, &model.Response{Session: nil, Data: data})
		return
	}

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

	newPwd, err := usermgmt.GeneratePassword(user.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// add user id, username, password
	var uid string
	if err := database.DB.QueryRow("INSERT INTO users (username, password) VALUES ($1,$2) RETURNING id", user.Username, newPwd).Scan(&uid); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	sessionID, expiry, err := addSessinToDB(uid)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	common.SetSessionCookie(w, sessionID, expiry)
	data.Success = true
	common.SendResponse(w, &model.Response{Session: nil, Data: data})
	wg.Wait()
}

package settings

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/cindyhont/projmgmt-backend/rest/common"
	"github.com/cindyhont/projmgmt-backend/usermgmt"
	"github.com/julienschmidt/httprouter"
)

func updatePassword(
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
		Success bool `json:"success"`
	}{
		Success: false,
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	var req struct {
		NewPassword     string `json:"newPassword"`
		CurrentPassword string `json:"currentPassword"`
	}

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	if common.TooLongTooShort(req.NewPassword, 6, 128) {
		json.NewEncoder(w).Encode(data)
		return
	}

	var pwdHash string
	database.DB.QueryRow("SELECT password FROM users WHERE id = $1 AND authorized = true", uid).Scan(&pwdHash)
	pwdMatch, _ := usermgmt.ComparePassword(req.CurrentPassword, pwdHash)

	if !pwdMatch {
		json.NewEncoder(w).Encode(data)
		return
	}

	newPwd, err := usermgmt.GeneratePassword(req.NewPassword)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	database.DB.Exec(
		"UPDATE users SET password = $1 WHERE id = $2",
		newPwd,
		uid,
	)

	data.Success = true
	json.NewEncoder(w).Encode(data)
}

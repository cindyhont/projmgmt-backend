package settings

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/instantcomm"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/cindyhont/projmgmt-backend/rest/common"
	"github.com/cindyhont/projmgmt-backend/usermgmt"
	"github.com/julienschmidt/httprouter"
)

func updateUsername(
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
		Success     bool   `json:"success"`
		WsRequestID string `json:"wsid"`
	}{
		Success:     false,
		WsRequestID: "",
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	var req model.User

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	if common.TooLongTooShort(req.Username, 6, 128) || common.TooLongTooShort(req.Password, 6, 128) {
		json.NewEncoder(w).Encode(data)
		return
	}

	var pwdHash string
	database.DB.QueryRow("SELECT password FROM users WHERE id = $1 AND authorized = true", uid).Scan(&pwdHash)
	pwdMatch, _ := usermgmt.ComparePassword(req.Password, pwdHash)

	if !pwdMatch {
		json.NewEncoder(w).Encode(data)
		return
	}

	database.DB.Exec(
		"UPDATE users SET username = $1 WHERE id = $2",
		req.Username,
		uid,
	)

	wsMessage := instantcomm.Response{
		Type: "settings_update-username",
		Payload: map[string]interface{}{
			"username": req.Username,
		},
	}

	data.WsRequestID = instantcomm.SaveWsMessageInDB(&wsMessage, &[]string{uid})
	data.Success = true
	json.NewEncoder(w).Encode(data)
}

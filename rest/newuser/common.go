package newuser

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/cindyhont/projmgmt-backend/rest/common"
)

func getUser(r *http.Request) (*model.User, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	var user model.User
	err = json.Unmarshal(body, &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func invalidInput(user *model.User) bool {
	return common.TooLongTooShort(user.Username, 6, 128) || common.TooLongTooShort(user.Password, 6, 128)
}

func addSessinToDB(uid string) (string, time.Time, error) {
	var sessionID string
	var expiry time.Time
	err := database.DB.QueryRow("INSERT INTO sessions (uid) VALUES ($1) RETURNING id, expiry", uid).Scan(&sessionID, &expiry)
	return sessionID, expiry, err
}

func setInvitationKeyToNull(wg *sync.WaitGroup, uid string) {
	database.DB.Exec("UPDATE user_details SET date_registered_dt = NOW(), last_active_dt = NOW(), invitation_mail_key = NULL WHERE id = $1", uid)
	wg.Done()
}

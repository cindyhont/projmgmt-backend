package usermgmt

import (
	"time"

	"github.com/cindyhont/projmgmt-backend/database"
)

/*
func NewSessionID(oldSID string) (newSID string, userID string, expires time.Time) {
	var uid string
	err := database.DB.QueryRow("SELECT uid FROM sessions WHERE id = $1", oldSID).Scan(&uid)
	if err != nil {
		return "", "", time.Now().Add(time.Duration(-10) * SessionDuration)
	}

	var exists bool
	err = database.DB.QueryRow("SELECT EXISTS (SELECT 1 FROM users WHERE id = $1 AND authorized = true)", uid).Scan(&exists)
	if err != nil {
		return "", "", time.Now().Add(time.Duration(-10) * SessionDuration)
	}

	if !exists {
		return "", "", time.Now().Add(time.Duration(-10) * SessionDuration)
	}

	var newsid string
	var expiry time.Time
	database.DB.QueryRow("INSERT INTO sessions (uid) VALUES ($1) RETURNING id, expiry", expiry, uid).Scan(&newsid, &expiry)
	database.DB.Exec("UPDATE user_details SET last_active_dt = NOW() WHERE id = $1", uid)

	return newsid, uid, expiry
}
*/

func NewSessionID(oldSID string) (newSID string, userID string, expires time.Time) {
	var newsid, uid string
	var expiry time.Time
	err := database.DB.QueryRow(`
		INSERT INTO 
			sessions (uid) 
		SELECT 
			uid 
		FROM 
			users U 
		INNER JOIN
			sessions S
		ON
			S.uid = U.id
		WHERE
			U.authorized = true
		AND 
			S.id = $1
		RETURNING 
			id, uid, expiry
	`, oldSID).Scan(&newsid, &uid, &expiry)
	if err != nil {
		return "", "", time.Now().Add(time.Duration(-10) * SessionDuration)
	}

	database.DB.Exec("UPDATE user_details SET last_active_dt = NOW() WHERE id = $1", uid)

	return newsid, uid, expiry
}

package common

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/cindyhont/projmgmt-backend/usermgmt"
	"github.com/julienschmidt/httprouter"
)

type AuthHandler func(
	w http.ResponseWriter,
	r *http.Request,
	p httprouter.Params,
	s *model.Session,
	signedIn bool,
	uid string,
)
type AuthUrHandler func(
	w http.ResponseWriter,
	r *http.Request,
	p httprouter.Params,
	s *model.Session,
	signedIn bool,
	uid string,
	userRight int,
	systemStarted bool,
)

func sessionID(r *http.Request) (string, string) {
	fetchSessionMethod := r.Header.Get("sMethod")
	oldSessionID := ""
	if fetchSessionMethod == "ck" && r.Header.Get("Origin") == os.Getenv("ORIGIN_REFERRER") {
		s, err := r.Cookie("sid")
		if err == nil {
			oldSessionID = s.Value
		}
	} else if fetchSessionMethod == "body" && strings.Split(r.RemoteAddr, ":")[0] == os.Getenv("FRONTEND_IP") {
		oldSessionID = r.Header.Get("sid")
	}
	return fetchSessionMethod, oldSessionID
}

func AuthRequired(next AuthHandler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		usermgmt.DeleteExpiredSessions()
		fetchSessionMethod, oldSessionID := sessionID(r)

		if oldSessionID == "" {
			next(w, r, p, nil, false, "")
		} else {
			newSID, uid, expiry := usermgmt.NewSessionID(oldSessionID)
			if fetchSessionMethod == "ck" {
				SetSessionCookie(w, newSID, expiry)
				next(w, r, p, nil, true, uid)
			} else {
				next(w, r, p, &model.Session{Sid: newSID, Expires: expiry.UnixMilli()}, true, uid)
			}
		}
	}
}

func AuthUserRight(next AuthUrHandler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		usermgmt.DeleteExpiredSessions()
		fetchSessionMethod, oldSessionID := sessionID(r)

		if oldSessionID == "" {
			next(w, r, p, nil, false, "", 0, false)
		} else {
			var uid string
			var userRight int

			if err := database.DB.QueryRow(`
						SELECT
							UD.id,
							UD.user_right
						FROM
							user_details UD
						INNER JOIN
							sessions S
						ON
							UD.id = S.uid
						INNER JOIN
							users U
						ON
							UD.id = U.id
						WHERE
							S.id = $1
							AND U.authorized = true
					`, oldSessionID).Scan(&uid, &userRight); err != nil {
				next(w, r, p, nil, false, "", 0, false)
				return
			}

			var systemStarted bool
			database.DB.QueryRow("select exists (select 1 from user_details)").Scan(&systemStarted)

			var newSID string
			var expiry time.Time
			database.DB.QueryRow("INSERT INTO sessions (uid) VALUES ($1) RETURNING id, expiry", uid).Scan(&newSID, &expiry)
			database.DB.Exec("UPDATE user_details SET last_active_dt = NOW() WHERE id = $1", uid)

			if fetchSessionMethod == "ck" {
				SetSessionCookie(w, newSID, expiry)
				next(w, r, p, nil, true, uid, userRight, systemStarted)
			} else {
				next(w, r, p, &model.Session{Sid: newSID, Expires: expiry.UnixMilli()}, true, uid, userRight, systemStarted)
			}
		}
	}
}

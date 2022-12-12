package websocket

import (
	"net/http"
	"os"

	"github.com/cindyhont/projmgmt-backend/database"
)

func getUserID(req *http.Request) string {
	s, err := req.Cookie("sid")
	if err != nil {
		return os.Getenv("DEMO_USER")
		// return ""
	}
	sid := s.Value
	var uid string
	err = database.DB.QueryRow("SELECT uid FROM sessions WHERE id = $1", sid).Scan(&uid)
	if err != nil {
		return ""
	}
	return uid
}

// func originOK(req *http.Request) bool {
// 	return req.Header.Get("Origin") == os.Getenv("ORIGIN_REFERRER")
// }

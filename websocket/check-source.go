package websocket

import (
	"fmt"

	"github.com/cindyhont/projmgmt-backend/database"
)

func checkUserExists(uid string) bool {
	var exists bool
	if err := database.DB.QueryRow("SELECT EXISTS (SELECT 1 FROM users WHERE id = $1 AND authorized)", uid).Scan(&exists); err != nil {
		fmt.Println(err)
		return false
	}
	fmt.Println(exists)
	return exists
}

/*
func getUserID(req *http.Request) string {
	s, err := req.Cookie("sid")
	if err != nil {
		fmt.Println(err)
		fmt.Println(req.Cookies())
		return ""
	}
	sid := s.Value
	var uid string
	err = database.DB.QueryRow("SELECT uid FROM sessions WHERE id = $1", sid).Scan(&uid)
	if err != nil {
		return ""
	}
	return uid
}
*/

// func originOK(req *http.Request) bool {
// 	return req.Header.Get("Origin") == os.Getenv("ORIGIN_REFERRER")
// }

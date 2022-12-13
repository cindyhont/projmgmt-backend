package websocket

import (
	"fmt"
	"net/http"
	"os"

	"github.com/cindyhont/projmgmt-backend/database"
)

func checkUserExists(uid string) bool {
	var exists bool
	if err := database.DB.QueryRow("SELECT EXISTS (SELECT 1 FROM users WHERE id = $1 AND authorized)", uid).Scan(&exists); err != nil {
		return false
	}
	return exists
}

func originOK(req *http.Request) bool {
	fmt.Println(req.Header.Get("Origin"))
	fmt.Println(os.Getenv("ORIGIN_REFERRER"))
	// return req.Header.Get("Origin") == os.Getenv("ORIGIN_REFERRER")
	return true
}

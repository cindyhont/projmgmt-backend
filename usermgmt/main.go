package usermgmt

import (
	"sync"
	"time"

	"github.com/cindyhont/projmgmt-backend/database"
)

const (
	SessionDuration  = time.Hour
	MaxWrongLogIn    = 3
	WrongLogInWindow = -10 * time.Minute
)

func DeleteExpiredSessions() {
	database.DB.Exec("DELETE FROM sessions WHERE expiry < NOW()")
}

func DeleteExpiredSessionsConcurrent(wg *sync.WaitGroup) {
	defer wg.Done()
	DeleteExpiredSessions()
}

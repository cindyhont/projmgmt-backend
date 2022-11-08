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

func newUserPrerender(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	inviteKey := r.Header.Get("ref")

	// delete expired sessions
	var wg sync.WaitGroup
	wg.Add(1)
	go usermgmt.DeleteExpiredSessionsConcurrent(&wg)

	var exists bool
	database.DB.QueryRow("select exists (select * from user_details where invitation_mail_key = $1)", inviteKey).Scan(&exists)

	common.SendResponse(w, &model.Response{
		Session: nil,
		Data:    map[string]bool{"exists": exists},
	})

	wg.Wait()
}

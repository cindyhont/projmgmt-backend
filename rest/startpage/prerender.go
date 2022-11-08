package startpage

import (
	"net/http"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/cindyhont/projmgmt-backend/rest/common"
	"github.com/julienschmidt/httprouter"
)

func prerender(w http.ResponseWriter, r *http.Request, _ httprouter.Params, s *model.Session, _ bool, uid string) {
	if s == nil {
		common.SendResponse(w, &model.Response{Session: s, Data: nil})
		return
	}

	var exists bool
	database.DB.QueryRow("select exists (select * from user_details)").Scan(&exists)

	data := struct {
		SystemStarted bool `json:"systemStarted"`
	}{SystemStarted: exists}
	common.SendResponse(w, &model.Response{Session: s, Data: data})
}

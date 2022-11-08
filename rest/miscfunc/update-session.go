package miscfunc

import (
	"net/http"

	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/cindyhont/projmgmt-backend/rest/common"
	"github.com/julienschmidt/httprouter"
)

func updateSession(w http.ResponseWriter, r *http.Request, _ httprouter.Params, s *model.Session, signedIn bool, _ string) {
	data := struct {
		Success bool `json:"success"`
	}{Success: signedIn}
	common.SendResponse(w, &model.Response{Session: s, Data: data})
}

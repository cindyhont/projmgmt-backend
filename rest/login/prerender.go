package login

import (
	"net/http"

	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/cindyhont/projmgmt-backend/rest/common"
	"github.com/julienschmidt/httprouter"
)

func prerender(w http.ResponseWriter, r *http.Request, p httprouter.Params, _ *model.Session, signedIn bool, _ string) {
	common.SendResponse(w, &model.Response{Session: nil, Data: map[string]bool{"sidValid": signedIn}})
}

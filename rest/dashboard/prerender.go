package dashboard

import (
	"net/http"

	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/cindyhont/projmgmt-backend/rest/common"
	"github.com/julienschmidt/httprouter"
)

func prerender(
	w http.ResponseWriter,
	r *http.Request,
	p httprouter.Params,
	s *model.Session,
	signedIn bool,
	uid string,
	userRight int,
	systemStarted bool,
) {
	data := struct {
		UserRight     int  `json:"userRight"`
		SystemStarted bool `json:"systemStarted"`
	}{
		UserRight:     userRight,
		SystemStarted: systemStarted,
	}
	common.SendResponse(w, &model.Response{
		Session: s,
		Data:    data,
	})
}

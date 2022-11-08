package startpage

import (
	"github.com/cindyhont/projmgmt-backend/rest/common"
	"github.com/cindyhont/projmgmt-backend/router"
)

func ListenHTTP() {
	router.Router.POST("/start/submit", common.AuthRequired(submit))
	router.Router.POST("/start/prerender", common.AuthRequired(prerender))
}

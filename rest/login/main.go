package login

import (
	"github.com/cindyhont/projmgmt-backend/rest/common"
	"github.com/cindyhont/projmgmt-backend/router"
)

func ListenHTTP() {
	router.Router.POST("/login", login)
	router.Router.POST("/login-prerender", common.AuthRequired(prerender))
}

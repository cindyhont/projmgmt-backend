package miscfunc

import (
	"github.com/cindyhont/projmgmt-backend/rest/common"
	"github.com/cindyhont/projmgmt-backend/router"
)

func ListenHTTP() {
	router.Router.POST("/update-session", common.AuthRequired(updateSession))
	// router.Router.GET("/search-user/:querystring", common.AuthRequired(searchUser))
	router.Router.POST("/search-user", common.AuthRequired(searchUser))
	router.Router.POST("/fetch-users", common.AuthRequired(fetchUsers))
}

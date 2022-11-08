package ws

import (
	"github.com/cindyhont/projmgmt-backend/rest/common"
	"github.com/cindyhont/projmgmt-backend/router"
)

func ListenHTTP() {
	router.Router.POST("/ws/fetch", common.AuthRequired(fetchOldWsMessages))
}

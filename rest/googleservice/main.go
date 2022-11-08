package googleservice

import (
	"github.com/cindyhont/projmgmt-backend/rest/common"
	"github.com/cindyhont/projmgmt-backend/router"
)

func ListenHTTP() {
	router.Router.GET("/googleservice/get-access-token", getToken)
	router.Router.POST("/googleservice/add-files", common.AuthRequired(addFiles))
	router.Router.POST("/googleservice/add-file", common.AuthRequired(addFile))
}

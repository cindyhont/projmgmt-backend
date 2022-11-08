package settings

import (
	"github.com/cindyhont/projmgmt-backend/rest/common"
	"github.com/cindyhont/projmgmt-backend/router"
)

func ListenHTTP() {
	router.Router.POST("/settings/update-username", common.AuthRequired(updateUsername))
	router.Router.POST("/settings/update-password", common.AuthRequired(updatePassword))
	router.Router.POST("/settings/update-avatar", common.AuthRequired(updateAvatar))
	router.Router.POST("/settings/update-max-child-task-lvl", common.AuthRequired(updateMaxChildTaskLvl))
}

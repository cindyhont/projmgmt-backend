package hrm

import (
	"github.com/cindyhont/projmgmt-backend/rest/common"
	"github.com/cindyhont/projmgmt-backend/rest/hrm/dept"
	"github.com/cindyhont/projmgmt-backend/rest/hrm/hrmcommon"
	"github.com/cindyhont/projmgmt-backend/rest/hrm/staff"
	"github.com/cindyhont/projmgmt-backend/router"
)

func ListenHTTP() {
	router.Router.DELETE("/hrm/delete", common.AuthRequired(hrmcommon.DeleteOnly))
	router.Router.PATCH("/hrm/update-single-field/:tableName", common.AuthRequired(hrmcommon.UpdateSingleField))

	router.Router.POST("/hrm/dept/get-frontend", common.AuthRequired(dept.GetFrontendList))
	router.Router.POST("/hrm/dept/create-active", common.AuthRequired(dept.ActiveCreate))
	router.Router.POST("/hrm/dept/create-passive/:id", common.AuthRequired(dept.PassiveCreate))
	router.Router.POST("/hrm/dept/backend-ids", common.AuthRequired(dept.GetBackendIDs))

	router.Router.POST("/hrm/staff/get-frontend", common.AuthRequired(staff.GetFrontendList))
	router.Router.GET("/hrm/staff/get-supervisor/:id", common.AuthRequired(staff.GetSupervisor))
	router.Router.GET("/hrm/staff/get-department/:id", common.AuthRequired(staff.GetDepartment))
	router.Router.GET("/hrm/staff/search-supervisor/:querystring", staff.SearchSupervisor)
	router.Router.GET("/hrm/staff/search-department/:querystring", staff.SearchDepartment)
	router.Router.POST("/hrm/staff/create-active", common.AuthRequired(staff.ActiveCreate))
	router.Router.POST("/hrm/staff/create-passive/:id", common.AuthRequired(staff.PassiveCreate))
	router.Router.POST("/hrm/staff/backend-ids", common.AuthRequired(staff.GetBackendIDs))
}

package rest

import (
	"github.com/cindyhont/projmgmt-backend/rest/chat"
	"github.com/cindyhont/projmgmt-backend/rest/dashboard"
	"github.com/cindyhont/projmgmt-backend/rest/googleservice"
	"github.com/cindyhont/projmgmt-backend/rest/hrm"
	"github.com/cindyhont/projmgmt-backend/rest/initindex"
	"github.com/cindyhont/projmgmt-backend/rest/login"
	"github.com/cindyhont/projmgmt-backend/rest/miscfunc"
	"github.com/cindyhont/projmgmt-backend/rest/newuser"
	"github.com/cindyhont/projmgmt-backend/rest/settings"
	"github.com/cindyhont/projmgmt-backend/rest/startpage"
	"github.com/cindyhont/projmgmt-backend/rest/tasks"
	"github.com/cindyhont/projmgmt-backend/rest/ws"
)

func ListenHTTP() {
	dashboard.ListenHTTP()
	hrm.ListenHTTP()
	login.ListenHTTP()
	miscfunc.ListenHTTP()
	newuser.ListenHTTP()
	startpage.ListenHTTP()
	chat.ListenHTTP()
	googleservice.ListenHTTP()
	initindex.ListenHTTP()
	tasks.ListenHTTP()
	ws.ListenHTTP()
	settings.ListenHTTP()
}

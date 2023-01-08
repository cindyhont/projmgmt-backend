package instantcomm

import (
	"github.com/cindyhont/projmgmt-backend/router"
)

func Run() {
	go pingWebsocket()
	router.Router.GET("/ws", runWS)
}

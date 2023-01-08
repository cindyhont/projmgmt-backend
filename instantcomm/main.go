package instantcomm

import (
	"github.com/cindyhont/projmgmt-backend/router"
)

func Run() {
	go pingWs()
	router.Router.GET("/ws", runWS)
}

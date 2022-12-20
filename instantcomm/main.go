package instantcomm

import "github.com/cindyhont/projmgmt-backend/router"

func Run() {
	go runRabbitmq()
	router.Router.GET("/ws", runWS)
}

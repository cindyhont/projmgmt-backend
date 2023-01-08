package instantcomm

import (
	"github.com/cindyhont/projmgmt-backend/router"
)

func Run() {
	router.Router.GET("/ws", runWS)
}

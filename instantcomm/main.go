package instantcomm

import (
	"github.com/cindyhont/projmgmt-backend/router"
)

func Run() {
	go pingWebsocket()
	go connectWebsocketAsClient()
	router.Router.GET("/ws", runWS)
}

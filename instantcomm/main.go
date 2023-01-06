package instantcomm

import (
	"os"

	"github.com/cindyhont/projmgmt-backend/router"
)

func Run() {
	if os.Getenv("SELF_PRIVATE") != "" {
		// run if in production mode
		go runRabbitmq()
	}
	router.Router.GET("/ws", runWS)
}

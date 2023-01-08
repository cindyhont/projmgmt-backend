package instantcomm

import (
	"time"

	"github.com/gobwas/ws"
)

func pingWs() {
	for {
		for _, connMap := range wsUsers {
			for conn := range connMap {
				(*conn).Write(ws.CompiledPing)
			}
		}
		time.Sleep(time.Second)
	}
}

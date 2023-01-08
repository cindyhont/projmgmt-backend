package instantcomm

import (
	"fmt"
	"time"

	"github.com/gobwas/ws"
)

func pingWs() {
	for {
		for _, connMap := range wsUsers {
			for conn := range connMap {
				n, err := (*conn).Write(ws.CompiledPing)
				if err != nil {
					fmt.Println(n)
					fmt.Println(err)
				}
			}
		}
		time.Sleep(time.Second)
	}
}

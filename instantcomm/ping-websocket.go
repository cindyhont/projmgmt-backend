package instantcomm

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

func pingWebsocket() {
	// message has to be map, or the client side will disconnect
	// interval has to be under 60 seconds

	myMap := make(map[string]string)
	for {
		for _, connMap := range wsUsers {
			for conn := range connMap {
				w := wsutil.NewWriter(*conn, ws.StateServerSide, ws.OpText)
				e := json.NewEncoder(w)
				e.Encode(myMap)

				if err := w.Flush(); err != nil {
					fmt.Println(err)
				}
			}
		}
		time.Sleep(30 * time.Second)
	}
}

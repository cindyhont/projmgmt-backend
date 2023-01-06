package instantcomm

import (
	"net"
)

var (
	wsUsers          = map[string]map[*net.Conn]bool{}
	otherServersConn = map[string]map[string]int{}
)

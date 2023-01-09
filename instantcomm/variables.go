package instantcomm

import (
	"net"
)

var (
	wsUsers               = map[string]map[*net.Conn]bool{}
	otherServersUserCount = map[string]int{}
)

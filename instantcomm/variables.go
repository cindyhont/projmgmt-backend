package instantcomm

import (
	"net"

	"github.com/streadway/amqp"
)

var (
	wsUsers = map[string]map[*net.Conn]bool{}
	// servers              = map[string]time.Time{}
	otherServersConn = map[string]map[string]int{}
	rabbitmqChannel  *amqp.Channel
	// messageQueue         amqp.Queue
	serverHeartbeatQueue amqp.Queue
)

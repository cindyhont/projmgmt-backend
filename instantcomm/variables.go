package instantcomm

import (
	"net"
	"time"

	"github.com/streadway/amqp"
)

var (
	wsUsers                = map[string]map[*net.Conn]bool{}
	servers                = map[string]time.Time{}
	otherServersConn       = map[string]map[string]int{}
	rabbitmqChannel        *amqp.Channel
	myMessageQueue         amqp.Queue
	myServerHeartbeatQueue amqp.Queue
	otherMessageQueues     []amqp.Queue
	// otherServerHeartbeatQueues []amqp.Queue
)

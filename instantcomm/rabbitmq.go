package instantcomm

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/streadway/amqp"
)

const rabbitMqExchangeName = "projmgmt"

func runRabbitmq() {
	conn, err := amqp.Dial(os.Getenv("RABBITMQ_URL"))
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	defer conn.Close()

	rabbitmqChannel, err = conn.Channel()
	if err != nil {
		panic(err)
	}
	defer rabbitmqChannel.Close()

	err = rabbitmqChannel.ExchangeDeclare(
		rabbitMqExchangeName,
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		panic(err)
	}

	serverHeartbeatQueue, err = rabbitmqChannel.QueueDeclare(
		"projmgmt-server-heartbeat",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		panic(err)
	}

	err = rabbitmqChannel.QueueBind(
		serverHeartbeatQueue.Name,
		"",
		rabbitMqExchangeName,
		false,
		nil,
	)
	if err != nil {
		panic(err)
	}

	messageQueue, err = rabbitmqChannel.QueueDeclare(
		"projmgmt-message-queue",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		panic(err)
	}

	err = rabbitmqChannel.QueueBind(
		messageQueue.Name,
		"",
		rabbitMqExchangeName,
		false,
		nil,
	)
	if err != nil {
		panic(err)
	}
	go publishHeartbeat()
	subscribeServerHeartbeat()
	go checkServerWorking()
	subscribeServerMessage()
}

func subscribeServerMessage() {
	fmt.Println("subscribing message queue")
	var forever chan struct{}

	msgs, err := rabbitmqChannel.Consume(
		messageQueue.Name, // queue
		"",                // consumer
		true,              // auto-ack
		false,             // exclusive
		false,             // no-local
		false,             // no-wait
		nil,               // args
	)
	if err != nil {
		panic(err)
	}

	go func() {
		thisServerIP := os.Getenv("SELF_PRIVATE")
		for msg := range msgs {
			var res Response
			if err = json.Unmarshal(msg.Body, &res); err != nil {
				continue
			}

			if res.FromIP == thisServerIP {
				continue
			}

			if res.Type == "user-status" {
				uid := res.Payload["id"].(string)
				online := res.Payload["online"].(bool)

				if online {
					if _, serverExists := otherServersConn[res.FromIP]; serverExists {
						if _, userExists := otherServersConn[res.FromIP][uid]; userExists {
							otherServersConn[res.FromIP][uid] = otherServersConn[res.FromIP][uid] + 1
						} else {
							otherServersConn[res.FromIP][uid] = 1
						}
					} else {
						var server = make(map[string]int)
						server[uid] = 1
						otherServersConn[res.FromIP] = server
					}
				} else {
					if _, serverExists := otherServersConn[res.FromIP]; serverExists {
						if _, userExists := otherServersConn[res.FromIP][uid]; userExists {
							if otherServersConn[res.FromIP][uid] > 1 {
								otherServersConn[res.FromIP][uid] = otherServersConn[res.FromIP][uid] - 1
							} else {
								delete(otherServersConn[res.FromIP], uid)
								if len(otherServersConn[res.FromIP]) == 0 {
									delete(otherServersConn, res.FromIP)
								}
							}
						}
					}
				}
			}

			res.FromIP = ""

			if res.ToAllRecipients {
				toAllRecipients(&res, nil)
			} else if len(res.UserIDs) != 0 {
				userIDs := make([]string, 0)
				copy(userIDs, res.UserIDs)
				res.UserIDs = nil
				toSelectedUsers(&userIDs, &res, nil)
			}
		}
	}()
	<-forever
}

func publishHeartbeat() {
	for {
		err := rabbitmqChannel.Publish(
			rabbitMqExchangeName,
			serverHeartbeatQueue.Name,
			false,
			false,
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte(os.Getenv("SELF_PRIVATE")),
			},
		)
		if err != nil {
			panic(err)
		}
		time.Sleep(time.Second)
	}
}

func subscribeServerHeartbeat() {
	fmt.Println("subscribing heartbeat")
	var forever chan struct{}

	msgs, err := rabbitmqChannel.Consume(
		serverHeartbeatQueue.Name, // queue
		"",                        // consumer
		true,                      // auto-ack
		false,                     // exclusive
		false,                     // no-local
		false,                     // no-wait
		nil,                       // args
	)
	if err != nil {
		panic(err)
	}

	go func() {
		for msg := range msgs {
			serverIP := string(msg.Body)
			servers[serverIP] = time.Now()
		}
	}()
	<-forever
}

func checkServerWorking() {
	interval := time.Second * 5
	for {
		fiveSecAgo := time.Now().Add(interval)
		for serverIP, lastHeartbeatTime := range servers {
			if lastHeartbeatTime.Before(fiveSecAgo) {
				delete(otherServersConn, serverIP)
				delete(servers, serverIP)
			}
		}
		time.Sleep(interval)
	}
}

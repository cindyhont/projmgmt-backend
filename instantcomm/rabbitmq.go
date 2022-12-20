package instantcomm

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/streadway/amqp"
)

const rabbitMqExchangeName = "projmgmt"

func heartbeatQueueName(ip string) string {
	return fmt.Sprintf("projmgmt-server-heartbeat-%s", ip)
}

func messageQueueName(ip string) string {
	return fmt.Sprintf("projmgmt-message-queue-%s", ip)
}

func runRabbitmq() {
	thisServerIP := os.Getenv("SELF_PRIVATE")

	ips := []string{
		os.Getenv("INSTANCE_A_PRIVATE"),
		os.Getenv("INSTANCE_B_PRIVATE"),
	}

	conn, err := amqp.Dial(os.Getenv("RABBITMQ_URL"))
	if err != nil {
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

	for _, serverIP := range ips {
		if serverIP == thisServerIP {
			myServerHeartbeatQueue, err = rabbitmqChannel.QueueDeclare(
				heartbeatQueueName(serverIP),
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
				myServerHeartbeatQueue.Name,
				"",
				rabbitMqExchangeName,
				false,
				nil,
			)
			if err != nil {
				panic(err)
			}
			go subscribeServerHeartbeat()

			///////////////

			myMessageQueue, err = rabbitmqChannel.QueueDeclare(
				messageQueueName(serverIP),
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
				myMessageQueue.Name,
				"",
				rabbitMqExchangeName,
				false,
				nil,
			)
			if err != nil {
				panic(err)
			}

			go subscribeServerMessage()
		} else {
			go publishHeartbeat(serverIP)

			queue, err := rabbitmqChannel.QueueDeclare(
				messageQueueName(serverIP),
				true,
				false,
				false,
				false,
				nil,
			)
			if err != nil {
				panic(err)
			}

			otherMessageQueues = append(otherMessageQueues, queue)
		}
	}

	go checkServerWorking()
}

func publishHeartbeat(ip string) {
	queue, err := rabbitmqChannel.QueueDeclare(
		heartbeatQueueName(ip),
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		panic(err)
	}

	thisServerIpBytes := []byte(os.Getenv("SELF_PRIVATE"))

	for {
		err := rabbitmqChannel.Publish(
			rabbitMqExchangeName,
			queue.Name,
			false,
			false,
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        thisServerIpBytes,
			},
		)
		if err != nil {
			panic(err)
		}
		time.Sleep(time.Second)
	}
}

func subscribeServerHeartbeat() {
	var forever chan struct{}

	msgs, err := rabbitmqChannel.Consume(
		myServerHeartbeatQueue.Name, // queue
		"",                          // consumer
		true,                        // auto-ack
		false,                       // exclusive
		false,                       // no-local
		false,                       // no-wait
		nil,                         // args
	)
	if err != nil {
		panic(err)
	}

	go func() {
		for msg := range msgs {
			serverIP := string(msg.Body)
			fmt.Println(serverIP)
			servers[serverIP] = time.Now()
		}
	}()
	<-forever
}

func subscribeServerMessage() {
	var forever chan struct{}

	msgs, err := rabbitmqChannel.Consume(
		myMessageQueue.Name, // queue
		"",                  // consumer
		true,                // auto-ack
		false,               // exclusive
		false,               // no-local
		false,               // no-wait
		nil,                 // args
	)
	if err != nil {
		panic(err)
	}

	go func() {
		for msg := range msgs {
			var res Response
			if err = json.Unmarshal(msg.Body, &res); err != nil {
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

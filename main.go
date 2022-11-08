package main

import (
	"fmt"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/rest"
	"github.com/cindyhont/projmgmt-backend/router"
	"github.com/cindyhont/projmgmt-backend/websocket"
)

func init() {
	database.Setup()
	fmt.Println("init complete")
}

func main() {
	rest.ListenHTTP()
	websocket.RunWS()
	router.Listen()
}

package main

import (
	"fmt"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/instantcomm"
	"github.com/cindyhont/projmgmt-backend/rest"
	"github.com/cindyhont/projmgmt-backend/router"
)

func init() {
	database.Setup()
	fmt.Println("init complete")
}

func main() {
	rest.ListenHTTP()
	instantcomm.Run()
	router.Listen()
}

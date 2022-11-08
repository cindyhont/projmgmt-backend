package newuser

import "github.com/cindyhont/projmgmt-backend/router"

func ListenHTTP() {
	router.Router.POST("/first-user", firstUser)
	router.Router.POST("/new-user", newUserByInvitation)
	router.Router.POST("/new-user-prerender", newUserPrerender)
	router.Router.POST("/create-visitor", createVisitor)
}

package chat

import (
	"github.com/cindyhont/projmgmt-backend/rest/common"
	"github.com/cindyhont/projmgmt-backend/router"
)

func ListenHTTP() {
	router.Router.POST("/chat/fetch-more-rooms", common.AuthRequired(fetchMoreRooms))
	router.Router.POST("/chat/fetch-specific-rooms", common.AuthRequired(fetchSpecificRooms))

	router.Router.GET("/chat/search-chatrooms/:querystring", common.AuthRequired(searchChatroom))

	router.Router.POST("/chat/create-room-with-first-convo", common.AuthRequired(createRoomWithFirstConvo))
	router.Router.GET("/chat/create-room-no-convo/:roommate-id", common.AuthRequired(createRoomNoConvo))
	router.Router.POST("/chat/create-convo", common.AuthRequired(createConvo))
	router.Router.POST("/chat/create-group", common.AuthRequired(createGroup))

	router.Router.GET("/chat/update-last-seen/:room-id", common.AuthRequired(updateLastSeen))

	router.Router.POST("/chat/forward-convo", common.AuthRequired(forwardConvo))
	router.Router.POST("/chat/update-pinned", common.AuthRequired(updatePinned))
	router.Router.POST("/chat/update-mark-as-read", common.AuthRequired(updateMarkAsRead))
	router.Router.POST("/chat/edit-convo", common.AuthRequired(editConvo))
	router.Router.POST("/chat/fetch-more-convos", common.AuthRequired(fetchMoreConvos))
	router.Router.POST("/chat/fetch-replied-convos", common.AuthRequired(fetchRepliedConvos))
}

package tasks

import (
	"github.com/cindyhont/projmgmt-backend/rest/common"
	"github.com/cindyhont/projmgmt-backend/router"
)

func ListenHTTP() {
	router.Router.GET("/tasks/fetch-task/:task-id", common.AuthRequired(fetchTask))
	router.Router.POST("/tasks/fetch-parent-child-tasks", common.AuthRequired(fetchParentChildTasks))
	router.Router.POST("/tasks/add-task", common.AuthRequired(addTask))
	router.Router.POST("/tasks/edit-main-field", common.AuthRequired(editTaskMainField))
	router.Router.POST("/tasks/edit-extra-field", common.AuthRequired(editTaskExtraField))
	router.Router.POST("/tasks/edit-custom-field-config", common.AuthRequired(editCustomFieldConfig))
	router.Router.POST("/tasks/delete-board-column", common.AuthRequired(deleteBoardColumn))
	router.Router.POST("/tasks/update-timer", common.AuthRequired(updateTimer))
	router.Router.POST("/tasks/add-custom-field", common.AuthRequired(addCustomField))
	router.Router.POST("/tasks/delete-custom-field", common.AuthRequired(deleteCustomField))
	router.Router.POST("/tasks/add-comment", common.AuthRequired(addComment))
	router.Router.POST("/tasks/delete-comment", common.AuthRequired(deleteComment))
	router.Router.POST("/tasks/edit-task-with-new-files", common.AuthRequired(editTaskWithNewFiles))
	router.Router.POST("/tasks/edit-comment", common.AuthRequired(editComment))
	router.Router.POST("/tasks/task-moved-in-board", common.AuthRequired(taskMovedInBoard))
	router.Router.POST("/tasks/delete-task", common.AuthRequired(deleteTask))
	router.Router.POST("/tasks/search-tasks", common.AuthRequired(searchTasks))
	router.Router.POST("/tasks/update-parents", common.AuthRequired(updateParents))
}

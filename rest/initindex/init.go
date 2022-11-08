package initindex

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/cindyhont/projmgmt-backend/rest/chat"
	"github.com/cindyhont/projmgmt-backend/rest/common"
	"github.com/cindyhont/projmgmt-backend/rest/tasks"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/lib/pq"
)

type InitResponse struct {
	SystemStarted      bool   `json:"systemStarted"`
	UID                string `json:"uid"`
	MaxChildTaskLvl    int    `json:"maxChildTaskLvl"`
	UserName           string `json:"username"`
	Visitor            bool   `json:"visitor"`
	MoveToChatMainPage bool   `json:"moveToChatMainPage"`

	Rooms chat.RoomEntity         `json:"rooms"`
	Users model.UserDetailsEntity `json:"users"`
	Files model.FileEntity        `json:"files"`

	Approval         tasks.ApprovalEntity        `json:"approvalList"`
	CustomFieldTypes tasks.CustomFieldTypeEntity `json:"customFieldTypes"`
	UserCustomFields []tasks.UserCustomFields    `json:"customFields"`
	TaskComments     tasks.CommentEntity         `json:"comments"`
	TimeRecords      tasks.TimeRecordEntity      `json:"timeRecords"`
	Tasks            tasks.TaskEntity            `json:"tasks"`
	TaskRecords      tasks.TaskRecordEntity      `json:"taskRecords"`
}

func valueIsInList(arr *[]string, value string) bool {
	for _, v := range *arr {
		if value == v {
			return true
		}
	}
	return false
}

func initIndexPage(
	w http.ResponseWriter,
	r *http.Request,
	p httprouter.Params,
	s *model.Session,
	signedIn bool,
	uid string,
) {
	if s == nil {
		fmt.Println("no session")
		common.SendResponse(w, &model.Response{Session: s, Data: nil})
		return
	}

	data := InitResponse{
		SystemStarted:      false,
		UID:                uid,
		UserName:           "",
		Visitor:            false,
		MoveToChatMainPage: false,

		Rooms: chat.RoomEntity{
			IDs:      make([]string, 0),
			Entities: make(map[string]chat.Room),
		},
		Users: model.UserDetailsEntity{
			IDs:      make([]string, 0),
			Entities: make(map[string]model.UserDetails),
		},
		Files: model.FileEntity{
			IDs:      make([]string, 0),
			Entities: make(map[string]model.File),
		},

		Approval: tasks.ApprovalEntity{
			IDs:      make([]int, 0),
			Entities: make(map[int]tasks.Approval),
		},
		CustomFieldTypes: tasks.CustomFieldTypeEntity{
			IDs:      make([]string, 0),
			Entities: make(map[string]tasks.CustomFieldType),
		},
		UserCustomFields: make([]tasks.UserCustomFields, 0),
		TaskComments: tasks.CommentEntity{
			IDs:      make([]string, 0),
			Entities: make(map[string]tasks.Comment),
		},
		TimeRecords: tasks.TimeRecordEntity{
			IDs:      make([]string, 0),
			Entities: make(map[string]tasks.TimeRecord),
		},
		Tasks: tasks.TaskEntity{
			IDs:      make([]string, 0),
			Entities: make(map[string]map[string]interface{}),
		},
		TaskRecords: tasks.TaskRecordEntity{
			IDs:      make([]string, 0),
			Entities: make(map[string]tasks.TaskRecord),
		},
	}

	chatRoomID := r.Header.Get("chatroomid")
	chatUserID := r.Header.Get("chatuserid")

	if len(chatRoomID) != 0 || len(chatUserID) != 0 {
		data.MoveToChatMainPage = true
	}

	database.DB.QueryRow("select exists (select * from user_details)").Scan(&data.SystemStarted)

	if !data.SystemStarted {
		fmt.Println("system not started")
		common.SendResponse(w, &model.Response{Session: s, Data: data})
		return
	}

	roomIDs := *chat.GetInitRoomIDs(uid)

	_, err := uuid.Parse(chatRoomID)
	if len(chatRoomID) != 0 && err == nil {
		chatRoomIdIsInList := valueIsInList(&roomIDs, chatRoomID)
		if chatRoomIdIsInList {
			data.MoveToChatMainPage = false
		} else {
			roomExists := false
			database.DB.QueryRow(`
				SELECT EXISTS (
					SELECT 1
					FROM chatrooms_users
					WHERE
						rid = $1
					AND
						uid = $2
					AND
						in_users_list
				)
			`, chatRoomID, uid).Scan(roomExists)
			if roomExists {
				roomIDs = append(roomIDs, chatRoomID)
				data.MoveToChatMainPage = false
			}
		}
	}

	fileIDs := make([]string, 0)
	userIDs := make([]string, 0)

	userIDs = append(userIDs, uid)

	if len(chatUserID) != 0 {
		_, err := uuid.Parse(chatUserID)
		if err != nil || chatUserID == uid {
			data.MoveToChatMainPage = true
		} else {
			chatUserIdExists := false

			database.DB.QueryRow(`
				SELECT EXISTS (
					SELECT 1 
					FROM user_details UD
					INNER JOIN users U
					ON UD.id = U.id
					WHERE UD.id = $1 
					AND UD.date_registered_dt IS NOT NULL
					AND U.authorized
				)
			`, chatUserID).Scan(&chatUserIdExists)

			if chatUserIdExists {
				data.MoveToChatMainPage = false
				userIDs = append(userIDs, chatUserID)

				matchingRoomID := ""
				err = database.DB.QueryRow(`
					SELECT
						rid
					FROM
						(
							SELECT
								CU.rid,
								ARRAY_AGG(CU.uid) as uids,
								C.room_name AS room_name
							FROM
								chatrooms_users CU
							inner join chatrooms C
							on
								CU.rid = C.id
							where 
								room_name is null
							GROUP BY
								rid, C.room_name
						) AS TEMP
					WHERE 
						uids <@ $1
					AND
						room_name IS NULL
				`, pq.Array([]string{uid, chatUserID})).Scan(&matchingRoomID)
				if err != nil {
					fmt.Println("b", err)
				}

				if len(matchingRoomID) != 0 && !valueIsInList(&roomIDs, matchingRoomID) {
					roomIDs = append(roomIDs, matchingRoomID)
				}
			}
		}
	}

	if data.MoveToChatMainPage {
		fmt.Println("move to chart main page")
		common.SendResponse(w, &model.Response{Session: s, Data: data})
		return
	}

	database.DB.QueryRow(`
		SELECT 
			U.username,
			UD.max_child_task_level,
			UD.visitor
		FROM 
			users U
		INNER JOIN 
			user_details UD
		ON
			U.id = UD.id
		WHERE 
			U.id = $1
	`, uid).Scan(&data.UserName, &data.MaxChildTaskLvl, &data.Visitor)

	for _, roomID := range roomIDs {
		room, _fileIDs, _possibleUserIDs := chat.GetInitRoomInfo(roomID, uid)
		if len(room.ID) == 0 {
			continue
		}
		fileIDs = append(fileIDs, *_fileIDs...)
		userIDs = append(userIDs, room.User.IDs...)
		userIDs = append(userIDs, *_possibleUserIDs...)
		data.Rooms.IDs = append(data.Rooms.IDs, roomID)
		data.Rooms.Entities[room.ID] = *room
	}

	_approvalIDs, _approvalMap := tasks.GetApprovalList()
	data.Approval.IDs = *_approvalIDs
	data.Approval.Entities = *_approvalMap

	_customFieldTypeIDs, _customFieldTypeMap := tasks.GetCustomFieldTypes()
	data.CustomFieldTypes.IDs = *_customFieldTypeIDs
	data.CustomFieldTypes.Entities = *_customFieldTypeMap

	_userCustomField := tasks.GetUserCustomFields(uid)
	data.UserCustomFields = *_userCustomField

	_taskIDs, _myTaskIDs, _taskMap, _taskUserIDs, _taskFileIDs := tasks.GetInitTasksEntity(uid)
	userIDs = append(userIDs, *_taskUserIDs...)
	fileIDs = append(fileIDs, *_taskFileIDs...)

	_timeRecordIDs, _timeRecordMap, _timeRecordUserIDs := tasks.GetTimeRecordsEntity(_myTaskIDs)
	userIDs = append(userIDs, *_timeRecordUserIDs...)
	data.TimeRecords.IDs = *_timeRecordIDs
	data.TimeRecords.Entities = *_timeRecordMap

	_taskRecordIDs, _taskRecordMap, _taskRecordUserIDs := tasks.GetTaskRecordsEntity(_myTaskIDs)
	userIDs = append(userIDs, *_taskRecordUserIDs...)
	data.TaskRecords.IDs = *_taskRecordIDs
	data.TaskRecords.Entities = *_taskRecordMap

	_taskCommentIDs, _tasksCommentMap, _taskCommentUserIDs, _taskCommentFileIDs := tasks.GetCommentsEntity(_myTaskIDs)
	userIDs = append(userIDs, *_taskCommentUserIDs...)
	fileIDs = append(fileIDs, *_taskCommentFileIDs...)
	data.TaskComments.IDs = *_taskCommentIDs
	data.TaskComments.Entities = *_tasksCommentMap

	_tasksHaveExtraFields, _taskExtraFieldsMap := tasks.GetUserCustomFieldValues(uid)
	if len(*_taskIDs) != 0 {
		taskMap := make(map[string]map[string]interface{})
		data.Tasks.IDs = *_taskIDs
		tempTaskData, _ := json.Marshal(*_taskMap)
		json.Unmarshal(tempTaskData, &taskMap)

		if _tasksHaveExtraFields {
			for taskID, extraFieldValues := range *_taskExtraFieldsMap {
				if _, thisTaskValid := taskMap[taskID]; thisTaskValid {
					for key, value := range extraFieldValues {
						taskMap[taskID][key] = value
					}
				}
			}
		}

		data.Tasks.Entities = taskMap
	}

	_userIDs, _userMap := GetInitUserDetails(&userIDs)
	data.Users.IDs = *_userIDs
	data.Users.Entities = *_userMap

	if len(fileIDs) != 0 {
		fileIdList, fileMap := GetInitFileDetails(&fileIDs)
		data.Files.IDs = *fileIdList
		data.Files.Entities = *fileMap
	}

	common.SendResponse(w, &model.Response{Session: s, Data: data})
}

func GetInitFileDetails(fileIDs *[]string) (*[]string, *map[string]model.File) {
	fileIdList := make([]string, 0)
	fileMap := map[string]model.File{}

	rows, err := database.DB.Query(`
		select
			id,
			name,
			size
		from
			files
		where
			id = any($1)
	`, pq.Array(common.UniqueStringFromSlice(fileIDs)))
	if err != nil {
		return &fileIdList, &fileMap
	}
	defer rows.Close()

	for rows.Next() {
		var f model.File
		rows.Scan(
			&f.ID,
			&f.Name,
			&f.Size,
		)
		fileIdList = append(fileIdList, f.ID)
		f.Downloading = false
		f.Progress = 0
		f.Url = ""
		fileMap[f.ID] = f
	}
	return &fileIdList, &fileMap
}

func GetInitUserDetails(userIDs *[]string) (*[]string, *map[string]model.UserDetails) {
	userIdList := make([]string, 0)
	userMap := map[string]model.UserDetails{}

	rows, err := database.DB.Query(`
		select
			id,
			first_name,
			last_name,
			coalesce(avatar,'') as avatar
		from
			user_details
		where
			id = any($1)
	`, pq.Array(common.UniqueStringFromSlice(userIDs)))
	if err != nil {
		return &userIdList, &userMap
	}

	defer rows.Close()

	for rows.Next() {
		var u model.UserDetails
		rows.Scan(
			&u.ID,
			&u.FirstName,
			&u.LastName,
			&u.Avatar,
		)
		u.Online = false
		userIdList = append(userIdList, u.ID)
		userMap[u.ID] = u
	}
	return &userIdList, &userMap
}

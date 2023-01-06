package chat

import (
	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/lib/pq"
)

type Convo struct {
	ID             string   `json:"id"`
	Content        string   `json:"content"`
	Sender         string   `json:"sender"`
	DateTime       int64    `json:"dt"`
	ReplyMsgID     string   `json:"replyMsgID"`
	ReplyMsg       string   `json:"replyMsg"`
	ReplyMsgSender string   `json:"replyMsgSender"`
	EditDateTime   int64    `json:"editDt"`
	FileIDs        []string `json:"fileIDs"`
	Error          bool     `json:"error"`
	Sent           bool     `json:"sent"`
}

type ConvoEntity struct {
	IDs      []string         `json:"ids"`
	Entities map[string]Convo `json:"entities"`
}

type RoomUser struct {
	ID       string `json:"id"`
	Typing   bool   `json:"typing"`
	LastSeen int64  `json:"lastSeen"`
}

type RoomUserEntity struct {
	IDs      []string            `json:"ids"`
	Entities map[string]RoomUser `json:"entities"`
}

type FileInputEntity struct {
	IDs      []string               `json:"ids"`
	Entities map[string]interface{} `json:"entities"`
}

type Room struct {
	ID                    string          `json:"id"`
	Name                  string          `json:"name"`
	Avatar                string          `json:"avatar"`
	IsGroup               bool            `json:"isGroup"`
	MarkAsRead            int             `json:"markAsRead"`
	Pinned                bool            `json:"pinned"`
	Convo                 ConvoEntity     `json:"convos"`
	User                  RoomUserEntity  `json:"users"`
	FileInputs            FileInputEntity `json:"fileInputs"`
	Draft                 string          `json:"draft"`
	ReplyMsgID            string          `json:"replyMsgID"`
	Reply                 bool            `json:"reply"`
	EditMsgID             string          `json:"editMsgID"`
	Edit                  bool            `json:"edit"`
	ScrollY               int             `json:"scrollY"`
	HasMoreConvos         bool            `json:"hasMoreConvos"`
	FetchingConvos        bool            `json:"fetchingConvos"`
	ViewportLatestConvoID string          `json:"viewportLatestConvoID"`
}

type RoomEntity struct {
	IDs      []string        `json:"ids"`
	Entities map[string]Room `json:"entities"`
}

func GetInitRoomInfo(roomID string, uid string) (*Room, *[]string, *[]string) {
	data := Room{
		FileInputs: FileInputEntity{
			IDs:      make([]string, 0),
			Entities: make(map[string]interface{}),
		},
	}

	database.DB.QueryRow(`
		WITH room_info AS (
			SELECT 
				C.id,
				COALESCE(C.room_name,TRIM(CONCAT(UD.first_name,' ',UD.last_name))) AS name,
				COALESCE(C.avatar,UD.avatar,'') AS avatar,
				C.room_name NOTNULL AS isgroup
			FROM 
				chatrooms C
			INNER JOIN
				chatrooms_users CU
			ON 
				C.id = CU.rid
			INNER JOIN
				user_details UD
			ON
				CU.uid = UD.id
			WHERE
				C.id = $1
			AND 
				CU.uid <> $2
			LIMIT 1
		),
		user_info AS (
			SELECT 
				mark_as_read,
				pinned
			FROM 
				chatrooms_users
			WHERE
				rid = $1
			AND 
				uid = $2
		),
		message_info AS (
			SELECT
				COUNT(id) > 20 AS has_more_convos
			FROM 
				chat_messages
			WHERE
				rid = $1
		)
		SELECT
			RI.id,
			RI.name,
			RI.avatar,
			RI.isgroup,
			UI.mark_as_read,
			UI.pinned,
			MI.has_more_convos
		FROM
			room_info RI,
			user_info UI,
			message_info MI
	`, roomID, uid).Scan(&data.ID, &data.Name, &data.Avatar, &data.IsGroup, &data.MarkAsRead, &data.Pinned, &data.HasMoreConvos)

	if len(data.ID) == 0 {
		return &data, &[]string{}, &[]string{}
	}

	// get convos
	convoIDs, convoMap, fileIDs, possibleUserIDs := GetInitConvos(roomID)
	data.Convo.IDs = *convoIDs
	data.Convo.Entities = *convoMap

	// get users
	userIDs, userMap := GetInitRoomUsers(roomID)
	data.User.IDs = *userIDs
	data.User.Entities = *userMap

	return &data, fileIDs, possibleUserIDs
}

func GetInitRoomUsers(roomID string) (*[]string, *map[string]RoomUser) {
	userMap := map[string]RoomUser{}
	userIDs := make([]string, 0)

	rows, err := database.DB.Query(`
		select
			UD.id,
			floor(extract(epoch from CU.last_seen) * 1000) as last_seen
		from
			chatrooms_users CU
		inner join
			user_details UD
		on
			CU.uid = UD.id
		where
			CU.rid = $1
	`, roomID)
	if err != nil {
		return &userIDs, &userMap
	}

	defer rows.Close()

	for rows.Next() {
		var u RoomUser
		rows.Scan(&u.ID, &u.LastSeen)
		userIDs = append(userIDs, u.ID)
		userMap[u.ID] = u
	}
	return &userIDs, &userMap
}

func GetInitConvos(roomID string) (*[]string, *map[string]Convo, *[]string, *[]string) {
	convoMap := map[string]Convo{}
	convoIDs := make([]string, 0)
	fileIDs := make([]string, 0)
	possibleUserIDs := make([]string, 0)

	rows, err := database.DB.Query(`
		SELECT 
			id,
			content,
			sender_id,
			floor(extract(epoch from dt) * 1000) as dtInt,
			coalesce(reply_msg_id::varchar,'') as reply_msg_id,
			coalesce(reply_msg,'') as reply_msg,
			coalesce(reply_msg_sender::varchar,'') as reply_msg_sender,
			coalesce(floor(extract(epoch from edit_time) * 1000),0) as edit_time,
			coalesce(files,array[]::varchar[]) as file_ids
		FROM
			chat_messages
		WHERE
			rid = $1
		ORDER BY
			dt DESC
		LIMIT 20 OFFSET 0
	`, roomID)
	if err != nil {
		return &convoIDs, &convoMap, &fileIDs, &possibleUserIDs
	}
	defer rows.Close()

	for rows.Next() {
		var c Convo
		rows.Scan(
			&c.ID,
			&c.Content,
			&c.Sender,
			&c.DateTime,
			&c.ReplyMsgID,
			&c.ReplyMsg,
			&c.ReplyMsgSender,
			&c.EditDateTime,
			(*pq.StringArray)(&c.FileIDs),
		)
		c.Error = false
		c.Sent = true
		convoIDs = append(convoIDs, c.ID)
		fileIDs = append(fileIDs, c.FileIDs...)
		possibleUserIDs = append(possibleUserIDs, c.Sender)
		if c.ReplyMsgSender != "" {
			possibleUserIDs = append(possibleUserIDs, c.ReplyMsgSender)
		}
		convoMap[c.ID] = c
	}
	return &convoIDs, &convoMap, &fileIDs, &possibleUserIDs
}

func GetInitRoomIDs(uid string) *[]string {
	result := *GetPinnedIDs(uid)

	if len(result) < 20 {
		remainingIDs := getRemainingIDs(uid, &result)
		result = append(result, *remainingIDs...)
	}

	return &result
}

func GetPinnedIDs(uid string) *[]string {
	result := make([]string, 0)
	rows, err := database.DB.Query(`
		select 
			CU.rid
		from 
			chatrooms_users CU
		inner join
			chat_messages CM
		on 
			CU.rid = CM.rid
		where
			CU.uid = $1
		and 
			CU.in_users_list
		and 
			CU.pinned
		group by
			CU.rid
		order by
			max(CM.dt) desc
	`, uid)
	if err != nil {
		return &result
	}

	defer rows.Close()

	for rows.Next() {
		var s string
		rows.Scan(&s)
		result = append(result, s)
	}
	return &result
}

func getRemainingIDs(uid string, pinnedIDs *[]string) *[]string {
	result := make([]string, 0)

	rows, err := database.DB.Query(`
		select 
			CU.rid
		from 
			chatrooms_users CU
		inner join
			chat_messages CM
		on 
			CU.rid = CM.rid
		where
			CU.uid = $1
		and 
			CU.in_users_list
		and 
			CU.rid <> all($2)
		group by
			CU.rid
		order by
			max(CM.dt) desc
		limit $3 offset 0
	`, uid, pq.Array(*pinnedIDs), 20-len(*pinnedIDs))
	if err != nil {
		return &result
	}

	defer rows.Close()

	for rows.Next() {
		var s string
		rows.Scan(&s)
		result = append(result, s)
	}
	return &result
}

package chat

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/cindyhont/projmgmt-backend/rest/common"
	"github.com/cindyhont/projmgmt-backend/rest/googleservice"
	"github.com/julienschmidt/httprouter"
	"github.com/lib/pq"
)

func fetchMoreConvos(
	w http.ResponseWriter,
	r *http.Request,
	p httprouter.Params,
	s *model.Session,
	signedIn bool,
	uid string,
) {
	if !signedIn {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	data := struct {
		Convos        []Convo              `json:"convos"`
		HasMoreConvos bool                 `json:"hasMoreConvos"`
		Files         []googleservice.File `json:"files"`
		Users         []model.UserDetails  `json:"users"`
	}{
		Convos:        make([]Convo, 0),
		HasMoreConvos: true,
		Files:         make([]googleservice.File, 0),
		Users:         make([]model.UserDetails, 0),
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	var req struct {
		RoomID      string `json:"roomID"`
		LastConvoID string `json:"lastConvoID"`
	}

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	fileIDs := make([]string, 0)
	userIDs := make([]string, 0)
	convoIDs := make([]string, 0)

	rows, err := database.DB.Query(`
		with all_convo_ids as (
			select 
				id,
				dt,
				row_number() over (order by dt desc) as rn
			from 
				chat_messages 
			where 
				rid = $1
			order by
				dt desc
		),
		get_oldest_rn as (
			select
				rn
			from
				all_convo_ids
			where 
				id = $2
		),
		ids_to_fetch as (
			select 
				AL.id
			from
				all_convo_ids AL,
				get_oldest_rn RN
			where
				AL.rn > RN.rn
			order by 
				AL.rn
			LIMIT 20
		)
		select
			CM.id,
			CM.content,
			CM.sender_id,
			floor(extract(epoch from CM.dt) * 1000) as dtInt,
			coalesce(CM.reply_msg_id::varchar,'') as reply_msg_id,
			coalesce(CM.reply_msg,'') as reply_msg,
			coalesce(CM.reply_msg_sender::varchar,'') as reply_msg_sender,
			coalesce(floor(extract(epoch from CM.edit_time) * 1000),0) as edit_time,
			coalesce(CM.files,array[]::varchar[]) as file_ids
		from
			chat_messages CM
		inner join
			ids_to_fetch IDS
		on
			CM.id = IDS.id
	`, req.RoomID, req.LastConvoID)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
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
		if len(c.FileIDs) != 0 {
			fileIDs = append(fileIDs, c.FileIDs...)
		}
		userIDs = append(userIDs, c.Sender)
		if c.ReplyMsgSender != "" {
			userIDs = append(userIDs, c.ReplyMsgSender)
		}
		convoIDs = append(convoIDs, c.ID)
		data.Convos = append(data.Convos, c)
	}

	if len(fileIDs) != 0 {
		data.Files = *googleservice.FetchFilesInfo(common.UniqueStringFromSlice(&fileIDs))
	}
	data.Users = *getMoreUserDetails(common.UniqueStringFromSlice(&userIDs))

	if len(data.Convos) < 20 {
		data.HasMoreConvos = false
	} else {
		data.HasMoreConvos = checkHasMoreConvos(&convoIDs, req.RoomID)
	}

	json.NewEncoder(w).Encode(data)
}

func checkHasMoreConvos(convoIDs *[]string, roomID string) bool {
	var result bool
	database.DB.QueryRow(`
		with all_ids as (
			select
				id
			from 
				chat_messages
			where
				rid = $1
			order by
				dt asc
			limit 1
		)
		select id <> all($2) from all_ids
	`, roomID, pq.Array(convoIDs)).Scan(&result)
	return result
}

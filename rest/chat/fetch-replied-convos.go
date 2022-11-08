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

func fetchRepliedConvos(
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
		RoomID   string   `json:"roomID"`
		ConvoIDs []string `json:"convoIDs"`
	}

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	fileIDs := make([]string, 0)
	userIDs := make([]string, 0)
	convoIDs := make([]string, 0)

	rows, err := database.DB.Query(`
		with all_ids as (
			select
				id,
				row_number() over (order by dt) as rn
			from 
				chat_messages
			where
				rid = $1
		),
		row_numbers as (
			select 
				rn
			from 
				all_ids
			where
				id = any($2)
		),
		min_max as (
			select 
				min(rn) as min_rn, 
				max(rn) as max_rn 
			from 
				row_numbers
		),
		selected_ids as (
			select
				AL.id
			from 
				all_ids AL,
				min_max MM
			where
				AL.rn < MM.max_rn
			and
				AL.rn > greatest(0,MM.min_rn - 20)
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
			selected_ids SI
		on
			CM.id = SI.id
	`, req.RoomID, pq.Array(req.ConvoIDs))
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
	data.HasMoreConvos = checkHasMoreConvos(&convoIDs, req.RoomID)

	json.NewEncoder(w).Encode(data)
}

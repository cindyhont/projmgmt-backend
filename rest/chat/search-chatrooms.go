package chat

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/julienschmidt/httprouter"
)

type roomInfo struct {
	RoomID  string `json:"rid"`
	Avatar  string `json:"avatar"`
	Name    string `json:"name"`
	IsGroup bool   `json:"isGroup"`
}

func searchChatroom(
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

	type userInfo struct {
		ID        string `json:"uid"`
		Avatar    string `json:"avatar"`
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
	}

	type result struct {
		Rooms []roomInfo `json:"rooms"`
		Users []userInfo `json:"users"`
	}

	data := result{
		Rooms: make([]roomInfo, 0),
		Users: make([]userInfo, 0),
	}

	encodedQuery := p.ByName("querystring")
	query, err := url.QueryUnescape(encodedQuery)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	query = strings.Join(strings.Split(strings.Trim(query, " "), " "), " & ") + ":*"

	hasRooms := true

	rows, err := database.DB.Query(`
		select 
			distinct C.id,
			coalesce(C.room_name,btrim(concat(UD.first_name,' ',UD.last_name))) as name,
			coalesce(C.avatar,UD.avatar,'') as avatar,
			C.room_name notnull as isgroup,
			case
				when C.room_name is null then CU.uid::varchar
				else ''
			end as uid,
			ts_rank_cd(C.tsv_w_position,query) qc,
			ts_rank_cd(UD.tsv,query) qud
		from 
			to_tsquery($1) query,
			chatrooms C
		inner join
			chatrooms_users CU
		on 
			C.id = CU.rid
		inner join
			user_details UD
		on
			CU.uid = UD.id
		inner join 
			departments D
		on
			UD.department_id = D.id
		where
			CU.uid <> $2
		and 
			(
				$1::tsquery @@ C.tsv
				or
				C.room_name is null AND query @@ UD.tsv
			)
		ORDER BY 
			qc,qud
		LIMIT 5
	`, query, uid)
	if err != nil {
		fmt.Println(err)
		hasRooms = false
	}

	uids := make([]string, 0)

	if hasRooms {
		defer rows.Close()

		for rows.Next() {
			var r roomInfo
			var userID string
			var qc, qud float32
			rows.Scan(&r.RoomID, &r.Name, &r.Avatar, &r.IsGroup, &userID, &qc, &qud)

			if userID != "" {
				uids = append(uids, userID)
			}
			data.Rooms = append(data.Rooms, r)
		}

		if len(data.Rooms) == 10 {
			json.NewEncoder(w).Encode(data)
			return
		}
	}

	rows, err = database.DB.Query(`
		SELECT
			UD.id,
			coalesce(UD.avatar,'') AS avatars,
			UD.first_name,
			UD.last_name
		FROM
			user_details UD,
			to_tsquery($1) query
		INNER JOIN
			departments D
		ON
			UD.department_id = D.id
		WHERE
			UD.id <> $2
		AND
			UD.date_registered_dt is not null
		AND
			UD.id <> all($3)
		AND
			query @@ UD.tsv
		limit %d
	`, query, uid, uids)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var user userInfo
		err = rows.Scan(&user.ID, &user.Avatar, &user.FirstName, &user.LastName)
		if err != nil {
			json.NewEncoder(w).Encode(data)
			return
		}
		data.Users = append(data.Users, user)
	}
	json.NewEncoder(w).Encode(data)
}

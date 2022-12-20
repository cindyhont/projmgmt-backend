package ws

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/instantcomm"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/julienschmidt/httprouter"
)

func fetchOldWsMessages(
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

	data := make([]instantcomm.Response, 0)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	var req struct {
		LastWebsocketOfflineTime int64 `json:"lastWebsocketOfflineTime"`
	}

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	instantcomm.CleanOldWsRecords()

	rows, err := database.DB.Query(`
		SELECT
			C.action_type,
			C.payload
		FROM
			ws_message_content C
		INNER JOIN
			ws_message_to T
		ON
			C.id = T.message_id
		WHERE
			C.dt > $1
		AND
			(
				to_all_recipients = TRUE
				OR
				T.uid = $2
			)
		ORDER BY
			C.dt
	`, time.UnixMilli(req.LastWebsocketOfflineTime), uid)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var r instantcomm.Response
		var s string

		rows.Scan(
			&r.Type,
			&s,
		)

		json.Unmarshal([]byte(s), &r.Payload)

		data = append(data, r)
	}

	json.NewEncoder(w).Encode(data)
}

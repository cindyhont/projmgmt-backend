package ws

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/cindyhont/projmgmt-backend/websocket"
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

	data := make([]websocket.Response, 0)

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

	websocket.CleanOldWsRecords()

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
		var r websocket.Response
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

/*

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
		C.dt > current_timestamp - '1 hour'::interval
	AND
		(
			to_all_recipients = TRUE
			OR
			T.uid = $1
		)
	ORDER BY
		C.dt
`, uid)

*/

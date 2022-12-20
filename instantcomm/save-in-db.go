package instantcomm

import (
	"encoding/json"
	"fmt"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/lib/pq"
)

func SaveWsMessageInDB(res *Response, recipients *[]string) string {
	jsonInBytes, err := json.Marshal(res.Payload)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	var reqID string

	err = database.DB.QueryRow(
		"INSERT INTO ws_message_content (action_type, payload, to_all_recipients) VALUES ($1,$2,$3) RETURNING id",
		res.Type,
		string(jsonInBytes),
		res.ToAllRecipients,
	).Scan(&reqID)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	if len(*recipients) == 0 {
		return reqID
	}

	txn, err := database.DB.Begin()
	if err != nil {
		return ""
	}

	stmt, err := txn.Prepare(pq.CopyIn("ws_message_to", "message_id", "uid"))
	if err != nil {
		return ""
	}

	for _, uid := range *recipients {
		_, err = stmt.Exec(reqID, uid)
		if err != nil {
			return ""
		}
	}

	_, err = stmt.Exec()
	if err != nil {
		return ""
	}

	err = stmt.Close()
	if err != nil {
		return ""
	}

	err = txn.Commit()
	if err != nil {
		return ""
	}

	return reqID
}

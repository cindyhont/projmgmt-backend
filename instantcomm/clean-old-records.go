package instantcomm

import "github.com/cindyhont/projmgmt-backend/database"

func CleanOldWsRecords() {
	database.DB.Exec("DELETE FROM ws_message_content WHERE dt < current_timestamp - '1 hour'::interval")
	database.DB.Exec("DELETE FROM ws_message_to WHERE message_id NOT IN (SELECT id FROM ws_message_content)")
}

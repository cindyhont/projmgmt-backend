package chat

import "github.com/cindyhont/projmgmt-backend/database"

func getChatRoomUserIDs(rid string) *[]string {
	userIDs := make([]string, 0)

	rows, err := database.DB.Query("SELECT uid FROM chatrooms_users WHERE rid = $1", rid)
	if err != nil {
		return &userIDs
	}

	defer rows.Close()

	for rows.Next() {
		var s string
		rows.Scan(&s)
		userIDs = append(userIDs, s)
	}

	return &userIDs
}

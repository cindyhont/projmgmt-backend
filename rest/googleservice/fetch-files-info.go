package googleservice

import (
	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/lib/pq"
)

func FetchFilesInfo(fileIDs *[]string) *[]File {
	result := make([]File, 0)
	rows, err := database.DB.Query("SELECT id, name, size FROM files WHERE id=ANY($1)", pq.Array(fileIDs))
	if err != nil {
		return &result
	}

	defer rows.Close()

	for rows.Next() {
		var f File
		rows.Scan(&f.ID, &f.Name, &f.Size)
		result = append(result, f)
	}
	return &result
}

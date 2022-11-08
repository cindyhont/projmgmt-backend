package miscfunc

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/julienschmidt/httprouter"
	"github.com/lib/pq"
)

func fetchUsers(
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

	data := make([]User, 0)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	var req struct {
		UserIDs []string `json:"uids"`
	}

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	data = *FetchUsers(&req.UserIDs)

	json.NewEncoder(w).Encode(data)
}

func FetchUsers(userIDs *[]string) *[]User {
	result := make([]User, 0)
	rows, err := database.DB.Query(`
		SELECT
			id,
			first_name,
			last_name,
			coalesce(avatar,'') as avatar
		FROM
			user_details
		WHERE
			id = ANY($1)
	`, pq.Array(userIDs))
	if err != nil {
		return &result
	}

	defer rows.Close()

	for rows.Next() {
		var u User
		rows.Scan(&u.ID, &u.FirstName, &u.LastName, &u.Avatar)
		result = append(result, u)
	}
	return &result
}

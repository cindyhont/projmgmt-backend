package miscfunc

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/julienschmidt/httprouter"
	"github.com/lib/pq"
)

func searchUser(
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
		Query       string   `json:"query"`
		ExcludeUIDs []string `json:"exclude"`
	}

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	insertStr := ""
	vars := make([]interface{}, 0)
	vars = append(vars, strings.Join(strings.Split(strings.Trim(req.Query, " "), " "), " & ")+":*")

	if len(req.ExcludeUIDs) != 0 {
		vars = append(vars, pq.Array(req.ExcludeUIDs))
		insertStr = "and UD.id <> ALL($2)"
	}

	rows, err := database.DB.Query(fmt.Sprintf(`
		select
			UD.id,
			UD.first_name,
			UD.last_name,
			coalesce(UD.avatar,'') as avatar
		from
			to_tsquery($1) query,
			user_details UD
		inner join
			users U
		on
			UD.id = U.id
		where
			UD.date_registered_dt is not null
		and
			U.authorized
		and
			query @@ UD.tsv
		%s
		order by
			ts_rank_cd(UD.tsv,query) desc
		limit 10
	`, insertStr), vars...)
	if err != nil {
		fmt.Println(err)
		json.NewEncoder(w).Encode(data)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var u User
		rows.Scan(&u.ID, &u.FirstName, &u.LastName, &u.Avatar)
		data = append(data, u)
	}

	json.NewEncoder(w).Encode(data)
}

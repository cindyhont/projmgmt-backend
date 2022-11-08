package staff

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
)

func GetDepartment(
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

	id := p.ByName("id")

	if id == uuid.Nil.String() {
		json.NewEncoder(w).Encode(map[string]string{"name": "(No Department)"})
		return
	}

	var name string
	database.DB.QueryRow("SELECT name FROM departments WHERE id = $1", id).Scan(&name)
	json.NewEncoder(w).Encode(map[string]string{"name": name})
}

func SearchDepartment(
	w http.ResponseWriter,
	r *http.Request,
	p httprouter.Params,
) {
	type resultStruct struct {
		ID    string `json:"id"`
		Label string `json:"label"`
	}
	data := make([]resultStruct, 0)

	encodedQuery := p.ByName("querystring")
	if encodedQuery == "" {
		json.NewEncoder(w).Encode(data)
		return
	}

	query, err := url.QueryUnescape(encodedQuery)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	rows, err := database.DB.Query(`
		SELECT 
			id, 
			name
		FROM
			departments, 
			to_tsquery($1) query
		WHERE 
			query @@ tsv
		ORDER BY 
			ts_rank_cd(tsv,query) DESC
		LIMIT 5
	`, strings.Join(strings.Split(query, " "), " & ")+":*")
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var i resultStruct
		err = rows.Scan(&i.ID, &i.Label)
		if err != nil {
			json.NewEncoder(w).Encode(data)
			return
		}
		data = append(data, i)
	}

	json.NewEncoder(w).Encode(data)
}

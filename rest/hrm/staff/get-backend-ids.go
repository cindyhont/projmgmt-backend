package staff

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/cindyhont/projmgmt-backend/rest/hrm/hrmcommon"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
)

func GetBackendIDs(
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

	data := make([]string, 0)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	var filters hrmcommon.FilterCollection
	if err = json.Unmarshal(body, &filters); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	json.NewEncoder(w).Encode(fetchIDs(&filters))
}

func fetchIDs(filters *hrmcommon.FilterCollection) *[]string {
	ids := make([]string, 0)
	conditionStr, variables := getConditionString(filters, 2, "AND ")
	vars := make([]interface{}, 0)
	vars = append(vars, uuid.Nil.String())
	vars = append(vars, variables...)

	rows, err := database.DB.Query(fmt.Sprintf(`
		SELECT 
			UD.id
		FROM 
			user_details UD 
		INNER JOIN 
			users U
		ON
			UD.id = U.id
		WHERE 
			U.authorized = true 
		AND 
			UD.id<>$1 %s
	`, conditionStr), vars...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		err = rows.Scan(&id)
		if err != nil {
			return nil
		}
		ids = append(ids, id)
	}
	return &ids
}

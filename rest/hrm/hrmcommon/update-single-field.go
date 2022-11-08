package hrmcommon

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
)

func UpdateSingleField(
	w http.ResponseWriter,
	r *http.Request,
	p httprouter.Params,
	s *model.Session,
	signedIn bool,
	uid string,
) {
	tableName := p.ByName("tableName")

	if !signedIn || !TableNameOK(tableName) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	data := struct {
		Success bool `json:"success"`
	}{
		Success: false,
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	var request struct {
		Field string      `json:"field"`
		Value interface{} `json:"value"`
		ID    string      `json:"id"`
	}
	if err = json.Unmarshal(body, &request); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	if _, err := database.DB.Exec(fmt.Sprintf("UPDATE %s SET %s = $1 WHERE id = $2", tableName, request.Field), request.Value, request.ID); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	if tableName == "departments" {
		_, err = database.DB.Exec("UPDATE departments SET tsv = to_tsvector(concat(internal_id,' ',name)) where id = $1", request.ID)
		if err != nil {
			json.NewEncoder(w).Encode(data)
			return
		}

		_, err = database.DB.Exec(`
			update 
				user_details UD 
			set 
				tsv = to_tsvector(concat(
					UD.staff_id,
					' ',
					UD.first_name,
					' ',
					UD.last_name,
					' ',
					UD.title,
					' ',
					D.internal_id,
					' ',
					D.name
				))
			from 
				departments D
			where 
				UD.department_id = D.id
			and 
				UD.department_id = $1
		`, request.ID)
		if err != nil {
			json.NewEncoder(w).Encode(data)
			return
		}
	} else {
		_, err = database.DB.Exec(`
			update 
				user_details UD 
			set 
				tsv = to_tsvector(concat(
					UD.staff_id,
					' ',
					UD.first_name,
					' ',
					UD.last_name,
					' ',
					UD.title,
					' ',
					D.internal_id,
					' ',
					D.name
				))
			from 
				departments D
			where 
				UD.department_id = D.id
			and 
				UD.id = $1
			and 
				UD.department_id <> $2
		`, request.ID, uuid.Nil.String())
		if err != nil {
			json.NewEncoder(w).Encode(data)
			return
		}

		_, err = database.DB.Exec(`
			update 
				user_details
			set 
				tsv = to_tsvector(concat(
					staff_id,
					' ',
					first_name,
					' ',
					last_name,
					' ',
					title
				))
			where 
				id = $1
			and
				department_id = $2
		`, request.ID, uuid.Nil.String())
		if err != nil {
			json.NewEncoder(w).Encode(data)
			return
		}
	}

	data.Success = true
	json.NewEncoder(w).Encode(data)
}

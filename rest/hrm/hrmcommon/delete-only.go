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
	"github.com/lib/pq"
)

func DeleteOnly(
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

	data := struct {
		Success bool `json:"success"`
	}{
		Success: false,
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var request struct {
		Table string   `json:"table"`
		IDs   []string `json:"ids"`
	}
	if err = json.Unmarshal(body, &request); err != nil {
		fmt.Println(err)
		json.NewEncoder(w).Encode(data)
		return
	}

	if request.Table != "departments" && request.Table != "user_details" {
		json.NewEncoder(w).Encode(data)
		return
	}

	if request.Table == "departments" {
		database.DB.Exec("UPDATE user_details SET department_id = $1 WHERE department_id = ANY($2)", uuid.Nil.String(), pq.Array(request.IDs))
	} else {
		database.DB.Exec("UPDATE user_details SET supervisor_id = $1 WHERE supervisor_id = ANY($2)", uuid.Nil.String(), pq.Array(request.IDs))
	}

	if request.Table == "user_details" {
		database.DB.Exec("UPDATE users SET authorized = false WHERE id = ANY($1)", pq.Array(request.IDs))
	} else {
		database.DB.Exec(fmt.Sprintf("DELETE FROM %s WHERE id = ANY($1)", request.Table), pq.Array(request.IDs))
	}

	data.Success = true
	json.NewEncoder(w).Encode(data)
}

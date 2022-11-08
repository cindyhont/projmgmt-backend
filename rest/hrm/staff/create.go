package staff

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/cindyhont/projmgmt-backend/rest/hrm/hrmcommon"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
)

func ActiveCreate(
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
		InList bool   `json:"inList"`
		ID     string `json:"id"`
	}{
		InList: false,
		ID:     "",
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	var req struct {
		Staff_ID      string                     `json:"staffID"`
		FirstName     string                     `json:"firstName"`
		LastName      string                     `json:"lastName"`
		Title         string                     `json:"title"`
		Department_ID string                     `json:"departmentID"`
		Supervisor_ID string                     `json:"supervisorID"`
		UserRight     int                        `json:"userRight"`
		Email         string                     `json:"email"`
		Filters       hrmcommon.FilterCollection `json:"filters"`
	}

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	database.DB.QueryRow("INSERT INTO users (id, authorized) VALUES (default,default) RETURNING id").Scan(&data.ID)

	tsvArr := []string{
		req.Staff_ID,
		req.FirstName,
		req.LastName,
		req.Title,
	}
	if req.Department_ID != "" && req.Department_ID != uuid.Nil.String() {
		var deptInternalID, deptName string
		err = database.DB.QueryRow("SELECT internal_id, name FROM departments WHERE id = $1", req.Department_ID).Scan(&deptInternalID, &deptName)
		if err != nil {
			fmt.Println(err)
			json.NewEncoder(w).Encode(data)
			return
		}
		tsvArr = append(tsvArr, deptInternalID, deptName)
	}

	_, err = database.DB.Exec(`
		INSERT INTO user_details (
			id,
			staff_id,
			first_name,
			last_name,
			title,
			department_id,
			supervisor_id,
			user_right,
			email,
			tsv
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,to_tsvector($10))
	`,
		data.ID,
		req.Staff_ID,
		req.FirstName,
		req.LastName,
		req.Title,
		req.Department_ID,
		req.Supervisor_ID,
		req.UserRight,
		req.Email,
		strings.Join(tsvArr, " "),
	)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		database.DB.Exec("DELETE FROM users WHERE id = $1", data.ID)
		return
	}

	conditionStr, variables := getConditionString(&req.Filters, 2, " AND ")

	vars := make([]interface{}, 0)
	vars = append(vars, data.ID)
	vars = append(vars, variables...)

	err = database.DB.QueryRow(fmt.Sprintf("SELECT EXISTS (SELECT * FROM user_details WHERE id=$1 %s)", conditionStr), vars...).Scan(&data.InList)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	json.NewEncoder(w).Encode(data)
}

func PassiveCreate(
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

	var data Staff

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

	id := p.ByName("id")

	conditionStr, variables := getConditionString(&filters, 2, " AND ")

	vars := make([]interface{}, 0)
	vars = append(vars, id)
	vars = append(vars, variables...)

	var exists bool
	err = database.DB.QueryRow(fmt.Sprintf("SELECT EXISTS (SELECT * FROM user_details WHERE id=$1 %s)", conditionStr), vars...).Scan(&exists)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	if !exists {
		json.NewEncoder(w).Encode(data)
		return
	}

	err = database.DB.QueryRow(`
		SELECT 
			id,
			staff_id,
			first_name,
			last_name,
			title,
			department_id,
			supervisor_id,
			user_right,
			email,
			floor(COALESCE(extract(epoch from last_invite_dt) * 1000,0)::numeric) as last_invite_dt,
			floor(COALESCE(extract(epoch from date_registered_dt) * 1000,0)::numeric) as date_registered_dt,
			floor(COALESCE(extract(epoch from last_active_dt) * 1000,0)::numeric) as last_active_dt
		FROM 
			user_details
		WHERE
			id = $1
	`, id).Scan(
		&data.ID,
		&data.StaffID,
		&data.FirstName,
		&data.LastName,
		&data.Title,
		&data.Department,
		&data.SupervisorID,
		&data.UserRight,
		&data.Email,
		&data.LastInviteDT,
		&data.DateRegistered,
		&data.LastActive,
	)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	json.NewEncoder(w).Encode(data)
}

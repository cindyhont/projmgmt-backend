package dept

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/cindyhont/projmgmt-backend/rest/hrm/hrmcommon"
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
		// SignedIn bool   `json:"signedIn"`
		InList bool   `json:"inList"`
		ID     string `json:"id"`
	}{
		// SignedIn: signedIn,
		InList: false,
		ID:     "",
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	var request struct {
		Internal_ID string                     `json:"internal_id"`
		Name        string                     `json:"name"`
		Filters     hrmcommon.FilterCollection `json:"filters"`
	}

	if err = json.Unmarshal(body, &request); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	err = database.DB.QueryRow(`
		INSERT INTO departments (internal_id, name, tsv) 
		VALUES ($1,$2,$3) RETURNING id`, request.Internal_ID, request.Name, strings.Join([]string{request.Internal_ID, request.Name}, " ")).Scan(&data.ID)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	conditionStr, variables := getConditionString(&request.Filters, 2, " AND ")

	vars := make([]interface{}, 0)
	vars = append(vars, data.ID)
	vars = append(vars, variables...)

	err = database.DB.QueryRow(fmt.Sprintf("SELECT EXISTS (SELECT 1 FROM departments WHERE id=$1 %s)", conditionStr), vars...).Scan(&data.InList)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	// data.WsID, err = hrm.CacheMessage(&hrm.Request{
	// 	ID:     data.ID,
	// 	Action: hrm.CREATE,
	// 	Table:  hrm.DEPARTMENTS,
	// })
	// if err != nil {
	// 	fmt.Println(err)
	// 	json.NewEncoder(w).Encode(data)
	// 	return
	// }

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

	var data struct {
		Internal_ID string `json:"internal_id"`
		Name        string `json:"name"`
	}

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
	err = database.DB.QueryRow(fmt.Sprintf("SELECT EXISTS (SELECT * FROM departments WHERE id=$1 %s)", conditionStr), vars...).Scan(&exists)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	if !exists {
		json.NewEncoder(w).Encode(data)
		return
	}

	err = database.DB.QueryRow("SELECT internal_id, name FROM departments WHERE id = $1", id).Scan(&data.Internal_ID, &data.Name)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	json.NewEncoder(w).Encode(data)
}

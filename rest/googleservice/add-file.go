package googleservice

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/julienschmidt/httprouter"
)

func addFile(
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
		json.NewEncoder(w).Encode(data)
		return
	}

	var req File

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	_, err = database.DB.Exec("INSERT INTO files (id,name,size) VALUES ($1,$2,$3)", req.ID, req.Name, req.Size)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	data.Success = true
	json.NewEncoder(w).Encode(data)
}

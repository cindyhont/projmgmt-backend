package googleservice

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/julienschmidt/httprouter"
	"github.com/lib/pq"
)

func addFiles(
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

	var req []File

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	err = AddFiles(&req)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	data.Success = true
	json.NewEncoder(w).Encode(data)
}

func AddFiles(files *[]File) error {
	f := *files
	txn, err := database.DB.Begin()
	if err != nil {
		return err
	}

	var stmt *sql.Stmt
	if stmt, err = txn.Prepare(pq.CopyIn("files", "id", "name", "size")); err != nil {
		return err
	}

	for _, file := range f {
		if _, err = stmt.Exec(file.ID, file.Name, file.Size); err != nil {
			return err
		}
	}

	if _, err = stmt.Exec(); err != nil {
		return err
	}

	if err = stmt.Close(); err != nil {
		return err
	}

	if err = txn.Commit(); err != nil {
		return err
	}
	return nil
}

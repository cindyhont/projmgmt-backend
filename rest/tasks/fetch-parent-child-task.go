package tasks

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/julienschmidt/httprouter"
)

func fetchParentChildTasks(
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

	var data []Task

	body, err := io.ReadAll(r.Body)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	var req struct {
		TaskIDs []string `json:"taskIDs"`
	}

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}
	data = *GetTasks(&req.TaskIDs)

	json.NewEncoder(w).Encode(data)
}

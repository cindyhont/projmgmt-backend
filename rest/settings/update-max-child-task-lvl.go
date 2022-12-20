package settings

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/instantcomm"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/julienschmidt/httprouter"
)

func updateMaxChildTaskLvl(
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
		Success     bool   `json:"success"`
		WsRequestID string `json:"wsid"`
	}{
		Success:     false,
		WsRequestID: "",
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	var req struct {
		MaxChildTaskLvl int `json:"maxChildTaskLvl"`
	}

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	database.DB.Exec(
		"UPDATE user_details SET max_child_task_level = $1 WHERE id = $2",
		req.MaxChildTaskLvl,
		uid,
	)

	wsMessage := instantcomm.Response{
		Type: "settings_update-max-child-task-lvl",
		Payload: map[string]interface{}{
			"maxChildTaskLvl": req.MaxChildTaskLvl,
			"fromWS":          true,
		},
	}

	data.WsRequestID = instantcomm.SaveWsMessageInDB(&wsMessage, &[]string{uid})
	data.Success = true

	json.NewEncoder(w).Encode(data)
}

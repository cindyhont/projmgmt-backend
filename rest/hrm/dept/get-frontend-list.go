package dept

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/cindyhont/projmgmt-backend/rest/hrm/hrmcommon"
	"github.com/julienschmidt/httprouter"
)

func GetFrontendList(
	w http.ResponseWriter,
	r *http.Request,
	p httprouter.Params,
	s *model.Session,
	signedIn bool,
	uid string,
) {
	// fmt.Println(r.Header.Get("Origin")) <--- this prints http://localhost:3000

	if !signedIn {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	data := make([]Department, 0)

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

	json.NewEncoder(w).Encode(FetchDepartments(&filters))
}

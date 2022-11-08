package staff

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
	if !signedIn {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	data := make([]Staff, 0)

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

	json.NewEncoder(w).Encode(FetchStaff(&filters))
}

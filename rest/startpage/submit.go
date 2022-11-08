package startpage

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/cindyhont/projmgmt-backend/rest/common"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/lib/pq"
)

type dept_start struct {
	ID   string `json:"i"`
	Name string `json:"n"`
}

type staffDetails struct {
	ID            string
	InviteMailKey string
	StaffID       string `json:"id"`
	FirstName     string `json:"fn"`
	LastName      string `json:"ln"`
	Title         string `json:"t"`
	DeptID        string `json:"d"`
	Email         string `json:"e"`
	Supervisor    string `json:"s"`
	UserRights    int    `json:"ur"`
}

type starterPack struct {
	Departments  []dept_start   `json:"depts"`
	StaffDetails []staffDetails `json:"staffDetails"`
	AdminStaffID string         `json:"adminStaffID"`
}

func insertDepartments(departments *[]dept_start) (map[string]string, error) {
	var txn *sql.Tx
	var err error
	var deptMap = make(map[string]string)
	if txn, err = database.DB.Begin(); err != nil {
		return nil, err
	}

	var stmt *sql.Stmt
	if stmt, err = txn.Prepare(pq.CopyIn("departments", "id", "internal_id", "name")); err != nil {
		return nil, err
	}

	for _, dept := range *departments {
		id := uuid.New().String()
		deptMap[dept.ID] = id
		if _, err = stmt.Exec(id, dept.ID, dept.Name); err != nil {
			return nil, err
		}
	}

	if _, err = stmt.Exec(); err != nil {
		return nil, err
	}

	if err = stmt.Close(); err != nil {
		return nil, err
	}

	if err = txn.Commit(); err != nil {
		return nil, err
	}

	_, err = database.DB.Exec("UPDATE departments SET tsv = to_tsvector(concat(internal_id,' ',name)) where id <> $1", uuid.Nil.String())
	if err != nil {
		return nil, err
	}
	return deptMap, nil
}

func insertUsersTable(invitees *[]staffDetails, adminUid string) error {
	var txn *sql.Tx
	var err error
	if txn, err = database.DB.Begin(); err != nil {
		return err
	}

	var stmt *sql.Stmt
	if stmt, err = txn.Prepare(pq.CopyIn("users", "id")); err != nil {
		return err
	}

	for _, invitee := range *invitees {
		if invitee.ID == adminUid {
			continue
		}
		if _, err = stmt.Exec(invitee.ID); err != nil {
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

func uuidOrFake(s string, fake string) string {
	if s == "" {
		return fake
	} else {
		return s
	}
}

func insertUserDetails(users *[]staffDetails) error {
	fakeInviteMailKey := uuid.New().String()
	var txn *sql.Tx
	var err error
	if txn, err = database.DB.Begin(); err != nil {
		return err
	}

	var stmt *sql.Stmt
	if stmt, err = txn.Prepare(pq.CopyIn(
		"user_details",
		"id",
		"invitation_mail_key",
		"staff_id",
		"first_name",
		"last_name",
		"title",
		"department_id",
		"supervisor_id",
		"user_right",
		"email",
	)); err != nil {
		return err
	}

	for _, user := range *users {
		if _, err = stmt.Exec(
			user.ID,
			uuidOrFake(user.InviteMailKey, fakeInviteMailKey),
			user.StaffID,
			user.FirstName,
			user.LastName,
			user.Title,
			user.DeptID,
			uuidOrFake(user.Supervisor, uuid.Nil.String()),
			user.UserRights,
			user.Email,
		); err != nil {
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

	database.DB.Exec("UPDATE user_details SET invitation_mail_key = NULL WHERE invitation_mail_key = $1", fakeInviteMailKey)

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
			UD.department_id <> $1
	`, uuid.Nil.String())
	if err != nil {
		return err
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
			department_id = $1
		and 
			id <> $1
	`, uuid.Nil.String())
	if err != nil {
		return err
	}

	return nil
}

func submit(w http.ResponseWriter, r *http.Request, _ httprouter.Params, s *model.Session, signedIn bool, uid string) {
	if !signedIn {
		common.SendResponse(w, &model.Response{Session: s, Data: nil})
		return
	}

	res := struct {
		Success  bool   `json:"success"`
		SignedIn bool   `json:"signedIn"`
		Message  string `json:"message"`
		Error    string `json:"error"`
	}{
		Success:  false,
		SignedIn: true,
		Message:  "Failed to upload",
		Error:    "",
	}

	var pack starterPack
	body, err := io.ReadAll(r.Body)
	if err != nil {
		res.Error = err.Error()
		common.SendResponse(w, &model.Response{Session: s, Data: res})
		return
	}
	err = json.Unmarshal(body, &pack)
	if err != nil {
		res.Error = err.Error()
		common.SendResponse(w, &model.Response{Session: s, Data: res})
		return
	}

	deptMap, err := insertDepartments(&pack.Departments)
	if err != nil {
		res.Error = err.Error()
		common.SendResponse(w, &model.Response{Session: s, Data: res})
		return
	}

	var idMap = make(map[string]string) // key: internal staff id, value: uuid
	for i := range pack.StaffDetails {
		if pack.AdminStaffID == pack.StaffDetails[i].StaffID {
			pack.StaffDetails[i].ID = uid
			idMap[pack.StaffDetails[i].StaffID] = uid
		} else {
			id := uuid.New().String()
			pack.StaffDetails[i].ID = id
			pack.StaffDetails[i].InviteMailKey = uuid.New().String()
			idMap[pack.StaffDetails[i].StaffID] = id
		}
		pack.StaffDetails[i].DeptID = deptMap[pack.StaffDetails[i].DeptID]
	}

	for j := range pack.StaffDetails {
		pack.StaffDetails[j].Supervisor = idMap[pack.StaffDetails[j].Supervisor]
	}

	if err = insertUsersTable(&pack.StaffDetails, uid); err != nil {
		res.Error = err.Error()
		res.Message = "Failed to upload staff IDs"
		common.SendResponse(w, &model.Response{Session: s, Data: res})
		return
	}

	// json.NewEncoder(w).Encode(pack.StaffDetails)

	if err = insertUserDetails(&pack.StaffDetails); err != nil {
		res.Error = err.Error()
		res.Message = "Failed to upload staff details"
		common.SendResponse(w, &model.Response{Session: s, Data: res})
		return
	}

	res.Error = ""
	res.Message = ""
	res.Success = true
	common.SendResponse(w, &model.Response{Session: s, Data: res})
}

package staff

type Staff struct {
	ID             string `json:"id"`
	StaffID        string `json:"staff_id"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	Title          string `json:"title"`
	Department     string `json:"department_id"`
	SupervisorID   string `json:"supervisor_id"`
	UserRight      int    `json:"user_right"`
	Email          string `json:"email"`
	LastInviteDT   int64  `json:"last_invite_dt"`
	DateRegistered int64  `json:"date_registered_dt"`
	LastActive     int64  `json:"last_active_dt"`
}

type Supervisor struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

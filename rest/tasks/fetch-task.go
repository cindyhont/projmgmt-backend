package tasks

import (
	"encoding/json"
	"net/http"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/cindyhont/projmgmt-backend/rest/googleservice"
	"github.com/cindyhont/projmgmt-backend/rest/miscfunc"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/lib/pq"
)

func fetchTask(
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
		Task        map[string]interface{} `json:"task"`
		Comments    []Comment              `json:"comments"`
		TimeRecords []TimeRecord           `json:"timeRecords"`
		UserDetails []miscfunc.User        `json:"users"`
		Files       []googleservice.File   `json:"files"`
	}{}

	taskID := p.ByName("task-id")

	_, err := uuid.Parse(taskID)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	_taskIDs := []string{taskID}
	userIDs := make([]string, 0)

	_task := (*GetTasks(&_taskIDs))[0]
	taskBytes, err := json.Marshal(_task)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}
	json.Unmarshal(taskBytes, &data.Task)

	var extraFieldStr string
	database.DB.QueryRow("SELECT values FROM task_custom_user_field_values WHERE uid=$1 AND task_id=$2", uid, taskID).Scan(&extraFieldStr)

	if extraFieldStr != "" {
		var extraFieldMap map[string]interface{}
		json.Unmarshal([]byte(extraFieldStr), &extraFieldMap)

		for k, v := range extraFieldMap {
			data.Task[k] = v
		}
	}

	/********************************/

	if len(_task.Supervisors) != 0 {
		userIDs = append(userIDs, _task.Supervisors...)
	}
	if len(_task.Participants) != 0 {
		userIDs = append(userIDs, _task.Participants...)
	}
	if len(_task.Viewers) != 0 {
		userIDs = append(userIDs, _task.Viewers...)
	}
	userIDs = append(userIDs, _task.Owner)

	data.UserDetails = *miscfunc.FetchUsers(&userIDs)
	if len(_task.FileIDs) != 0 {
		data.Files = *googleservice.FetchFilesInfo(&_task.FileIDs)
	}

	data.Comments = *GetComments(&_taskIDs)
	for _, c := range data.Comments {
		userIDs = append(userIDs, c.Sender, c.ReplyMsgSender)
	}

	data.TimeRecords = *GetTimeRecords(&_taskIDs)
	for _, r := range data.TimeRecords {
		userIDs = append(userIDs, r.UserID)
	}

	json.NewEncoder(w).Encode(data)
}

func GetTasks(taskIDs *[]string) *[]Task {
	result := make([]Task, 0)
	rows, err := database.DB.Query(`
		SELECT
			id,
			name,
			coalesce(description,''),
			floor(extract(epoch from create_dt) * 1000) as create_dt,
			coalesce(floor(extract(epoch from start_dt) * 1000),0) as start_dt,
			coalesce(floor(extract(epoch from deadline_dt) * 1000),0) as deadline_dt,
			owner,
			coalesce(supervisors,array[]::uuid[]) as supervisors,
			coalesce(participants,array[]::uuid[]) as participants,
			coalesce(viewers,array[]::uuid[]) as viewers,
			string_to_array(ltree2text(parents),'.')::uuid[],
			hourly_rate,
			track_time,
			coalesce(files,array[]::varchar[]) as files,
			is_group_task,
			assignee
		FROM
			tasks
		WHERE
			id = ANY($1)
		AND
			deleted = false
	`, pq.Array(taskIDs))
	if err != nil {
		return &result
	}

	defer rows.Close()

	for rows.Next() {
		var t Task
		rows.Scan(
			&t.ID,
			&t.Name,
			&t.Description,
			&t.CreateDT,
			&t.StartDT,
			&t.DeadlineDT,
			&t.Owner,
			pq.Array(&t.Supervisors),
			pq.Array(&t.Participants),
			pq.Array(&t.Viewers),
			pq.Array(&t.Parents),
			&t.HourlyRate,
			&t.TrackTime,
			pq.Array(&t.FileIDs),
			&t.IsGroupTask,
			&t.Assignee,
		)
		t.Sent = true
		t.FilesToDelete = make([]string, 0)
		result = append(result, t)
	}
	return &result
}

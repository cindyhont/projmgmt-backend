package tasks

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/julienschmidt/httprouter"
	"github.com/lib/pq"
)

func searchTasks(
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

	data := make([]Task, 0)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	var req struct {
		Query      string   `json:"query"`
		ExcludeIDs []string `json:"exclude"`
	}

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	query := strings.Join(strings.Split(strings.Trim(req.Query, " "), " "), " & ") + ":*"

	insertStr := ""
	vars := make([]interface{}, 0)
	vars = append(vars, uid, query)

	if len(req.ExcludeIDs) != 0 {
		insertStr = "AND id <> ALL($3)"
		vars = append(vars, pq.Array(req.ExcludeIDs))
	}

	rows, err := database.DB.Query(fmt.Sprintf(`
		WITH is_visitor AS (
			SELECT visitor as v FROM user_details WHERE id = $1
		),
		get_users as (
			SELECT DISTINCT uid FROM chatrooms_users WHERE rid IN (
				SELECT 
					CU.rid 
				FROM 
					chatrooms_users CU
				INNER JOIN
					chatrooms C
				ON
					C.id = CU.rid
				WHERE 
					CU.uid = $1 
				AND 
					C.deleted = false
			)
		)
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
			approval,
			assignee
		FROM
			to_tsquery($2) query,
			tasks,
			is_visitor IV
		WHERE
			query @@ tsv
		AND
			deleted = false
		AND
			(
				CASE 
					WHEN IV.v THEN tasks.owner IN (SELECT uid FROM get_users)
					ELSE TRUE
				END
			)
		%s
		ORDER BY
			ts_rank_cd(tsv,query) desc
		LIMIT 5
	`, insertStr), vars...)
	if err != nil {
		fmt.Println(err)
		json.NewEncoder(w).Encode(data)
		return
	}

	/*
		rows, err := database.DB.Query(fmt.Sprintf(`
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
				approval,
				assignee
			FROM
				to_tsquery($1) query,
				tasks
			WHERE
				query @@ tsv
			AND
				deleted = false
			%s
			ORDER BY
				ts_rank_cd(tsv,query) desc
			LIMIT 5
		`, insertStr), vars...)
		if err != nil {
			fmt.Println(err)
			json.NewEncoder(w).Encode(data)
			return
		}
	*/

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
			&t.Approval,
			&t.Assignee,
		)
		t.Sent = true
		t.FilesToDelete = make([]string, 0)
		data = append(data, t)
	}
	json.NewEncoder(w).Encode(data)
}

package tasks

import (
	"github.com/cindyhont/projmgmt-backend/database"
)

func getTaskUserIDs(taskID string) *[]string {
	userIDs := make([]string, 0)
	rows, err := database.DB.Query(`
		select 
			distinct unnest(coalesce(supervisors,ARRAY[]::uuid[]) 
				|| coalesce(participants,ARRAY[]::uuid[]) 
				|| coalesce(viewers,ARRAY[]::uuid[]) 
				|| owner 
				|| assignee
			)
		from 
			tasks
		where
			id = $1
	`, taskID)
	if err != nil {
		return &userIDs
	}

	defer rows.Close()

	for rows.Next() {
		var s string
		rows.Scan(&s)
		userIDs = append(userIDs, s)
	}
	return &userIDs
}

// func getExistingTaskUserIDs(taskID string) *[]string {
// 	userIDs := make([]string, 0)
// 	rows, err := database.DB.Query("SELECT uid FROM task_custom_user_field_values WHERE task_id = $1", taskID)
// 	if err != nil {
// 		return &userIDs
// 	}

// 	defer rows.Close()

// 	for rows.Next() {
// 		var s string
// 		rows.Scan(&s)
// 		userIDs = append(userIDs, s)
// 	}
// 	return &userIDs
// }

func getExtraStringsFromSlices(checkArr *[]string, benchmarkArr *[]string) *[]string {
	extraValues := make([]string, 0)
	for _, value := range *checkArr {
		match := false
		for _, benchmarkValue := range *benchmarkArr {
			if value == benchmarkValue {
				match = true
				break
			}
		}
		if !match {
			extraValues = append(extraValues, value)
		}
	}
	return &extraValues
}

func getParentChildTasksUserIDs(taskID string) *[]string {
	result := make([]string, 0)

	rows, err := database.DB.Query(`
		with this_task_parents as (
			select parents, nlevel(parents) as lvl from tasks where id = $1
		),
		parent_child_task_ids as (
			select id from tasks where parents ~ concat('*.',replace($1,'-',''),'.*{0,5}')::lquery
			union
			select case when P.lvl = 1 then null else ltree2text(subpath(P.parents,P.lvl-2,1))::uuid end from this_task_parents P
			except 
			select null
		)
		select 
			distinct unnest(coalesce(supervisors,ARRAY[]::uuid[]) 
			|| coalesce(participants,ARRAY[]::uuid[]) 
			|| coalesce(viewers,ARRAY[]::uuid[]) 
			|| owner 
			|| assignee
		) as user_id
		from 
			tasks,
			parent_child_task_ids
		where 
			tasks.id in (parent_child_task_ids.id)
		except 
		select 
			distinct unnest(coalesce(supervisors,ARRAY[]::uuid[]) 
			|| coalesce(participants,ARRAY[]::uuid[]) 
			|| coalesce(viewers,ARRAY[]::uuid[]) 
			|| owner 
			|| assignee
		) as user_id
		from
			tasks
		where 
			id = $1
	`, taskID)
	if err != nil {
		return &result
	}

	defer rows.Close()

	for rows.Next() {
		var s string
		rows.Scan(&s)
		result = append(result, s)
	}
	return &result
}

package staff

import (
	"fmt"
	"math"
	"strings"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/rest/hrm/hrmcommon"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

func getSortKey(sortBy string) string {
	sortKey := ""
	if sortBy == "department_id" {
		sortKey = "D.name"
	} else if sortBy == "supervisor_id" {
		sortKey = "S.supervisor_name"
	} else {
		sortKey = "UD." + sortBy
	}
	if strings.HasSuffix(sortBy, "_dt") {
		return sortKey
	} else {
		return fmt.Sprintf("LOWER(%s)", sortKey)
	}
}

func getConditionString(filters *hrmcommon.FilterCollection, startInt int, prefix string) (str string, variables []interface{}) {
	filterStr := ""
	filterStrings := make([]string, 0)
	sortStr := ""
	vars := make([]interface{}, 0)

	startint := startInt - 1

	if len(filters.Filters) != 0 {
		for _, filter := range filters.Filters {
			str := ""
			if filter.Operator == "equals" {
				startint++
				str = fmt.Sprintf("lower(UD.%s) = $%d", filter.Field, startint)
				vars = append(vars, filter.Value)
				filterStrings = append(filterStrings, str)
			} else if filter.Operator == "includes" {
				startint++
				str = fmt.Sprintf("UD.%s = ANY($%d)", filter.Field, startint)
				vars = append(vars, pq.Array(filter.Values))
				filterStrings = append(filterStrings, str)
			} else if filter.Operator == "excludes" {
				startint++
				str = fmt.Sprintf("UD.%s <> ALL($%d)", filter.Field, startint)
				vars = append(vars, pq.Array(filter.Values))
				filterStrings = append(filterStrings, str)
			} else if filter.Operator == "can" {
				tempStr := make([]string, 0)
				for _, v := range filter.Values {
					tempStr = append(tempStr, fmt.Sprintf("UD.%s & %d = 1", filter.Field, int(math.Pow(2, v.(float64)))))
				}
				str = "(" + strings.Join(tempStr, " AND ") + ")"
				filterStrings = append(filterStrings, str)
			} else if filter.Operator == "cannot" {
				tempStr := make([]string, 0)
				for _, v := range filter.Values {
					tempStr = append(tempStr, fmt.Sprintf("UD.%s & %d = 0", filter.Field, int(math.Pow(2, v.(float64)))))
				}
				str = "(" + strings.Join(tempStr, " AND ") + ")"
				filterStrings = append(filterStrings, str)
			} else {
				startint++
				str = fmt.Sprintf("UD.%s ilike $%d", filter.Field, startint)

				switch filter.Operator {
				case "contains":
					vars = append(vars, "%"+filter.Value.(string)+"%")
				case "start_with":
					vars = append(vars, filter.Value.(string)+"%")
				case "end_with":
					vars = append(vars, "%"+filter.Value.(string))
				}
				filterStrings = append(filterStrings, str)
			}
		}
		filterStr = strings.Join(filterStrings, fmt.Sprintf(" %s ", filters.Mode))
		if len(filterStr) != 0 {
			filterStr = prefix + " ( " + filterStr + " ) "
		}
	}

	if filters.SortBy != "" && filters.SortOrder != "" {
		sortStr = fmt.Sprintf("ORDER BY %s %s", getSortKey(filters.SortBy), filters.SortOrder)
	}
	resultStr := strings.Trim(strings.Join([]string{filterStr, sortStr}, " "), " ")
	return resultStr, vars
}

func FetchStaff(filters *hrmcommon.FilterCollection) *[]Staff {
	staffs := make([]Staff, 0)
	conditionStr, variables := getConditionString(filters, 2, " AND ")
	vars := make([]interface{}, 0)
	vars = append(vars, uuid.Nil.String())
	vars = append(vars, variables...)

	mainStr := `
		SELECT 
			UD.id,
			UD.staff_id,
			UD.first_name,
			UD.last_name,
			UD.title,
			UD.department_id,
			UD.supervisor_id,
			UD.user_right,
			UD.email,
			floor(COALESCE(extract(epoch from UD.last_invite_dt) * 1000,0)::numeric) as last_invite_dt,
			floor(COALESCE(extract(epoch from UD.date_registered_dt) * 1000,0)::numeric) as date_registered_dt,
			floor(COALESCE(extract(epoch from UD.last_active_dt) * 1000,0)::numeric) as last_active_dt
		FROM 
			user_details UD 
		INNER JOIN 
			users U
		ON
			UD.id = U.id
	`

	if filters.SortBy == "department_id" {
		mainStr = mainStr + " inner join departments D on UD.department_id = D.id "
	} else if filters.SortBy == "supervisor_id" {
		mainStr = mainStr + `
			inner join 
			(select id, concat(first_name,' ',last_name) as supervisor_name from user_details) S 
			on UD.supervisor_id = S.id 
		`
	}

	mainStr = mainStr + " WHERE U.authorized = true AND UD.id <> $1 "
	mainStr = mainStr + fmt.Sprintf("%s limit %d offset %d", conditionStr, filters.Limit, filters.Page*filters.Limit)

	rows, err := database.DB.Query(mainStr, vars...)
	if err != nil {
		fmt.Println(err)
		return &staffs
	}
	defer rows.Close()
	for rows.Next() {
		var (
			staff Staff
		)
		err = rows.Scan(
			&staff.ID,
			&staff.StaffID,
			&staff.FirstName,
			&staff.LastName,
			&staff.Title,
			&staff.Department,
			&staff.SupervisorID,
			&staff.UserRight,
			&staff.Email,
			&staff.LastInviteDT,
			&staff.DateRegistered,
			&staff.LastActive,
		)
		if err != nil {
			fmt.Println(err)
			return &[]Staff{}
		}
		staffs = append(staffs, staff)
	}
	return &staffs
}

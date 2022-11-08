package dept

import (
	"fmt"
	"math"
	"strings"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/rest/hrm/hrmcommon"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

func FetchDepartments(filters *hrmcommon.FilterCollection) *[]Department {
	departments := make([]Department, 0)
	conditionStr, variables := getConditionString(filters, 2, "AND ")
	vars := make([]interface{}, 0)
	vars = append(vars, uuid.Nil.String())
	vars = append(vars, variables...)

	query := fmt.Sprintf("SELECT id, internal_id, name FROM departments WHERE id<>$1 %s limit %d offset %d", conditionStr, filters.Limit, filters.Page*filters.Limit)

	rows, err := database.DB.Query(query, vars...)
	if err != nil {
		return &departments
	}
	defer rows.Close()
	for rows.Next() {
		var dept Department
		err = rows.Scan(&dept.ID, &dept.Internal_ID, &dept.Name)
		if err != nil {
			return &[]Department{}
		}
		departments = append(departments, dept)
	}
	return &departments
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
				str = fmt.Sprintf("lower(%s) = $%d", filter.Field, startint)
				vars = append(vars, filter.Value)
				filterStrings = append(filterStrings, str)
			} else if filter.Operator == "includes" {
				startint++
				str = fmt.Sprintf("%s = ANY($%d)", filter.Field, startint)
				vars = append(vars, pq.Array(filter.Values))
				filterStrings = append(filterStrings, str)
			} else if filter.Operator == "excludes" {
				startint++
				str = fmt.Sprintf("%s <> ALL($%d)", filter.Field, startint)
				vars = append(vars, pq.Array(filter.Values))
				filterStrings = append(filterStrings, str)
			} else if filter.Operator == "can" {
				tempStr := make([]string, 0)
				for _, v := range filter.Values {
					tempStr = append(tempStr, fmt.Sprintf("%s & %d = 1", filter.Field, int(math.Pow(2, v.(float64)))))
				}
				str = "(" + strings.Join(tempStr, " AND ") + ")"
				filterStrings = append(filterStrings, str)
			} else if filter.Operator == "cannot" {
				tempStr := make([]string, 0)
				for _, v := range filter.Values {
					tempStr = append(tempStr, fmt.Sprintf("%s & %d = 0", filter.Field, int(math.Pow(2, v.(float64)))))
				}
				str = "(" + strings.Join(tempStr, " AND ") + ")"
				filterStrings = append(filterStrings, str)
			} else {
				startint++
				str = fmt.Sprintf("%s ilike $%d", filter.Field, startint)

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
		sortStr = fmt.Sprintf("ORDER BY LOWER(%s) %s", filters.SortBy, filters.SortOrder)
	}
	resultStr := strings.Trim(strings.Join([]string{filterStr, sortStr}, " "), " ")
	return resultStr, vars
}

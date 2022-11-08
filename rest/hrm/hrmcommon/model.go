package hrmcommon

type Filter struct {
	Field    string        `json:"field"`
	Operator string        `json:"operator"`
	Value    interface{}   `json:"value"`
	Values   []interface{} `json:"values"`
}

type FilterCollection struct {
	Mode      string   `json:"filterMode"`
	SortBy    string   `json:"sortBy"`
	SortOrder string   `json:"sortOrder"`
	Filters   []Filter `json:"filters"`
	Limit     int      `json:"limit"`
	Page      int      `json:"page"`
}

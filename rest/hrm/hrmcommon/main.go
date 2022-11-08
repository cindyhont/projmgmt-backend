package hrmcommon

func TableNameOK(s string) bool {
	names := []string{
		"departments",
		"user_details",
	}

	for _, name := range names {
		if s == name {
			return true
		}
	}
	return false
}

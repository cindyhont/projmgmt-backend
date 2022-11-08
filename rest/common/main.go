package common

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/julienschmidt/httprouter"
)

func SendResponse(w http.ResponseWriter, response *model.Response) {
	json.NewEncoder(w).Encode(response)
}

func SetSessionCookie(w http.ResponseWriter, newSID string, expiry time.Time) {
	cookie := http.Cookie{
		Name:     "sid",
		Value:    newSID,
		Path:     "/",
		Expires:  expiry,
		HttpOnly: true,
	}
	http.SetCookie(w, &cookie)
}

func UpdateSession(w http.ResponseWriter, r *http.Request, _ httprouter.Params, s *model.Session, signedIn bool, _ string) {
	data := struct {
		Success bool `json:"success"`
	}{Success: signedIn}
	SendResponse(w, &model.Response{Session: s, Data: data})
}

func TooLongTooShort(s string, min int, max int) bool {
	len := len(s)
	return len < min || len > max
}

func UniqueStringFromSlice(arr *[]string) *[]string {
	keys := make(map[string]bool)
	_arr := *arr
	result := make([]string, 0)

	for _, item := range _arr {
		_, value := keys[item]
		if !value {
			keys[item] = true
			result = append(result, item)
		}
	}
	return &result
}

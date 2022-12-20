package tasks

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/instantcomm"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/cindyhont/projmgmt-backend/rest/common"
	"github.com/julienschmidt/httprouter"
)

func updateTimer(
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
		Success     bool   `json:"success"`
		WsRequestID string `json:"wsid"`
	}{
		Success:     false,
		WsRequestID: "",
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	var req struct {
		TaskID          string `json:"taskID"`
		NewTimeRecordID string `json:"newTimeRecordID"`
		Time            int64  `json:"time"`
	}

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	var timerRunning bool
	database.DB.QueryRow("SELECT EXISTS (SELECT 1 FROM task_time_track WHERE uid=$1 AND end_dt IS NULL)", uid).Scan(&timerRunning)

	wsUserIDs := *getTaskUserIDs(req.TaskID)
	wsPayload := map[string]interface{}{
		"userID": uid,
		"time":   req.Time,
	}

	if timerRunning {
		var taskID, timerID string
		database.DB.QueryRow("SELECT id, task_id FROM task_time_track WHERE uid=$1 AND end_dt IS NULL LIMIT 1", uid).Scan(&timerID, &taskID)
		database.DB.Exec("UPDATE task_time_track SET end_dt=$1 WHERE uid=$2 AND end_dt IS NULL", time.UnixMilli(req.Time), uid)

		wsPayload["endTimerID"] = timerID
		wsUserIDs = append(wsUserIDs, *getTaskUserIDs(taskID)...)

		if taskID != req.TaskID {
			addTimerFunc(req.NewTimeRecordID, uid, req.TaskID, req.Time)
			wsPayload["startTaskID"] = req.TaskID
			wsPayload["newTimeRecordID"] = req.NewTimeRecordID
		}
	} else {
		addTimerFunc(req.NewTimeRecordID, uid, req.TaskID, req.Time)
		wsPayload["startTaskID"] = req.TaskID
		wsPayload["newTimeRecordID"] = req.NewTimeRecordID
	}

	// fmt.Println(wsPayload)
	// fmt.Println(wsUserIDs)

	wsMessage := instantcomm.Response{
		Type:    "tasks_update-timer",
		Payload: wsPayload,
	}

	data.WsRequestID = instantcomm.SaveWsMessageInDB(&wsMessage, common.UniqueStringFromSlice(&wsUserIDs))
	data.Success = true
	json.NewEncoder(w).Encode(data)
}

func addTimerFunc(recordID string, uid string, taskID string, timeNow int64) {
	database.DB.Exec(
		"INSERT INTO task_time_track (id,uid,task_id,start_dt) VALUES ($1,$2,$3,$4)",
		recordID,
		uid,
		taskID,
		time.UnixMilli(timeNow),
	)
}

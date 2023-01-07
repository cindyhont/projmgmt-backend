package newuser

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/rest/common"
	"github.com/cindyhont/projmgmt-backend/usermgmt"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
)

func createVisitor(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	data := struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}{
		Success: false,
		Message: "",
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	var req struct {
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Username  string `json:"username"`
		Password  string `json:"password"`
	}

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	if common.TooLongTooShort(req.Username, 6, 128) || common.TooLongTooShort(req.Password, 6, 128) || common.TooLongTooShort(req.FirstName, 1, 128) || common.TooLongTooShort(req.LastName, 1, 128) {
		data.Message = "Invalid input."
		json.NewEncoder(w).Encode(data)
		return
	}

	var usernameExists bool
	database.DB.QueryRow("SELECT EXISTS (SELECT 1 FROM users WHERE username = $1)", req.Username).Scan(&usernameExists)
	if usernameExists {
		data.Message = "Username already exists."
		json.NewEncoder(w).Encode(data)
		return
	}

	// hash password
	newPwd, err := usermgmt.GeneratePassword(req.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var uid string
	err = database.DB.QueryRow("INSERT INTO users (username, password) VALUES ($1,$2) RETURNING id", req.Username, newPwd).Scan(&uid)
	if err != nil {
		data.Message = "Server error. Please try again later."
		json.NewEncoder(w).Encode(data)
		return
	}

	database.DB.Exec(`
		INSERT INTO
			user_details
			(
				id,
				staff_id,
				first_name,
				last_name,
				title,
				user_right,
				email,
				date_registered_dt,
				last_active_dt,
				tsv,
				visitor
			)
		VALUES
			(
				$1,
				gen_random_uuid()::text,
				$2,
				$3,
				'',
				0,
				gen_random_uuid()::text,
				now(),
				now(),
				to_tsvector($4),
				true
			)
	`,
		uid,
		req.FirstName,
		req.LastName,
		req.FirstName+" "+req.LastName,
	)

	database.DB.Exec(`
		WITH source AS (
			SELECT id, $1::uuid as uid, type_name, default_value FROM task_custom_field_type WHERE default_value IS NOT NULL
		)
		INSERT INTO
			task_custom_user_fields
			(uid,field_type,details,field_name)
		SELECT
			uid, id, default_value, type_name
		FROM
			source
	`, uid)

	// create chatroom and insert messages to visitor
	chatRoomID := uuid.New().String()
	_, err = database.DB.Exec("INSERT INTO chatrooms (id) VALUES ($1)", chatRoomID)
	if err != nil {
		fmt.Println("b: ", err)
		data.Message = "Server error. Please try again later."
		json.NewEncoder(w).Encode(data)
		return
	}
	database.DB.Exec(`
		INSERT INTO
			chatrooms_users
			(rid,uid)
		VALUES
			($1,$2),
			($1,$3)
	`,
		chatRoomID,
		uid,
		os.Getenv("PROJMGMT_DEMO_USER"),
	)

	database.DB.Exec(`
		INSERT INTO
			chat_messages
			(rid,content,sender_id)
		VALUES
			($1,$3,$2),
			($1,$4,$2),
			($1,$5,$2),
			($1,$6,$2)
	`,
		chatRoomID,
		os.Getenv("PROJMGMT_DEMO_USER"),
		fmt.Sprintf("<p>Hi %s! This chat app is similar to WhatsApp, Telegram, etc.</p>", req.FirstName),
		"<p>On top of the main panel, you can search for other users or groups by typing the name of the user/group.</p>",
		"<p>If you want to write to me, leave a message here with your contact information.</p>",
		"<p>Regards, Cindy</p>",
	)

	sessionID, expiryMS, err := addSessinToDB(uid)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	common.SetSessionCookie(w, sessionID, expiryMS)
	data.Success = true
	json.NewEncoder(w).Encode(data)
}

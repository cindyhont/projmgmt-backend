package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/cindyhont/projmgmt-backend/websocket"
	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/julienschmidt/httprouter"
	"github.com/lib/pq"
)

func createGroup(
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
		RoomID      string `json:"roomID"`
		Success     bool   `json:"success"`
		WsRequestID string `json:"wsid"`
	}{
		RoomID:      "",
		Success:     false,
		WsRequestID: "",
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	var req struct {
		Name   string   `json:"name"`
		Avatar string   `json:"avatar"`
		Users  []string `json:"uids"`
	}

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	var colStr, varStr string
	vars := make([]interface{}, 0)
	refinedName := refineGroupName(req.Name)
	vars = append(vars, refinedName)
	vars = append(vars, pq.Array(strings.Split(refinedName, " ")))
	vars = append(vars, refinedName)
	if req.Avatar != "" {
		colStr = ",avatar"
		varStr = ",$4"

		// here upload image to cloudinary
		cld, _ := cloudinary.NewFromParams(os.Getenv("CLOUDINARY_CLOUD_NAME"), os.Getenv("CLOUDINARY_API_KEY"), os.Getenv("CLOUDINARY_API_SECRET"))
		resp, err := cld.Upload.Upload(context.Background(), req.Avatar, uploader.UploadParams{})
		if err != nil {
			json.NewEncoder(w).Encode(data)
			return
		}

		vars = append(vars, resp.SecureURL)
	}
	err = database.DB.QueryRow(fmt.Sprintf(`INSERT INTO chatrooms (room_name,tsv,tsv_w_position%s) VALUES ($1,array_to_tsvector($2),to_tsvector($3)%s) RETURNING id`, colStr, varStr), vars...).Scan(&data.RoomID)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	data.Success = bulkImportRoomUsers(req.Users, uid, data.RoomID)

	if !data.Success {
		json.NewEncoder(w).Encode(data)
		return
	}

	userIDs := make([]string, 0)
	userIDs = append(userIDs, req.Users...)
	userIDs = append(userIDs, uid)

	wsMessage := websocket.Response{
		Type: "chat_new-group",
		Payload: map[string]interface{}{
			"roomID": data.RoomID,
			"users":  userIDs,
			"name":   req.Name,
			"avatar": req.Avatar,
		},
	}

	data.WsRequestID = websocket.SaveWsMessageInDB(&wsMessage, &userIDs)

	json.NewEncoder(w).Encode(data)
}

// https://stackoverflow.com/questions/62343536/remove-all-punctuation-except-in-numbers
func refineGroupName(text string) string {
	// Regexp that finds all puncuation characters grouping the characters that wrap it
	re := regexp.MustCompile(`(.{0,1})([^\w\s])(.{0,1})`)

	// Regexp that determines if a given string is wrapped by digit characters
	isFloat := regexp.MustCompile(`\d([^\w\s])\d`)

	// Get the parts using the punctuation regexp... e.g. "t. "
	parts := re.FindAllString(text, -1)

	// Iterate through the parts
	for _, part := range parts {
		// Determine if the part is a float...
		isAFloat := isFloat.MatchString(part)
		// If it is not a float, make a single replacement to remove the puncuation
		if !isAFloat {
			newPart := re.ReplaceAllString(part, "$1$3")
			text = strings.Replace(text, part, newPart, 1)
		}
	}
	return text
}

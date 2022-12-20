package settings

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/cindyhont/projmgmt-backend/database"
	"github.com/cindyhont/projmgmt-backend/instantcomm"
	"github.com/cindyhont/projmgmt-backend/model"
	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/julienschmidt/httprouter"
)

func updateAvatar(
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
		Avatar      string `json:"avatar"`
	}{
		Success:     false,
		WsRequestID: "",
		Avatar:      "",
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	var req struct {
		Avatar string `json:"avatar"`
	}

	if err = json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	cld, _ := cloudinary.NewFromParams(os.Getenv("CLOUDINARY_CLOUD_NAME"), os.Getenv("CLOUDINARY_API_KEY"), os.Getenv("CLOUDINARY_API_SECRET"))
	resp, err := cld.Upload.Upload(context.Background(), req.Avatar, uploader.UploadParams{})
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	data.Avatar = resp.SecureURL

	database.DB.Exec("UPDATE user_details SET avatar = $1 WHERE id = $2", resp.SecureURL, uid)

	wsMessage := instantcomm.Response{
		Type: "hrm_update-avatar",
		Payload: map[string]interface{}{
			"avatar": resp.SecureURL,
			"user":   uid,
		},
		ToAllRecipients: true,
	}

	data.WsRequestID = instantcomm.SaveWsMessageInDB(&wsMessage, &[]string{})
	data.Success = true
	json.NewEncoder(w).Encode(data)
}

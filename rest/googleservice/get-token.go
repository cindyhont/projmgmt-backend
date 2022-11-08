package googleservice

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/julienschmidt/httprouter"
	"golang.org/x/oauth2/google"
)

func getToken(
	w http.ResponseWriter,
	r *http.Request,
	p httprouter.Params,
) {
	resp, err := http.Get(os.Getenv("GOOGLE_API_KEY_URL"))
	if err != nil {
		fmt.Println(err)
	}

	data := struct {
		AccessToken string `json:"accessToken"`
		Expiry      int64  `json:"expiry"`
	}{}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	config, err := google.JWTConfigFromJSON(
		body,
		"https://www.googleapis.com/auth/drive",
	)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}
	token, err := config.TokenSource(context.Background()).Token()
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	data.AccessToken = strings.TrimRight(token.AccessToken, ".")
	data.Expiry = token.Expiry.UnixMilli() - 30000
	json.NewEncoder(w).Encode(data)
}

package instantcomm

type Response struct {
	Type            string                 `json:"type"`
	DateTime        int64                  `json:"dt,omitempty"`
	Payload         map[string]interface{} `json:"payload"`
	ToAllRecipients bool                   `json:"toAllRecipients"`
	FromIP          string                 `json:"fromIP,omitempty"`
	UserIDs         []string               `json:"userIDs,omitempty"`
}

type request struct {
	Request    string   `json:"req"`
	Requests   []string `json:"reqs"`
	ChatRoomID string   `json:"roomid,omitempty"`
	Typing     bool     `json:"typing,omitempty"`
	UserID     string   `json:"uid,omitempty"`
}

package scene

import (
	"encoding/base64"
	"encoding/json"

	"github.com/gin-gonic/gin"
)

type ToastMessage struct {
	Heading string `json:"heading"`
	Message string `json:"message"`
	Type    string `json:"type"` // success, error, info, warning
}

type ToastMessages []ToastMessage

func (t ToastMessages) MarshalCookie() string {
	data, err := json.Marshal(t)
	if err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(data)
}

func (t *ToastMessages) UnmarshalCookie(cookie string) {
	if cookie == "" {
		return
	}

	data, err := base64.RawURLEncoding.DecodeString(cookie)
	if err != nil {
		panic(err)
	}

	if err := json.Unmarshal(data, &t); err != nil {
		panic(err)
	}
}

func (s Scene) WithToastMessages(c *gin.Context) Scene {
	// Are there any toast messages in the context?
	if cookie, err := c.Cookie(ToastCookie); err == nil && cookie != "" {
		var messages ToastMessages
		messages.UnmarshalCookie(cookie)
		if len(messages) > 0 {
			s[ToastMsgsKey] = messages
		}
	}
	return s
}

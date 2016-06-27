package req

import (
	"encoding/json"
)

const (
	REQ_RES      = "req-res"
	PUB_SUB      = "pub-sub"
	NOTIFICATION = "notification"
)

type Message struct {
	Type      string          `json:"type"`
	Topic     string          `json:"topic"`
	Namespace string          `json:"namespace"`
	Body      json.RawMessage `json:"body"`
}

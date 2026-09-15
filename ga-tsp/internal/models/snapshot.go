package models

import (
	"encoding/json"
	"time"
)

type Snapshot struct {
	ID        uint64          `json:"id"`
	Kind      string          `json:"kind"`
	CreatedAt time.Time       `json:"createdAt"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

package audit

import (
	"encoding/json"
	"time"
)

type Event struct {
	ID             string          `json:"id"`
	OrganizationID *string         `json:"organization_id,omitempty"`
	ActorType      string          `json:"actor_type"`
	ActorID        string          `json:"actor_id"`
	Action         string          `json:"action"`
	TargetType     string          `json:"target_type"`
	TargetID       string          `json:"target_id"`
	CorrelationID  string          `json:"correlation_id"`
	Metadata       json.RawMessage `json:"metadata"`
	CreatedAt      time.Time       `json:"created_at"`
}

package workload

import "time"

type Workload struct {
	ID            string     `json:"id"`
	OrganizationID string     `json:"organization_id"`
	TrustDomainID string     `json:"trust_domain_id"`
	Name          string     `json:"name"`
	Environment   string     `json:"environment"`
	SPIFFEID      string     `json:"spiffe_id"`
	Status        string     `json:"status"`
	LastSeenAt    *time.Time `json:"last_seen_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

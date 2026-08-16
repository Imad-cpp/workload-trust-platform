package diagnostics

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("diagnostics organization not found")

type Summary struct {
	OrganizationID        string     `json:"organization_id"`
	WorkloadsTotal        int64      `json:"workloads_total"`
	WorkloadsHealthy      int64      `json:"workloads_healthy"`
	WorkloadsDegraded     int64      `json:"workloads_degraded"`
	WorkloadsOffline      int64      `json:"workloads_offline"`
	RegistrationsPending  int64      `json:"registrations_pending"`
	RegistrationsConverged int64     `json:"registrations_converged"`
	RegistrationsError    int64      `json:"registrations_error"`
	PoliciesDraft         int64      `json:"policies_draft"`
	PoliciesActive        int64      `json:"policies_active"`
	PoliciesDisabled      int64      `json:"policies_disabled"`
	AuditEvents           int64      `json:"audit_events"`
	SecurityEvents        int64      `json:"security_events"`
	LatestAuditAt         *time.Time `json:"latest_audit_at,omitempty"`
}

type Reader interface {
	SummaryByOrganization(context.Context, string) (Summary, error)
}

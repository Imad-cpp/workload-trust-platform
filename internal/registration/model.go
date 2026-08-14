package registration

import "time"

type Rule struct {
	ID                 string
	OrganizationID     string
	WorkloadID         string
	WorkloadSPIFFEID   string
	Selectors          []string
	DesiredState       string
	Revision           int64
	ParentSPIFFEID     *string
	X509SVIDTTLSeconds int32
	SPIREEntryID       *string
	ReconcileStatus    string
	LastReconciledAt   *time.Time
	LastErrorCode      *string
}

package registration

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

var (
	ErrInvalidInput = errors.New("invalid registration mutation")
	ErrNotFound     = errors.New("registration mutation target not found")
	ErrConflict     = errors.New("registration revision conflict")
)

var (
	uuidPattern      = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
	selectorTypeExpr = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)
)

const (
	MinX509SVIDTTLSeconds = 60
	MaxX509SVIDTTLSeconds = 86400
	MaxSelectors          = 32
	MaxSelectorValueBytes = 512
	MaxSPIFFEIDBytes      = 2048
)

type MutationActor struct {
	Type string
	ID   string
	Role string
}

type CreateDesiredInput struct {
	WorkloadID         string
	Selectors          []string
	ParentSPIFFEID     string
	X509SVIDTTLSeconds int32
}

type ReplaceDesiredInput struct {
	ExpectedRevision   int64
	DesiredState       string
	Selectors          []string
	ParentSPIFFEID     string
	X509SVIDTTLSeconds int32
}

type MutationResult struct {
	ID                 string `json:"id"`
	OrganizationID     string `json:"organization_id"`
	WorkloadID         string `json:"workload_id"`
	DesiredState       string `json:"desired_state"`
	Revision           int64  `json:"revision"`
	X509SVIDTTLSeconds int32  `json:"x509_svid_ttl_seconds"`
	SelectorCount      int    `json:"selector_count"`
	ReconcileStatus    string `json:"reconcile_status"`
}

type Mutator interface {
	CreateDesired(ctx context.Context, actor MutationActor, correlationID string, input CreateDesiredInput) (MutationResult, error)
	ReplaceDesired(ctx context.Context, actor MutationActor, correlationID, ruleID string, input ReplaceDesiredInput) (MutationResult, error)
}

func validateActor(actor MutationActor) error {
	if actor.Type != "operator" || strings.TrimSpace(actor.ID) == "" || strings.TrimSpace(actor.Role) == "" {
		return fmt.Errorf("%w: operator actor attribution is required", ErrInvalidInput)
	}
	return nil
}

func validateCreateInput(input CreateDesiredInput) error {
	if !uuidPattern.MatchString(input.WorkloadID) {
		return fmt.Errorf("%w: workload_id must be a UUID", ErrInvalidInput)
	}
	return validateDesiredFields("present", input.Selectors, input.ParentSPIFFEID, input.X509SVIDTTLSeconds)
}

func validateReplaceInput(ruleID string, input ReplaceDesiredInput) error {
	if !uuidPattern.MatchString(ruleID) {
		return fmt.Errorf("%w: registration rule ID must be a UUID", ErrInvalidInput)
	}
	if input.ExpectedRevision < 1 {
		return fmt.Errorf("%w: expected_revision must be positive", ErrInvalidInput)
	}
	return validateDesiredFields(input.DesiredState, input.Selectors, input.ParentSPIFFEID, input.X509SVIDTTLSeconds)
}

func validateDesiredFields(desiredState string, selectors []string, parentSPIFFEID string, ttl int32) error {
	if desiredState != "present" && desiredState != "absent" {
		return fmt.Errorf("%w: desired_state must be present or absent", ErrInvalidInput)
	}
	if ttl < MinX509SVIDTTLSeconds || ttl > MaxX509SVIDTTLSeconds {
		return fmt.Errorf("%w: x509_svid_ttl_seconds must be between %d and %d", ErrInvalidInput, MinX509SVIDTTLSeconds, MaxX509SVIDTTLSeconds)
	}
	if err := validateSPIFFEID(parentSPIFFEID); err != nil {
		return fmt.Errorf("%w: parent_spiffe_id: %v", ErrInvalidInput, err)
	}
	if len(selectors) == 0 || len(selectors) > MaxSelectors {
		return fmt.Errorf("%w: selectors must contain between 1 and %d values", ErrInvalidInput, MaxSelectors)
	}

	seen := make(map[string]struct{}, len(selectors))
	for _, raw := range selectors {
		kind, value, ok := strings.Cut(raw, ":")
		if !ok || !selectorTypeExpr.MatchString(kind) || value == "" || len(value) > MaxSelectorValueBytes {
			return fmt.Errorf("%w: selector %q is invalid", ErrInvalidInput, raw)
		}
		if strings.IndexFunc(value, unicode.IsControl) >= 0 {
			return fmt.Errorf("%w: selector %q contains control characters", ErrInvalidInput, raw)
		}
		if _, duplicate := seen[raw]; duplicate {
			return fmt.Errorf("%w: duplicate selector %q", ErrInvalidInput, raw)
		}
		seen[raw] = struct{}{}
	}
	return nil
}

func validateSPIFFEID(raw string) error {
	if raw == "" || len(raw) > MaxSPIFFEIDBytes {
		return errors.New("SPIFFE ID length is invalid")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return errors.New("SPIFFE ID is not a valid URI")
	}
	if parsed.Scheme != "spiffe" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Opaque != "" {
		return errors.New("SPIFFE ID must be an absolute spiffe URI without userinfo, query, or fragment")
	}
	if parsed.Hostname() != parsed.Host || parsed.Host != strings.ToLower(parsed.Host) {
		return errors.New("SPIFFE trust domain must be lowercase and must not contain a port")
	}
	if parsed.Path == "" || !strings.HasPrefix(parsed.Path, "/") || parsed.RawPath != "" {
		return errors.New("SPIFFE ID path must be canonical and absolute")
	}
	return nil
}

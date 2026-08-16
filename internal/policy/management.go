package policy

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
	ErrInvalidInput = errors.New("invalid policy mutation")
	ErrForbidden    = errors.New("policy mutation actor is forbidden")
	ErrNotFound     = errors.New("policy mutation target not found")
	ErrConflict     = errors.New("policy revision conflict")
)

var (
	uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
	namePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 _.-]{2,127}$`)
)

const (
	MaxSPIFFEIDBytes     = 2048
	MaxChangeReasonBytes = 1024
	ActionConnect        = "connect"
	EffectAllow          = "allow"
	EffectDeny           = "deny"
)

type MutationActor struct {
	Type string
	ID   string
	Role string
}

type RuleInput struct {
	SourceSPIFFEID      string
	DestinationSPIFFEID string
	Action              string
	Effect              string
	ChangeReason        string
}

type CreateInput struct {
	OrganizationID string
	Name           string
	InitialVersion RuleInput
}

type AppendVersionInput struct {
	ExpectedRevision int64
	Version          RuleInput
}

type ActivateInput struct {
	ExpectedRevision int64
	VersionID        string
}

type PolicyResult struct {
	ID              string  `json:"id"`
	OrganizationID  string  `json:"organization_id"`
	Name            string  `json:"name"`
	Status          string  `json:"status"`
	Revision        int64   `json:"revision"`
	ActiveVersionID *string `json:"active_version_id,omitempty"`
}

type VersionResult struct {
	ID             string `json:"id"`
	PolicyID       string `json:"policy_id"`
	VersionNumber  int64  `json:"version_number"`
	PolicyRevision int64  `json:"policy_revision"`
	Effect         string `json:"effect"`
}

type CreateResult struct {
	Policy  PolicyResult  `json:"policy"`
	Version VersionResult `json:"version"`
}

type Manager interface {
	Create(ctx context.Context, actor MutationActor, correlationID string, input CreateInput) (CreateResult, error)
	AppendVersion(ctx context.Context, actor MutationActor, correlationID, policyID string, input AppendVersionInput) (VersionResult, error)
	Activate(ctx context.Context, actor MutationActor, correlationID, policyID string, input ActivateInput) (PolicyResult, error)
}

func validateActor(actor MutationActor) error {
	if actor.Type != "operator" || strings.TrimSpace(actor.ID) == "" {
		return fmt.Errorf("%w: operator attribution is required", ErrInvalidInput)
	}
	if actor.Role != "operator" {
		return ErrForbidden
	}
	return nil
}

func normalizeCreateInput(input CreateInput) CreateInput {
	input.Name = strings.TrimSpace(input.Name)
	input.InitialVersion = normalizeRuleInput(input.InitialVersion)
	return input
}

func normalizeAppendInput(input AppendVersionInput) AppendVersionInput {
	input.Version = normalizeRuleInput(input.Version)
	return input
}

func validateCreateInput(input CreateInput) error {
	if !uuidPattern.MatchString(input.OrganizationID) {
		return fmt.Errorf("%w: organization_id must be a UUID", ErrInvalidInput)
	}
	if !namePattern.MatchString(input.Name) {
		return fmt.Errorf("%w: policy name is invalid", ErrInvalidInput)
	}
	return validateRuleInput(input.InitialVersion)
}

func validateAppendInput(policyID string, input AppendVersionInput) error {
	if !uuidPattern.MatchString(policyID) {
		return fmt.Errorf("%w: policy ID must be a UUID", ErrInvalidInput)
	}
	if input.ExpectedRevision < 1 {
		return fmt.Errorf("%w: expected_revision must be positive", ErrInvalidInput)
	}
	return validateRuleInput(input.Version)
}

func validateActivateInput(policyID string, input ActivateInput) error {
	if !uuidPattern.MatchString(policyID) || !uuidPattern.MatchString(input.VersionID) {
		return fmt.Errorf("%w: policy and version IDs must be UUIDs", ErrInvalidInput)
	}
	if input.ExpectedRevision < 1 {
		return fmt.Errorf("%w: expected_revision must be positive", ErrInvalidInput)
	}
	return nil
}

func validateRuleInput(input RuleInput) error {
	if err := validateSPIFFEID(input.SourceSPIFFEID); err != nil {
		return fmt.Errorf("%w: source_spiffe_id: %v", ErrInvalidInput, err)
	}
	if err := validateSPIFFEID(input.DestinationSPIFFEID); err != nil {
		return fmt.Errorf("%w: destination_spiffe_id: %v", ErrInvalidInput, err)
	}
	if input.Action != ActionConnect {
		return fmt.Errorf("%w: action must be %q in V1", ErrInvalidInput, ActionConnect)
	}
	if input.Effect != EffectAllow && input.Effect != EffectDeny {
		return fmt.Errorf("%w: effect must be allow or deny", ErrInvalidInput)
	}
	if len(input.ChangeReason) == 0 || len(input.ChangeReason) > MaxChangeReasonBytes || strings.IndexFunc(input.ChangeReason, unicode.IsControl) >= 0 {
		return fmt.Errorf("%w: change_reason is invalid", ErrInvalidInput)
	}
	return nil
}

func normalizeRuleInput(input RuleInput) RuleInput {
	input.Action = strings.ToLower(strings.TrimSpace(input.Action))
	input.Effect = strings.ToLower(strings.TrimSpace(input.Effect))
	input.ChangeReason = strings.TrimSpace(input.ChangeReason)
	return input
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
	if parsed.Path == "" || !strings.HasPrefix(parsed.Path, "/") || parsed.RawPath != "" || parsed.String() != raw {
		return errors.New("SPIFFE ID path must be canonical and absolute")
	}
	for _, segment := range strings.Split(strings.TrimPrefix(parsed.Path, "/"), "/") {
		if segment == "" || segment == "." || segment == ".." || strings.ContainsRune(segment, '\\') || strings.IndexFunc(segment, unicode.IsControl) >= 0 {
			return errors.New("SPIFFE ID path contains a non-canonical segment")
		}
	}
	return nil
}

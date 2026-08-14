package reconcile

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/Imad-cpp/workload-trust-platform/internal/audit"
	"github.com/Imad-cpp/workload-trust-platform/internal/registration"
	"github.com/Imad-cpp/workload-trust-platform/internal/spiremgmt"
)

const ownerHintPrefix = "wtp-rule:"

type Summary struct {
	Total     int
	Converged int
	Changed   int
	Failed    int
}

type Service struct {
	rules registration.Store
	spire spiremgmt.Client
	audit audit.Appender
}

func New(rules registration.Store, spire spiremgmt.Client, auditAppender audit.Appender) (*Service, error) {
	if rules == nil || spire == nil || auditAppender == nil {
		return nil, errors.New("reconciler dependencies are required")
	}
	return &Service{rules: rules, spire: spire, audit: auditAppender}, nil
}

func (s *Service) Run(ctx context.Context) (Summary, error) {
	rules, err := s.rules.List(ctx)
	if err != nil {
		return Summary{}, err
	}
	correlationID := newCorrelationID()
	summary := Summary{Total: len(rules)}
	var runErrors []error
	for _, rule := range rules {
		changed, err := s.reconcileRule(ctx, rule, correlationID)
		if err != nil {
			summary.Failed++
			runErrors = append(runErrors, fmt.Errorf("rule %s: %w", rule.ID, err))
			continue
		}
		summary.Converged++
		if changed {
			summary.Changed++
		}
	}
	return summary, errors.Join(runErrors...)
}

func (s *Service) reconcileRule(ctx context.Context, rule registration.Rule, correlationID string) (bool, error) {
	switch rule.DesiredState {
	case "present":
		return s.reconcilePresent(ctx, rule, correlationID)
	case "absent":
		return s.reconcileAbsent(ctx, rule, correlationID)
	default:
		return false, s.failRule(ctx, rule, "invalid_desired_state", fmt.Errorf("unsupported desired state %q", rule.DesiredState))
	}
}

func (s *Service) reconcilePresent(ctx context.Context, rule registration.Rule, correlationID string) (bool, error) {
	desired, err := desiredEntry(rule)
	if err != nil {
		return false, s.failRule(ctx, rule, "invalid_desired_state", err)
	}

	if rule.SPIREEntryID == nil {
		return s.createOrBind(ctx, rule, desired, correlationID)
	}

	actual, err := s.spire.GetEntry(ctx, *rule.SPIREEntryID)
	if errors.Is(err, spiremgmt.ErrNotFound) {
		return s.createOrBind(ctx, rule, desired, correlationID)
	}
	if err != nil {
		return false, s.failRule(ctx, rule, "spire_unavailable", err)
	}
	if !ownedByRule(actual, rule.ID) {
		return false, s.conflict(ctx, rule, actual.ID, correlationID)
	}

	desired.ID = actual.ID
	if entriesEquivalent(actual, desired) {
		if rule.ReconcileStatus != "converged" {
			if err := s.recordAudit(ctx, rule, correlationID, "bind_owned_existing", actual.ID); err != nil {
				return false, err
			}
			if err := s.rules.MarkConverged(ctx, rule.ID, &actual.ID); err != nil {
				return false, err
			}
		}
		return false, nil
	}

	updated, err := s.spire.UpdateEntry(ctx, desired)
	if errors.Is(err, spiremgmt.ErrNotFound) {
		return s.createOrBind(ctx, rule, desiredWithoutID(desired), correlationID)
	}
	if err != nil {
		return false, s.failRule(ctx, rule, "spire_update_failed", err)
	}
	if !ownedByRule(updated, rule.ID) {
		return false, s.conflict(ctx, rule, updated.ID, correlationID)
	}
	if err := s.recordAudit(ctx, rule, correlationID, "update", updated.ID); err != nil {
		return false, err
	}
	if err := s.rules.MarkConverged(ctx, rule.ID, &updated.ID); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) createOrBind(ctx context.Context, rule registration.Rule, desired spiremgmt.Entry, correlationID string) (bool, error) {
	result, err := s.spire.CreateEntry(ctx, desired)
	if err != nil {
		return false, s.failRule(ctx, rule, "spire_create_failed", err)
	}
	if !ownedByRule(result.Entry, rule.ID) {
		return false, s.conflict(ctx, rule, result.Entry.ID, correlationID)
	}
	operation := "create"
	changed := true
	if result.Disposition == spiremgmt.CreateDispositionExisting {
		operation = "bind_owned_existing"
		changed = false
	}
	if err := s.recordAudit(ctx, rule, correlationID, operation, result.Entry.ID); err != nil {
		return false, err
	}
	if err := s.rules.MarkConverged(ctx, rule.ID, &result.Entry.ID); err != nil {
		return false, err
	}
	return changed, nil
}

func (s *Service) reconcileAbsent(ctx context.Context, rule registration.Rule, correlationID string) (bool, error) {
	if rule.SPIREEntryID == nil {
		if rule.ReconcileStatus != "converged" {
			if err := s.recordAudit(ctx, rule, correlationID, "absent_noop", ""); err != nil {
				return false, err
			}
			if err := s.rules.MarkConverged(ctx, rule.ID, nil); err != nil {
				return false, err
			}
		}
		return false, nil
	}

	actual, err := s.spire.GetEntry(ctx, *rule.SPIREEntryID)
	if errors.Is(err, spiremgmt.ErrNotFound) {
		if err := s.recordAudit(ctx, rule, correlationID, "clear_stale_binding", *rule.SPIREEntryID); err != nil {
			return false, err
		}
		if err := s.rules.MarkConverged(ctx, rule.ID, nil); err != nil {
			return false, err
		}
		return false, nil
	}
	if err != nil {
		return false, s.failRule(ctx, rule, "spire_unavailable", err)
	}
	if !ownedByRule(actual, rule.ID) {
		return false, s.conflict(ctx, rule, actual.ID, correlationID)
	}
	if err := s.spire.DeleteEntry(ctx, actual.ID); err != nil && !errors.Is(err, spiremgmt.ErrNotFound) {
		return false, s.failRule(ctx, rule, "spire_delete_failed", err)
	}
	if err := s.recordAudit(ctx, rule, correlationID, "delete", actual.ID); err != nil {
		return false, err
	}
	if err := s.rules.MarkConverged(ctx, rule.ID, nil); err != nil {
		return false, err
	}
	return true, nil
}

func desiredEntry(rule registration.Rule) (spiremgmt.Entry, error) {
	if rule.ParentSPIFFEID == nil || *rule.ParentSPIFFEID == "" {
		return spiremgmt.Entry{}, errors.New("parent SPIFFE ID is required")
	}
	if rule.X509SVIDTTLSeconds < 60 || rule.X509SVIDTTLSeconds > 86400 {
		return spiremgmt.Entry{}, errors.New("X.509-SVID TTL is outside the supported range")
	}
	if len(rule.Selectors) == 0 {
		return spiremgmt.Entry{}, errors.New("at least one workload selector is required")
	}
	return spiremgmt.Entry{
		SPIFFEID:       rule.WorkloadSPIFFEID,
		ParentSPIFFEID: *rule.ParentSPIFFEID,
		Selectors:      append([]string(nil), rule.Selectors...),
		X509SVIDTTL:    rule.X509SVIDTTLSeconds,
		Hint:           ownerHintPrefix + rule.ID,
	}, nil
}

func desiredWithoutID(entry spiremgmt.Entry) spiremgmt.Entry {
	entry.ID = ""
	return entry
}

func ownedByRule(entry spiremgmt.Entry, ruleID string) bool {
	return entry.Hint == ownerHintPrefix+ruleID
}

func entriesEquivalent(actual, desired spiremgmt.Entry) bool {
	if actual.SPIFFEID != desired.SPIFFEID || actual.ParentSPIFFEID != desired.ParentSPIFFEID || actual.X509SVIDTTL != desired.X509SVIDTTL || actual.Hint != desired.Hint {
		return false
	}
	left := append([]string(nil), actual.Selectors...)
	right := append([]string(nil), desired.Selectors...)
	sort.Strings(left)
	sort.Strings(right)
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func (s *Service) conflict(ctx context.Context, rule registration.Rule, entryID, correlationID string) error {
	if err := s.recordAudit(ctx, rule, correlationID, "ownership_conflict", entryID); err != nil {
		return err
	}
	return s.failRule(ctx, rule, "ownership_mismatch", errors.New("refusing to mutate SPIRE entry not owned by this registration rule"))
}

func (s *Service) failRule(ctx context.Context, rule registration.Rule, code string, cause error) error {
	if err := s.rules.MarkError(ctx, rule.ID, code); err != nil {
		return errors.Join(cause, err)
	}
	return cause
}

func (s *Service) recordAudit(ctx context.Context, rule registration.Rule, correlationID, operation, entryID string) error {
	metadata := map[string]string{"operation": operation}
	if entryID != "" {
		metadata["spire_entry_id"] = entryID
	}
	raw, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	organizationID := rule.OrganizationID
	_, err = s.audit.Append(ctx, audit.Event{
		OrganizationID: &organizationID,
		ActorType:      "system",
		ActorID:        "spire-reconciler",
		Action:         "spire.registration_reconcile",
		TargetType:     "registration_rule",
		TargetID:       rule.ID,
		CorrelationID:  correlationID,
		Metadata:       raw,
	})
	if err != nil {
		return fmt.Errorf("append reconciliation audit event: %w", err)
	}
	return nil
}

func newCorrelationID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "reconcile-correlation-unavailable"
	}
	return hex.EncodeToString(value[:])
}

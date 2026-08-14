package registration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Imad-cpp/workload-trust-platform/internal/audit"
	"github.com/jackc/pgx/v5"
)

type transactionBeginner interface {
	Begin(context.Context) (pgx.Tx, error)
}

type PostgresMutationService struct {
	db transactionBeginner
}

func NewPostgresMutationService(db transactionBeginner) *PostgresMutationService {
	return &PostgresMutationService{db: db}
}

func (s *PostgresMutationService) CreateDesired(ctx context.Context, actor MutationActor, correlationID string, input CreateDesiredInput) (MutationResult, error) {
	if err := validateActor(actor); err != nil {
		return MutationResult{}, err
	}
	if correlationID == "" {
		return MutationResult{}, fmt.Errorf("%w: correlation ID is required", ErrInvalidInput)
	}
	if err := validateCreateInput(input); err != nil {
		return MutationResult{}, err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return MutationResult{}, fmt.Errorf("begin registration create transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	var organizationID string
	if err := tx.QueryRow(ctx, `
		SELECT organization_id::text
		FROM workloads
		WHERE id = $1::uuid
	`, input.WorkloadID).Scan(&organizationID); errors.Is(err, pgx.ErrNoRows) {
		return MutationResult{}, ErrNotFound
	} else if err != nil {
		return MutationResult{}, fmt.Errorf("resolve workload for registration create: %w", err)
	}

	selectorsJSON, err := json.Marshal(input.Selectors)
	if err != nil {
		return MutationResult{}, fmt.Errorf("encode registration selectors: %w", err)
	}

	result := MutationResult{
		OrganizationID:     organizationID,
		WorkloadID:         input.WorkloadID,
		DesiredState:       "present",
		Revision:           1,
		X509SVIDTTLSeconds: input.X509SVIDTTLSeconds,
		SelectorCount:      len(input.Selectors),
		ReconcileStatus:    "pending",
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO registration_rules (
			organization_id,
			workload_id,
			selectors,
			desired_state,
			revision,
			parent_spiffe_id,
			x509_svid_ttl_seconds,
			reconcile_status,
			last_error_code
		) VALUES ($1::uuid, $2::uuid, $3::jsonb, 'present', 1, $4, $5, 'pending', NULL)
		RETURNING id::text
	`, organizationID, input.WorkloadID, string(selectorsJSON), input.ParentSPIFFEID, input.X509SVIDTTLSeconds).Scan(&result.ID); err != nil {
		return MutationResult{}, fmt.Errorf("create registration desired state: %w", err)
	}

	metadata, err := json.Marshal(map[string]any{
		"operation":             "create",
		"operator_role":         actor.Role,
		"desired_state":         result.DesiredState,
		"revision":              result.Revision,
		"selector_count":        result.SelectorCount,
		"x509_svid_ttl_seconds": result.X509SVIDTTLSeconds,
	})
	if err != nil {
		return MutationResult{}, err
	}
	if _, err := audit.NewPostgresRepository(tx).Append(ctx, audit.Event{
		OrganizationID: &organizationID,
		ActorType:      actor.Type,
		ActorID:        actor.ID,
		Action:         "registration_rule.create_desired_state",
		TargetType:     "registration_rule",
		TargetID:       result.ID,
		CorrelationID:  correlationID,
		Metadata:       metadata,
	}); err != nil {
		return MutationResult{}, fmt.Errorf("audit registration create: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return MutationResult{}, fmt.Errorf("commit registration create: %w", err)
	}
	return result, nil
}

func (s *PostgresMutationService) ReplaceDesired(ctx context.Context, actor MutationActor, correlationID, ruleID string, input ReplaceDesiredInput) (MutationResult, error) {
	if err := validateActor(actor); err != nil {
		return MutationResult{}, err
	}
	if correlationID == "" {
		return MutationResult{}, fmt.Errorf("%w: correlation ID is required", ErrInvalidInput)
	}
	if err := validateReplaceInput(ruleID, input); err != nil {
		return MutationResult{}, err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return MutationResult{}, fmt.Errorf("begin registration replace transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	var (
		organizationID string
		workloadID     string
		beforeState    string
		beforeRevision int64
		beforeTTL      int32
		beforeSelectors []byte
	)
	if err := tx.QueryRow(ctx, `
		SELECT
			organization_id::text,
			workload_id::text,
			desired_state,
			revision,
			x509_svid_ttl_seconds,
			selectors
		FROM registration_rules
		WHERE id = $1::uuid
		FOR UPDATE
	`, ruleID).Scan(&organizationID, &workloadID, &beforeState, &beforeRevision, &beforeTTL, &beforeSelectors); errors.Is(err, pgx.ErrNoRows) {
		return MutationResult{}, ErrNotFound
	} else if err != nil {
		return MutationResult{}, fmt.Errorf("load registration rule for replace: %w", err)
	}
	if beforeRevision != input.ExpectedRevision {
		return MutationResult{}, ErrConflict
	}

	var decodedBeforeSelectors []string
	if err := json.Unmarshal(beforeSelectors, &decodedBeforeSelectors); err != nil {
		return MutationResult{}, fmt.Errorf("decode existing registration selectors: %w", err)
	}
	selectorsJSON, err := json.Marshal(input.Selectors)
	if err != nil {
		return MutationResult{}, fmt.Errorf("encode replacement registration selectors: %w", err)
	}

	result := MutationResult{
		ID:                 ruleID,
		OrganizationID:     organizationID,
		WorkloadID:         workloadID,
		DesiredState:       input.DesiredState,
		Revision:           beforeRevision + 1,
		X509SVIDTTLSeconds: input.X509SVIDTTLSeconds,
		SelectorCount:      len(input.Selectors),
		ReconcileStatus:    "pending",
	}
	var updatedID string
	if err := tx.QueryRow(ctx, `
		UPDATE registration_rules
		SET
			selectors = $3::jsonb,
			desired_state = $4,
			revision = revision + 1,
			parent_spiffe_id = $5,
			x509_svid_ttl_seconds = $6,
			reconcile_status = 'pending',
			last_error_code = NULL,
			updated_at = now()
		WHERE id = $1::uuid AND revision = $2
		RETURNING id::text
	`, ruleID, input.ExpectedRevision, string(selectorsJSON), input.DesiredState, input.ParentSPIFFEID, input.X509SVIDTTLSeconds).Scan(&updatedID); errors.Is(err, pgx.ErrNoRows) {
		return MutationResult{}, ErrConflict
	} else if err != nil {
		return MutationResult{}, fmt.Errorf("replace registration desired state: %w", err)
	}

	metadata, err := json.Marshal(map[string]any{
		"operation":     "replace",
		"operator_role": actor.Role,
		"before": map[string]any{
			"desired_state":         beforeState,
			"revision":              beforeRevision,
			"selector_count":        len(decodedBeforeSelectors),
			"x509_svid_ttl_seconds": beforeTTL,
		},
		"after": map[string]any{
			"desired_state":         result.DesiredState,
			"revision":              result.Revision,
			"selector_count":        result.SelectorCount,
			"x509_svid_ttl_seconds": result.X509SVIDTTLSeconds,
		},
	})
	if err != nil {
		return MutationResult{}, err
	}
	if _, err := audit.NewPostgresRepository(tx).Append(ctx, audit.Event{
		OrganizationID: &organizationID,
		ActorType:      actor.Type,
		ActorID:        actor.ID,
		Action:         "registration_rule.replace_desired_state",
		TargetType:     "registration_rule",
		TargetID:       ruleID,
		CorrelationID:  correlationID,
		Metadata:       metadata,
	}); err != nil {
		return MutationResult{}, fmt.Errorf("audit registration replace: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return MutationResult{}, fmt.Errorf("commit registration replace: %w", err)
	}
	return result, nil
}

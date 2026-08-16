package policy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Imad-cpp/workload-trust-platform/internal/audit"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type transactionBeginner interface {
	Begin(context.Context) (pgx.Tx, error)
}

type PostgresManager struct {
	db transactionBeginner
}

func NewPostgresManager(db transactionBeginner) *PostgresManager {
	return &PostgresManager{db: db}
}

func (m *PostgresManager) Create(ctx context.Context, actor MutationActor, correlationID string, input CreateInput) (CreateResult, error) {
	if err := validateActor(actor); err != nil {
		return CreateResult{}, err
	}
	if correlationID == "" {
		return CreateResult{}, fmt.Errorf("%w: correlation ID is required", ErrInvalidInput)
	}
	input = normalizeCreateInput(input)
	if err := validateCreateInput(input); err != nil {
		return CreateResult{}, err
	}

	tx, err := m.db.Begin(ctx)
	if err != nil {
		return CreateResult{}, fmt.Errorf("begin policy create transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	var organizationExists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM organizations WHERE id = $1::uuid)
	`, input.OrganizationID).Scan(&organizationExists); err != nil {
		return CreateResult{}, fmt.Errorf("resolve policy organization: %w", err)
	}
	if !organizationExists {
		return CreateResult{}, ErrNotFound
	}

	result := CreateResult{
		Policy: PolicyResult{
			OrganizationID: input.OrganizationID,
			Name:           input.Name,
			Status:         "draft",
			Revision:       1,
		},
		Version: VersionResult{
			VersionNumber:  1,
			PolicyRevision: 1,
			Effect:         input.InitialVersion.Effect,
		},
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO access_policies (organization_id, name, status, revision)
		VALUES ($1::uuid, $2, 'draft', 1)
		RETURNING id::text
	`, input.OrganizationID, input.Name).Scan(&result.Policy.ID); err != nil {
		if isUniqueViolation(err) {
			return CreateResult{}, ErrConflict
		}
		return CreateResult{}, fmt.Errorf("create access policy: %w", err)
	}
	result.Version.PolicyID = result.Policy.ID

	if err := tx.QueryRow(ctx, `
		INSERT INTO access_policy_versions (
			policy_id,
			version,
			source_spiffe_id,
			destination_spiffe_id,
			action,
			effect,
			created_by,
			change_reason
		) VALUES ($1::uuid, 1, $2, $3, $4, $5, $6, $7)
		RETURNING id::text
	`,
		result.Policy.ID,
		input.InitialVersion.SourceSPIFFEID,
		input.InitialVersion.DestinationSPIFFEID,
		input.InitialVersion.Action,
		input.InitialVersion.Effect,
		actor.ID,
		input.InitialVersion.ChangeReason,
	).Scan(&result.Version.ID); err != nil {
		return CreateResult{}, fmt.Errorf("create initial policy version: %w", err)
	}

	if err := appendAudit(ctx, tx, actor, correlationID, input.OrganizationID, "access_policy.create_with_version", result.Policy.ID, map[string]any{
		"operation":       "create_with_version",
		"operator_role":   actor.Role,
		"policy_status":   result.Policy.Status,
		"policy_revision": result.Policy.Revision,
		"version_number":  result.Version.VersionNumber,
		"effect":          result.Version.Effect,
	}); err != nil {
		return CreateResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return CreateResult{}, fmt.Errorf("commit policy create: %w", err)
	}
	return result, nil
}

func (m *PostgresManager) AppendVersion(ctx context.Context, actor MutationActor, correlationID, policyID string, input AppendVersionInput) (VersionResult, error) {
	if err := validateActor(actor); err != nil {
		return VersionResult{}, err
	}
	if correlationID == "" {
		return VersionResult{}, fmt.Errorf("%w: correlation ID is required", ErrInvalidInput)
	}
	input = normalizeAppendInput(input)
	if err := validateAppendInput(policyID, input); err != nil {
		return VersionResult{}, err
	}

	tx, err := m.db.Begin(ctx)
	if err != nil {
		return VersionResult{}, fmt.Errorf("begin policy version transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	var (
		organizationID  string
		currentRevision int64
	)
	if err := tx.QueryRow(ctx, `
		SELECT organization_id::text, revision
		FROM access_policies
		WHERE id = $1::uuid
		FOR UPDATE
	`, policyID).Scan(&organizationID, &currentRevision); errors.Is(err, pgx.ErrNoRows) {
		return VersionResult{}, ErrNotFound
	} else if err != nil {
		return VersionResult{}, fmt.Errorf("load access policy for version append: %w", err)
	}
	if currentRevision != input.ExpectedRevision {
		return VersionResult{}, ErrConflict
	}

	var nextVersion int64
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(version), 0) + 1
		FROM access_policy_versions
		WHERE policy_id = $1::uuid
	`, policyID).Scan(&nextVersion); err != nil {
		return VersionResult{}, fmt.Errorf("calculate next policy version: %w", err)
	}

	result := VersionResult{
		PolicyID:       policyID,
		VersionNumber:  nextVersion,
		PolicyRevision: currentRevision + 1,
		Effect:         input.Version.Effect,
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO access_policy_versions (
			policy_id,
			version,
			source_spiffe_id,
			destination_spiffe_id,
			action,
			effect,
			created_by,
			change_reason
		) VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id::text
	`,
		policyID,
		nextVersion,
		input.Version.SourceSPIFFEID,
		input.Version.DestinationSPIFFEID,
		input.Version.Action,
		input.Version.Effect,
		actor.ID,
		input.Version.ChangeReason,
	).Scan(&result.ID); err != nil {
		if isUniqueViolation(err) {
			return VersionResult{}, ErrConflict
		}
		return VersionResult{}, fmt.Errorf("append policy version: %w", err)
	}

	command, err := tx.Exec(ctx, `
		UPDATE access_policies
		SET revision = revision + 1, updated_at = now()
		WHERE id = $1::uuid AND revision = $2
	`, policyID, input.ExpectedRevision)
	if err != nil {
		return VersionResult{}, fmt.Errorf("advance policy revision: %w", err)
	}
	if command.RowsAffected() != 1 {
		return VersionResult{}, ErrConflict
	}

	if err := appendAudit(ctx, tx, actor, correlationID, organizationID, "access_policy.append_version", policyID, map[string]any{
		"operation":       "append_version",
		"operator_role":   actor.Role,
		"policy_revision": result.PolicyRevision,
		"version_number":  result.VersionNumber,
		"effect":          result.Effect,
	}); err != nil {
		return VersionResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return VersionResult{}, fmt.Errorf("commit policy version append: %w", err)
	}
	return result, nil
}

func (m *PostgresManager) Activate(ctx context.Context, actor MutationActor, correlationID, policyID string, input ActivateInput) (PolicyResult, error) {
	if err := validateActor(actor); err != nil {
		return PolicyResult{}, err
	}
	if correlationID == "" {
		return PolicyResult{}, fmt.Errorf("%w: correlation ID is required", ErrInvalidInput)
	}
	if err := validateActivateInput(policyID, input); err != nil {
		return PolicyResult{}, err
	}

	tx, err := m.db.Begin(ctx)
	if err != nil {
		return PolicyResult{}, fmt.Errorf("begin policy activation transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	result := PolicyResult{ID: policyID}
	var activeVersionID *string
	if err := tx.QueryRow(ctx, `
		SELECT organization_id::text, name, status, revision, active_version_id::text
		FROM access_policies
		WHERE id = $1::uuid
		FOR UPDATE
	`, policyID).Scan(
		&result.OrganizationID,
		&result.Name,
		&result.Status,
		&result.Revision,
		&activeVersionID,
	); errors.Is(err, pgx.ErrNoRows) {
		return PolicyResult{}, ErrNotFound
	} else if err != nil {
		return PolicyResult{}, fmt.Errorf("load policy for activation: %w", err)
	}
	if result.Revision != input.ExpectedRevision {
		return PolicyResult{}, ErrConflict
	}

	var (
		versionNumber int64
		storedRule    RuleInput
	)
	if err := tx.QueryRow(ctx, `
		SELECT version, source_spiffe_id, destination_spiffe_id, action, effect, change_reason
		FROM access_policy_versions
		WHERE id = $1::uuid AND policy_id = $2::uuid
	`, input.VersionID, policyID).Scan(
		&versionNumber,
		&storedRule.SourceSPIFFEID,
		&storedRule.DestinationSPIFFEID,
		&storedRule.Action,
		&storedRule.Effect,
		&storedRule.ChangeReason,
	); errors.Is(err, pgx.ErrNoRows) {
		return PolicyResult{}, ErrNotFound
	} else if err != nil {
		return PolicyResult{}, fmt.Errorf("load policy version for activation: %w", err)
	}
	if err := validateRuleInput(storedRule); err != nil {
		return PolicyResult{}, fmt.Errorf("%w: stored version is not activatable", ErrInvalidInput)
	}

	command, err := tx.Exec(ctx, `
		UPDATE access_policies
		SET active_version_id = $3::uuid, status = 'active', revision = revision + 1, updated_at = now()
		WHERE id = $1::uuid AND revision = $2
	`, policyID, input.ExpectedRevision, input.VersionID)
	if err != nil {
		return PolicyResult{}, fmt.Errorf("activate policy version: %w", err)
	}
	if command.RowsAffected() != 1 {
		return PolicyResult{}, ErrConflict
	}

	result.Status = "active"
	result.Revision = input.ExpectedRevision + 1
	active := input.VersionID
	result.ActiveVersionID = &active

	if err := appendAudit(ctx, tx, actor, correlationID, result.OrganizationID, "access_policy.activate_version", policyID, map[string]any{
		"operation":       "activate_version",
		"operator_role":   actor.Role,
		"policy_revision": result.Revision,
		"version_id":      input.VersionID,
		"version_number":  versionNumber,
	}); err != nil {
		return PolicyResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return PolicyResult{}, fmt.Errorf("commit policy activation: %w", err)
	}
	return result, nil
}

func appendAudit(ctx context.Context, tx pgx.Tx, actor MutationActor, correlationID, organizationID, action, targetID string, metadata map[string]any) error {
	raw, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("encode policy audit metadata: %w", err)
	}
	if _, err := audit.NewPostgresRepository(tx).Append(ctx, audit.Event{
		OrganizationID: &organizationID,
		ActorType:      actor.Type,
		ActorID:        actor.ID,
		Action:         action,
		TargetType:     "access_policy",
		TargetID:       targetID,
		CorrelationID:  correlationID,
		Metadata:       raw,
	}); err != nil {
		return fmt.Errorf("append policy audit event: %w", err)
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

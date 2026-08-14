package registration

import "context"

type Store interface {
	List(ctx context.Context) ([]Rule, error)
	MarkConverged(ctx context.Context, ruleID string, spireEntryID *string) error
	MarkError(ctx context.Context, ruleID, errorCode string) error
}

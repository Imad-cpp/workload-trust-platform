package audit

import "context"

// Appender intentionally exposes append only. Audit history has no update or
// delete operation in the application boundary, matching the database guard.
type Appender interface {
	Append(ctx context.Context, event Event) (Event, error)
}

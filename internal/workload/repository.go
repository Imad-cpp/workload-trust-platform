package workload

import "context"

type Lister interface {
	ListByOrganization(ctx context.Context, organizationID string) ([]Workload, error)
}

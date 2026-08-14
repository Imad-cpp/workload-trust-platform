package spiremgmt

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	entryv1 "github.com/spiffe/spire-api-sdk/proto/spire/api/server/entry/v1"
	types "github.com/spiffe/spire-api-sdk/proto/spire/api/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

var ErrNotFound = errors.New("SPIRE entry not found")

type Entry struct {
	ID              string
	SPIFFEID        string
	ParentSPIFFEID  string
	Selectors       []string
	X509SVIDTTL     int32
	Hint            string
}

type CreateDisposition string

const (
	CreateDispositionCreated  CreateDisposition = "created"
	CreateDispositionExisting CreateDisposition = "existing"
)

type CreateResult struct {
	Entry       Entry
	Disposition CreateDisposition
}

type Client interface {
	GetEntry(ctx context.Context, id string) (Entry, error)
	CreateEntry(ctx context.Context, entry Entry) (CreateResult, error)
	UpdateEntry(ctx context.Context, entry Entry) (Entry, error)
	DeleteEntry(ctx context.Context, id string) error
}

type GRPCClient struct {
	conn  *grpc.ClientConn
	entry entryv1.EntryClient
}

func NewUnixClient(socketPath string) (*GRPCClient, error) {
	if !strings.HasPrefix(socketPath, "/") {
		return nil, errors.New("SPIRE server socket path must be absolute")
	}
	conn, err := grpc.NewClient(
		"unix:"+socketPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			path := strings.TrimPrefix(addr, "unix:")
			return (&net.Dialer{}).DialContext(ctx, "unix", path)
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("create SPIRE management client: %w", err)
	}
	return &GRPCClient{conn: conn, entry: entryv1.NewEntryClient(conn)}, nil
}

func (c *GRPCClient) Close() error {
	return c.conn.Close()
}

func (c *GRPCClient) GetEntry(ctx context.Context, id string) (Entry, error) {
	value, err := c.entry.GetEntry(ctx, &entryv1.GetEntryRequest{ID: id, OutputMask: fullOutputMask()})
	if status.Code(err) == codes.NotFound {
		return Entry{}, ErrNotFound
	}
	if err != nil {
		return Entry{}, fmt.Errorf("get SPIRE entry: %w", err)
	}
	return fromProtoEntry(value), nil
}

func (c *GRPCClient) CreateEntry(ctx context.Context, entry Entry) (CreateResult, error) {
	value, err := toProtoEntry(entry)
	if err != nil {
		return CreateResult{}, err
	}
	response, err := c.entry.BatchCreateEntry(ctx, &entryv1.BatchCreateEntryRequest{
		Entries:    []*types.Entry{value},
		OutputMask: fullOutputMask(),
	})
	if err != nil {
		return CreateResult{}, fmt.Errorf("create SPIRE entry: %w", err)
	}
	if len(response.Results) != 1 || response.Results[0].Status == nil || response.Results[0].Entry == nil {
		return CreateResult{}, errors.New("SPIRE create returned an invalid result")
	}
	result := response.Results[0]
	switch codes.Code(result.Status.Code) {
	case codes.OK:
		return CreateResult{Entry: fromProtoEntry(result.Entry), Disposition: CreateDispositionCreated}, nil
	case codes.AlreadyExists:
		return CreateResult{Entry: fromProtoEntry(result.Entry), Disposition: CreateDispositionExisting}, nil
	default:
		return CreateResult{}, fmt.Errorf("SPIRE create rejected entry with status %s", codes.Code(result.Status.Code))
	}
}

func (c *GRPCClient) UpdateEntry(ctx context.Context, entry Entry) (Entry, error) {
	if entry.ID == "" {
		return Entry{}, errors.New("SPIRE entry ID is required for update")
	}
	value, err := toProtoEntry(entry)
	if err != nil {
		return Entry{}, err
	}
	response, err := c.entry.BatchUpdateEntry(ctx, &entryv1.BatchUpdateEntryRequest{
		Entries: []*types.Entry{value},
		InputMask: &types.EntryMask{
			SpiffeId:    true,
			ParentId:    true,
			Selectors:   true,
			X509SvidTtl: true,
			Hint:        true,
		},
		OutputMask: fullOutputMask(),
	})
	if err != nil {
		return Entry{}, fmt.Errorf("update SPIRE entry: %w", err)
	}
	if len(response.Results) != 1 || response.Results[0].Status == nil || response.Results[0].Entry == nil {
		return Entry{}, errors.New("SPIRE update returned an invalid result")
	}
	result := response.Results[0]
	if codes.Code(result.Status.Code) != codes.OK {
		if codes.Code(result.Status.Code) == codes.NotFound {
			return Entry{}, ErrNotFound
		}
		return Entry{}, fmt.Errorf("SPIRE update rejected entry with status %s", codes.Code(result.Status.Code))
	}
	return fromProtoEntry(result.Entry), nil
}

func (c *GRPCClient) DeleteEntry(ctx context.Context, id string) error {
	response, err := c.entry.BatchDeleteEntry(ctx, &entryv1.BatchDeleteEntryRequest{Ids: []string{id}})
	if err != nil {
		return fmt.Errorf("delete SPIRE entry: %w", err)
	}
	if len(response.Results) != 1 || response.Results[0].Status == nil {
		return errors.New("SPIRE delete returned an invalid result")
	}
	code := codes.Code(response.Results[0].Status.Code)
	if code == codes.NotFound {
		return ErrNotFound
	}
	if code != codes.OK {
		return fmt.Errorf("SPIRE delete rejected entry with status %s", code)
	}
	return nil
}

func toProtoEntry(entry Entry) (*types.Entry, error) {
	spiffeID, err := parseSPIFFEID(entry.SPIFFEID)
	if err != nil {
		return nil, fmt.Errorf("invalid workload SPIFFE ID: %w", err)
	}
	parentID, err := parseSPIFFEID(entry.ParentSPIFFEID)
	if err != nil {
		return nil, fmt.Errorf("invalid parent SPIFFE ID: %w", err)
	}
	selectors := make([]*types.Selector, 0, len(entry.Selectors))
	for _, raw := range entry.Selectors {
		selector, err := parseSelector(raw)
		if err != nil {
			return nil, err
		}
		selectors = append(selectors, selector)
	}
	if len(selectors) == 0 {
		return nil, errors.New("at least one selector is required")
	}
	return &types.Entry{
		Id:           entry.ID,
		SpiffeId:     spiffeID,
		ParentId:     parentID,
		Selectors:    selectors,
		X509SvidTtl:  entry.X509SVIDTTL,
		Hint:         entry.Hint,
	}, nil
}

func fromProtoEntry(entry *types.Entry) Entry {
	selectors := make([]string, 0, len(entry.GetSelectors()))
	for _, selector := range entry.GetSelectors() {
		selectors = append(selectors, selector.GetType()+":"+selector.GetValue())
	}
	return Entry{
		ID:             entry.GetId(),
		SPIFFEID:       formatSPIFFEID(entry.GetSpiffeId()),
		ParentSPIFFEID: formatSPIFFEID(entry.GetParentId()),
		Selectors:      selectors,
		X509SVIDTTL:    entry.GetX509SvidTtl(),
		Hint:           entry.GetHint(),
	}
}

func parseSPIFFEID(raw string) (*types.SPIFFEID, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme != "spiffe" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Opaque != "" {
		return nil, errors.New("SPIFFE ID must be an absolute spiffe URI without userinfo, query, or fragment")
	}
	if parsed.Hostname() != parsed.Host || strings.ToLower(parsed.Host) != parsed.Host {
		return nil, errors.New("SPIFFE trust domain must be canonical and must not contain a port")
	}
	if parsed.Path == "" || !strings.HasPrefix(parsed.Path, "/") || parsed.RawPath != "" {
		return nil, errors.New("SPIFFE ID path must be a canonical absolute path")
	}
	return &types.SPIFFEID{TrustDomain: parsed.Host, Path: parsed.Path}, nil
}

func formatSPIFFEID(id *types.SPIFFEID) string {
	if id == nil {
		return ""
	}
	return "spiffe://" + id.GetTrustDomain() + id.GetPath()
}

func parseSelector(raw string) (*types.Selector, error) {
	kind, value, ok := strings.Cut(raw, ":")
	if !ok || kind == "" || value == "" {
		return nil, fmt.Errorf("selector %q must use type:value format", raw)
	}
	return &types.Selector{Type: kind, Value: value}, nil
}

func fullOutputMask() *types.EntryMask {
	return &types.EntryMask{
		SpiffeId:    true,
		ParentId:    true,
		Selectors:   true,
		X509SvidTtl: true,
		Hint:        true,
	}
}

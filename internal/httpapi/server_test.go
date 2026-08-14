package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Imad-cpp/workload-trust-platform/internal/workload"
)

type readinessStub struct{ err error }

func (s readinessStub) Ping(context.Context) error { return s.err }

type workloadStub struct {
	items []workload.Workload
	err   error
}

func (s workloadStub) ListByOrganization(context.Context, string) ([]workload.Workload, error) {
	return s.items, s.err
}

func newTestHandler(t *testing.T, readinessErr error, lister workload.Lister) http.Handler {
	t.Helper()
	server, err := New(Dependencies{Readiness: readinessStub{err: readinessErr}, Workloads: lister})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return server.Handler()
}

func TestHealth(t *testing.T) {
	h := newTestHandler(t, nil, workloadStub{})
	r := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("expected Cache-Control: no-store")
	}
	if w.Header().Get("X-Request-ID") == "" {
		t.Fatal("expected request ID")
	}
}

func TestReadinessFailureIsGeneric(t *testing.T) {
	h := newTestHandler(t, errors.New("postgres password=do-not-leak"), workloadStub{})
	r := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
	if body := w.Body.String(); strings.Contains(body, "password") || strings.Contains(body, "do-not-leak") {
		t.Fatalf("response leaked readiness error: %s", body)
	}
}

func TestListWorkloadsRequiresUUID(t *testing.T) {
	h := newTestHandler(t, nil, workloadStub{})
	r := httptest.NewRequest(http.MethodGet, "/v1/workloads?organization_id=not-a-uuid", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestListWorkloadsReturnsData(t *testing.T) {
	items := []workload.Workload{{ID: "w1", Name: "frontend", SPIFFEID: "spiffe://workload-trust.test/lab/frontend", Status: "healthy"}}
	h := newTestHandler(t, nil, workloadStub{items: items})
	r := httptest.NewRequest(http.MethodGet, "/v1/workloads?organization_id=123e4567-e89b-12d3-a456-426614174000", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var response workloadListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Data) != 1 || response.Data[0].Name != "frontend" {
		t.Fatalf("unexpected data: %#v", response.Data)
	}
}

func TestMutationEndpointDoesNotExist(t *testing.T) {
	h := newTestHandler(t, nil, workloadStub{})
	r := httptest.NewRequest(http.MethodPost, "/v1/workloads", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
	if w.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("Allow = %q, want GET", w.Header().Get("Allow"))
	}
	if !strings.Contains(w.Body.String(), `"code":"method_not_allowed"`) {
		t.Fatalf("expected stable JSON method error: %s", w.Body.String())
	}
}

func TestUnknownRouteReturnsStableJSON(t *testing.T) {
	h := newTestHandler(t, nil, workloadStub{})
	r := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
	if !strings.Contains(w.Body.String(), `"code":"not_found"`) {
		t.Fatalf("expected stable JSON not-found error: %s", w.Body.String())
	}
}

func TestRepositoryErrorIsNotLeaked(t *testing.T) {
	h := newTestHandler(t, nil, workloadStub{err: errors.New("postgres secret internal detail")})
	r := httptest.NewRequest(http.MethodGet, "/v1/workloads?organization_id=123e4567-e89b-12d3-a456-426614174000", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	if body := w.Body.String(); strings.Contains(body, "postgres") || strings.Contains(body, "secret") {
		t.Fatalf("response leaked repository error: %s", body)
	}
}

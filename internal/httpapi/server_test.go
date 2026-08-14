package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Imad-cpp/workload-trust-platform/internal/operatorauth"
	"github.com/Imad-cpp/workload-trust-platform/internal/workload"
)

const testOperatorToken = "0123456789abcdef0123456789abcdef"

type readinessStub struct{ err error }

func (s readinessStub) Ping(context.Context) error { return s.err }

type workloadStub struct {
	items []workload.Workload
	err   error
}

func (s workloadStub) ListByOrganization(context.Context, string) ([]workload.Workload, error) {
	return s.items, s.err
}

func newTestHandler(t *testing.T, role operatorauth.Role, readinessErr error, lister workload.Lister) http.Handler {
	t.Helper()
	authenticator, err := operatorauth.NewStaticBearer(testOperatorToken, "test-operator", role)
	if err != nil {
		t.Fatalf("NewStaticBearer() error = %v", err)
	}
	server, err := New(Dependencies{
		Readiness:     readinessStub{err: readinessErr},
		Workloads:     lister,
		Authenticator: authenticator,
		Authorizer:    operatorauth.RBAC{},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return server.Handler()
}

func authenticatedRequest(method, target string) *http.Request {
	r := httptest.NewRequest(method, target, nil)
	r.Header.Set("Authorization", "Bearer "+testOperatorToken)
	return r
}

func TestHealth(t *testing.T) {
	h := newTestHandler(t, operatorauth.RoleViewer, nil, workloadStub{})
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
	h := newTestHandler(t, operatorauth.RoleViewer, errors.New("postgres password=do-not-leak"), workloadStub{})
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

func TestOperatorAPIRequiresAuthentication(t *testing.T) {
	h := newTestHandler(t, operatorauth.RoleViewer, nil, workloadStub{})
	r := httptest.NewRequest(http.MethodGet, "/v1/workloads?organization_id=123e4567-e89b-12d3-a456-426614174000", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	if w.Header().Get("WWW-Authenticate") == "" {
		t.Fatal("expected WWW-Authenticate header")
	}
	if !strings.Contains(w.Body.String(), `"code":"unauthorized"`) {
		t.Fatalf("expected stable unauthorized error: %s", w.Body.String())
	}
}

func TestViewerCanReadWorkloads(t *testing.T) {
	items := []workload.Workload{{ID: "w1", Name: "frontend", SPIFFEID: "spiffe://workload-trust.test/lab/frontend", Status: "healthy"}}
	h := newTestHandler(t, operatorauth.RoleViewer, nil, workloadStub{items: items})
	r := authenticatedRequest(http.MethodGet, "/v1/workloads?organization_id=123e4567-e89b-12d3-a456-426614174000")
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

func TestListWorkloadsRequiresUUID(t *testing.T) {
	h := newTestHandler(t, operatorauth.RoleViewer, nil, workloadStub{})
	r := authenticatedRequest(http.MethodGet, "/v1/workloads?organization_id=not-a-uuid")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestMutationEndpointStillDoesNotExist(t *testing.T) {
	h := newTestHandler(t, operatorauth.RoleOperator, nil, workloadStub{})
	r := authenticatedRequest(http.MethodPost, "/v1/workloads")
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

func TestUnknownOperatorRouteRequiresAuthenticationFirst(t *testing.T) {
	h := newTestHandler(t, operatorauth.RoleViewer, nil, workloadStub{})
	r := httptest.NewRequest(http.MethodGet, "/v1/unknown", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestUnknownAuthenticatedRouteReturnsStableJSON(t *testing.T) {
	h := newTestHandler(t, operatorauth.RoleViewer, nil, workloadStub{})
	r := authenticatedRequest(http.MethodGet, "/v1/unknown")
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
	h := newTestHandler(t, operatorauth.RoleViewer, nil, workloadStub{err: errors.New("postgres secret internal detail")})
	r := authenticatedRequest(http.MethodGet, "/v1/workloads?organization_id=123e4567-e89b-12d3-a456-426614174000")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	if body := w.Body.String(); strings.Contains(body, "postgres") || strings.Contains(body, "secret") {
		t.Fatalf("response leaked repository error: %s", body)
	}
}

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Imad-cpp/workload-trust-platform/internal/operatorauth"
	"github.com/Imad-cpp/workload-trust-platform/internal/policy"
	"github.com/Imad-cpp/workload-trust-platform/internal/registration"
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

type mutationStub struct {
	createResult    registration.MutationResult
	createErr       error
	replaceResult   registration.MutationResult
	replaceErr      error
	createCalls     int
	replaceCalls    int
	lastActor       registration.MutationActor
	lastCorrelation string
}

func (s *mutationStub) CreateDesired(_ context.Context, actor registration.MutationActor, correlationID string, _ registration.CreateDesiredInput) (registration.MutationResult, error) {
	s.createCalls++
	s.lastActor = actor
	s.lastCorrelation = correlationID
	return s.createResult, s.createErr
}

func (s *mutationStub) ReplaceDesired(_ context.Context, actor registration.MutationActor, correlationID, _ string, _ registration.ReplaceDesiredInput) (registration.MutationResult, error) {
	s.replaceCalls++
	s.lastActor = actor
	s.lastCorrelation = correlationID
	return s.replaceResult, s.replaceErr
}

type policyManagerStub struct{}

func (policyManagerStub) Create(context.Context, policy.MutationActor, string, policy.CreateInput) (policy.CreateResult, error) {
	return policy.CreateResult{}, nil
}

func (policyManagerStub) AppendVersion(context.Context, policy.MutationActor, string, string, policy.AppendVersionInput) (policy.VersionResult, error) {
	return policy.VersionResult{}, nil
}

func (policyManagerStub) Activate(context.Context, policy.MutationActor, string, string, policy.ActivateInput) (policy.PolicyResult, error) {
	return policy.PolicyResult{}, nil
}

func newTestHandler(t *testing.T, role operatorauth.Role, readinessErr error, lister workload.Lister, mutators ...registration.Mutator) http.Handler {
	t.Helper()
	authenticator, err := operatorauth.NewStaticBearer(testOperatorToken, "test-operator", role)
	if err != nil {
		t.Fatalf("NewStaticBearer() error = %v", err)
	}
	var mutator registration.Mutator = &mutationStub{}
	if len(mutators) > 0 {
		mutator = mutators[0]
	}
	server, err := New(Dependencies{
		Readiness:             readinessStub{err: readinessErr},
		Workloads:             lister,
		Diagnostics:           &diagnosticsStub{},
		RegistrationMutations: mutator,
		PolicyManager:         policyManagerStub{},
		Authenticator:         authenticator,
		Authorizer:            operatorauth.RBAC{},
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

func authenticatedJSONRequest(method, target, body string) *http.Request {
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer "+testOperatorToken)
	r.Header.Set("Content-Type", "application/json")
	return r
}

func validCreateBody() string {
	return `{"workload_id":"123e4567-e89b-12d3-a456-426614174002","selectors":["docker:label:service:api"],"parent_spiffe_id":"spiffe://workload-trust.test/spire/agent/test","x509_svid_ttl_seconds":300}`
}

func TestHealth(t *testing.T) {
	h := newTestHandler(t, operatorauth.RoleViewer, nil, workloadStub{})
	r := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("X-Request-ID") == "" {
		t.Fatal("expected security headers and request ID")
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
	if w.Code != http.StatusUnauthorized || w.Header().Get("WWW-Authenticate") == "" {
		t.Fatalf("unexpected authentication response: status=%d", w.Code)
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
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil || len(response.Data) != 1 {
		t.Fatalf("unexpected workload response: data=%#v err=%v", response.Data, err)
	}
}

func TestViewerCannotCreateRegistrationRule(t *testing.T) {
	mutator := &mutationStub{}
	h := newTestHandler(t, operatorauth.RoleViewer, nil, workloadStub{}, mutator)
	r := authenticatedJSONRequest(http.MethodPost, "/v1/registration-rules", validCreateBody())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d: %s", w.Code, http.StatusForbidden, w.Body.String())
	}
	if mutator.createCalls != 0 {
		t.Fatal("unauthorized viewer reached mutation service")
	}
}

func TestMutationRequiresAuthenticationBeforeAuthorization(t *testing.T) {
	mutator := &mutationStub{}
	h := newTestHandler(t, operatorauth.RoleOperator, nil, workloadStub{}, mutator)
	r := httptest.NewRequest(http.MethodPost, "/v1/registration-rules", strings.NewReader(validCreateBody()))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	if mutator.createCalls != 0 {
		t.Fatal("unauthenticated request reached mutation service")
	}
}

func TestOperatorCreatesRegistrationRule(t *testing.T) {
	mutator := &mutationStub{createResult: registration.MutationResult{
		ID:                 "123e4567-e89b-12d3-a456-426614174010",
		OrganizationID:     "123e4567-e89b-12d3-a456-426614174000",
		WorkloadID:         "123e4567-e89b-12d3-a456-426614174002",
		DesiredState:       "present",
		Revision:           1,
		X509SVIDTTLSeconds: 300,
		SelectorCount:      1,
		ReconcileStatus:    "pending",
	}}
	h := newTestHandler(t, operatorauth.RoleOperator, nil, workloadStub{}, mutator)
	body := `{"workload_id":"123e4567-e89b-12d3-a456-426614174002","selectors":["docker:label:service:api"],"parent_spiffe_id":"spiffe://workload-trust.test/spire/agent/join_token/do-not-return","x509_svid_ttl_seconds":300}`
	r := authenticatedJSONRequest(http.MethodPost, "/v1/registration-rules", body)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", w.Code, http.StatusCreated, w.Body.String())
	}
	if mutator.createCalls != 1 || mutator.lastActor.Role != "operator" || mutator.lastActor.ID != "test-operator" || mutator.lastCorrelation == "" {
		t.Fatalf("unexpected mutation attribution: %#v correlation=%q", mutator.lastActor, mutator.lastCorrelation)
	}
	if response := w.Body.String(); strings.Contains(response, "join_token") || strings.Contains(response, "do-not-return") || strings.Contains(response, "selectors") {
		t.Fatalf("mutation response exposed sensitive desired-state material: %s", response)
	}
	if w.Header().Get("Location") == "" {
		t.Fatal("expected Location header")
	}
}

func TestCreateRegistrationRejectsUnknownJSONField(t *testing.T) {
	mutator := &mutationStub{}
	h := newTestHandler(t, operatorauth.RoleOperator, nil, workloadStub{}, mutator)
	body := strings.TrimSuffix(validCreateBody(), "}") + `,"unexpected":true}`
	r := authenticatedJSONRequest(http.MethodPost, "/v1/registration-rules", body)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest || mutator.createCalls != 0 {
		t.Fatalf("unexpected unknown-field result: status=%d calls=%d body=%s", w.Code, mutator.createCalls, w.Body.String())
	}
}

func TestCreateRegistrationRequiresJSONContentType(t *testing.T) {
	mutator := &mutationStub{}
	h := newTestHandler(t, operatorauth.RoleOperator, nil, workloadStub{}, mutator)
	r := authenticatedRequest(http.MethodPost, "/v1/registration-rules")
	r.Body = io.NopCloser(strings.NewReader(validCreateBody()))
	r.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnsupportedMediaType || mutator.createCalls != 0 {
		t.Fatalf("unexpected media-type result: status=%d calls=%d", w.Code, mutator.createCalls)
	}
}

func TestCreateRegistrationRejectsOversizedBody(t *testing.T) {
	mutator := &mutationStub{}
	h := newTestHandler(t, operatorauth.RoleOperator, nil, workloadStub{}, mutator)
	body := `{"workload_id":"123e4567-e89b-12d3-a456-426614174002","selectors":["docker:` + strings.Repeat("x", 70*1024) + `"],"parent_spiffe_id":"spiffe://workload-trust.test/agent","x509_svid_ttl_seconds":300}`
	r := authenticatedJSONRequest(http.MethodPost, "/v1/registration-rules", body)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusRequestEntityTooLarge || mutator.createCalls != 0 {
		t.Fatalf("unexpected oversized result: status=%d calls=%d", w.Code, mutator.createCalls)
	}
}

func TestReplaceRegistrationMapsRevisionConflict(t *testing.T) {
	mutator := &mutationStub{replaceErr: registration.ErrConflict}
	h := newTestHandler(t, operatorauth.RoleOperator, nil, workloadStub{}, mutator)
	body := `{"expected_revision":1,"desired_state":"absent","selectors":["docker:label:service:api"],"parent_spiffe_id":"spiffe://workload-trust.test/agent","x509_svid_ttl_seconds":300}`
	r := authenticatedJSONRequest(http.MethodPatch, "/v1/registration-rules/123e4567-e89b-12d3-a456-426614174010", body)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusConflict || mutator.replaceCalls != 1 {
		t.Fatalf("unexpected revision conflict result: status=%d calls=%d body=%s", w.Code, mutator.replaceCalls, w.Body.String())
	}
}

func TestMutationInternalErrorIsGeneric(t *testing.T) {
	mutator := &mutationStub{createErr: errors.New("postgres password=do-not-leak")}
	h := newTestHandler(t, operatorauth.RoleOperator, nil, workloadStub{}, mutator)
	r := authenticatedJSONRequest(http.MethodPost, "/v1/registration-rules", validCreateBody())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	if body := w.Body.String(); strings.Contains(body, "postgres") || strings.Contains(body, "password") || strings.Contains(body, "do-not-leak") {
		t.Fatalf("mutation response leaked internal error: %s", body)
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
	if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), `"code":"not_found"`) {
		t.Fatalf("unexpected not-found response: status=%d body=%s", w.Code, w.Body.String())
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

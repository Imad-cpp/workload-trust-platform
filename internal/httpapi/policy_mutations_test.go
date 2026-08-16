package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Imad-cpp/workload-trust-platform/internal/operatorauth"
	"github.com/Imad-cpp/workload-trust-platform/internal/policy"
	"github.com/Imad-cpp/workload-trust-platform/internal/registration"
	"github.com/Imad-cpp/workload-trust-platform/internal/workload"
)

type policyHTTPManagerStub struct {
	createResult    policy.CreateResult
	createErr       error
	appendResult    policy.VersionResult
	appendErr       error
	activateResult  policy.PolicyResult
	activateErr     error
	createCalls     int
	appendCalls     int
	activateCalls   int
	lastActor       policy.MutationActor
	lastCorrelation string
}

func (s *policyHTTPManagerStub) Create(_ context.Context, actor policy.MutationActor, correlationID string, _ policy.CreateInput) (policy.CreateResult, error) {
	s.createCalls++
	s.lastActor = actor
	s.lastCorrelation = correlationID
	return s.createResult, s.createErr
}

func (s *policyHTTPManagerStub) AppendVersion(_ context.Context, actor policy.MutationActor, correlationID, _ string, _ policy.AppendVersionInput) (policy.VersionResult, error) {
	s.appendCalls++
	s.lastActor = actor
	s.lastCorrelation = correlationID
	return s.appendResult, s.appendErr
}

func (s *policyHTTPManagerStub) Activate(_ context.Context, actor policy.MutationActor, correlationID, _ string, _ policy.ActivateInput) (policy.PolicyResult, error) {
	s.activateCalls++
	s.lastActor = actor
	s.lastCorrelation = correlationID
	return s.activateResult, s.activateErr
}

type policyRegistrationStub struct{}

func (policyRegistrationStub) CreateDesired(context.Context, registration.MutationActor, string, registration.CreateDesiredInput) (registration.MutationResult, error) {
	return registration.MutationResult{}, nil
}

func (policyRegistrationStub) ReplaceDesired(context.Context, registration.MutationActor, string, string, registration.ReplaceDesiredInput) (registration.MutationResult, error) {
	return registration.MutationResult{}, nil
}

type policyWorkloadStub struct{}

func (policyWorkloadStub) ListByOrganization(context.Context, string) ([]workload.Workload, error) {
	return nil, nil
}

type policyReadinessStub struct{}

func (policyReadinessStub) Ping(context.Context) error { return nil }

func newPolicyHTTPHandler(t *testing.T, role operatorauth.Role, manager policy.Manager) http.Handler {
	t.Helper()
	authenticator, err := operatorauth.NewStaticBearer(testOperatorToken, "policy-test-operator", role)
	if err != nil {
		t.Fatalf("NewStaticBearer() error = %v", err)
	}
	server, err := New(Dependencies{
		Readiness:             policyReadinessStub{},
		Workloads:             policyWorkloadStub{},
		Diagnostics:           &diagnosticsStub{},
		RegistrationMutations: policyRegistrationStub{},
		PolicyManager:         manager,
		Authenticator:         authenticator,
		Authorizer:            operatorauth.RBAC{},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return server.Handler()
}

func validPolicyCreateBody() string {
	return `{"organization_id":"123e4567-e89b-12d3-a456-426614174000","name":"frontend to orders","initial_version":{"source_spiffe_id":"spiffe://workload-trust.test/prod/frontend","destination_spiffe_id":"spiffe://workload-trust.test/prod/orders-api","action":"connect","effect":"allow","change_reason":"reviewed route"}}`
}

func TestPolicyMutationRequiresAuthenticationBeforeAuthorization(t *testing.T) {
	manager := &policyHTTPManagerStub{}
	h := newPolicyHTTPHandler(t, operatorauth.RoleOperator, manager)
	r := httptest.NewRequest(http.MethodPost, "/v1/policies", strings.NewReader(validPolicyCreateBody()))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized || manager.createCalls != 0 {
		t.Fatalf("unexpected unauthenticated result: status=%d calls=%d", w.Code, manager.createCalls)
	}
}

func TestViewerCannotCreateOrActivatePolicy(t *testing.T) {
	manager := &policyHTTPManagerStub{}
	h := newPolicyHTTPHandler(t, operatorauth.RoleViewer, manager)

	create := authenticatedJSONRequest(http.MethodPost, "/v1/policies", validPolicyCreateBody())
	createRecorder := httptest.NewRecorder()
	h.ServeHTTP(createRecorder, create)
	if createRecorder.Code != http.StatusForbidden || manager.createCalls != 0 {
		t.Fatalf("viewer create: status=%d calls=%d", createRecorder.Code, manager.createCalls)
	}

	activate := authenticatedJSONRequest(http.MethodPost, "/v1/policies/123e4567-e89b-12d3-a456-426614174010/activate", `{"expected_revision":1,"version_id":"123e4567-e89b-12d3-a456-426614174011"}`)
	activateRecorder := httptest.NewRecorder()
	h.ServeHTTP(activateRecorder, activate)
	if activateRecorder.Code != http.StatusForbidden || manager.activateCalls != 0 {
		t.Fatalf("viewer activate: status=%d calls=%d", activateRecorder.Code, manager.activateCalls)
	}
}

func TestOperatorCreatesPolicyWithoutEchoingRuleMaterial(t *testing.T) {
	manager := &policyHTTPManagerStub{createResult: policy.CreateResult{
		Policy: policy.PolicyResult{
			ID:             "123e4567-e89b-12d3-a456-426614174010",
			OrganizationID: "123e4567-e89b-12d3-a456-426614174000",
			Name:           "frontend to orders",
			Status:         "draft",
			Revision:       1,
		},
		Version: policy.VersionResult{
			ID:             "123e4567-e89b-12d3-a456-426614174011",
			PolicyID:       "123e4567-e89b-12d3-a456-426614174010",
			VersionNumber:  1,
			PolicyRevision: 1,
			Effect:         "allow",
		},
	}}
	h := newPolicyHTTPHandler(t, operatorauth.RoleOperator, manager)
	body := strings.ReplaceAll(validPolicyCreateBody(), "reviewed route", "do-not-echo-secret-context")
	r := authenticatedJSONRequest(http.MethodPost, "/v1/policies", body)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if manager.createCalls != 1 || manager.lastActor.Role != "operator" || manager.lastCorrelation == "" {
		t.Fatalf("unexpected policy attribution: actor=%#v correlation=%q", manager.lastActor, manager.lastCorrelation)
	}
	response := w.Body.String()
	for _, forbidden := range []string{"source_spiffe_id", "destination_spiffe_id", "change_reason", "do-not-echo-secret-context"} {
		if strings.Contains(response, forbidden) {
			t.Fatalf("policy response leaked %q: %s", forbidden, response)
		}
	}
	if w.Header().Get("Location") != "/v1/policies/123e4567-e89b-12d3-a456-426614174010" {
		t.Fatalf("unexpected Location: %q", w.Header().Get("Location"))
	}
}

func TestPolicyCreateRejectsUnknownNestedField(t *testing.T) {
	manager := &policyHTTPManagerStub{}
	h := newPolicyHTTPHandler(t, operatorauth.RoleOperator, manager)
	body := strings.Replace(validPolicyCreateBody(), `"change_reason":"reviewed route"`, `"change_reason":"reviewed route","unexpected":true`, 1)
	r := authenticatedJSONRequest(http.MethodPost, "/v1/policies", body)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest || manager.createCalls != 0 {
		t.Fatalf("unknown nested field: status=%d calls=%d body=%s", w.Code, manager.createCalls, w.Body.String())
	}
}

func TestPolicyAppendMapsRevisionConflict(t *testing.T) {
	manager := &policyHTTPManagerStub{appendErr: policy.ErrConflict}
	h := newPolicyHTTPHandler(t, operatorauth.RoleOperator, manager)
	body := `{"expected_revision":1,"source_spiffe_id":"spiffe://workload-trust.test/prod/frontend","destination_spiffe_id":"spiffe://workload-trust.test/prod/orders-api","action":"connect","effect":"deny","change_reason":"review changed"}`
	r := authenticatedJSONRequest(http.MethodPost, "/v1/policies/123e4567-e89b-12d3-a456-426614174010/versions", body)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusConflict || manager.appendCalls != 1 {
		t.Fatalf("revision conflict: status=%d calls=%d body=%s", w.Code, manager.appendCalls, w.Body.String())
	}
}

func TestPolicyActivationReturnsSafeSummary(t *testing.T) {
	activeID := "123e4567-e89b-12d3-a456-426614174011"
	manager := &policyHTTPManagerStub{activateResult: policy.PolicyResult{
		ID:              "123e4567-e89b-12d3-a456-426614174010",
		OrganizationID:  "123e4567-e89b-12d3-a456-426614174000",
		Name:            "frontend to orders",
		Status:          "active",
		Revision:        3,
		ActiveVersionID: &activeID,
	}}
	h := newPolicyHTTPHandler(t, operatorauth.RoleOperator, manager)
	r := authenticatedJSONRequest(http.MethodPost, "/v1/policies/123e4567-e89b-12d3-a456-426614174010/activate", `{"expected_revision":2,"version_id":"123e4567-e89b-12d3-a456-426614174011"}`)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK || manager.activateCalls != 1 {
		t.Fatalf("activation: status=%d calls=%d body=%s", w.Code, manager.activateCalls, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "spiffe://") || strings.Contains(w.Body.String(), "change_reason") {
		t.Fatalf("activation response exposed rule material: %s", w.Body.String())
	}
}

func TestPolicyInternalErrorIsGeneric(t *testing.T) {
	manager := &policyHTTPManagerStub{createErr: errors.New("postgres password=do-not-leak")}
	h := newPolicyHTTPHandler(t, operatorauth.RoleOperator, manager)
	r := authenticatedJSONRequest(http.MethodPost, "/v1/policies", validPolicyCreateBody())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	for _, forbidden := range []string{"postgres", "password", "do-not-leak"} {
		if strings.Contains(w.Body.String(), forbidden) {
			t.Fatalf("policy error leaked %q: %s", forbidden, w.Body.String())
		}
	}
}

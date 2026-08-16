package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Imad-cpp/workload-trust-platform/internal/diagnostics"
	"github.com/Imad-cpp/workload-trust-platform/internal/operatorauth"
)

type diagnosticsStub struct {
	summary diagnostics.Summary
	err     error
	calls   int
}

func (s *diagnosticsStub) SummaryByOrganization(context.Context, string) (diagnostics.Summary, error) {
	s.calls++
	return s.summary, s.err
}

func TestDiagnosticsRequiresAuthenticationFirst(t *testing.T) {
	reader := &diagnosticsStub{}
	h := newDiagnosticsHandler(t, operatorauth.RoleViewer, reader)
	r := httptest.NewRequest(http.MethodGet, "/v1/diagnostics?organization_id=123e4567-e89b-12d3-a456-426614174000", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized || reader.calls != 0 {
		t.Fatalf("unauthenticated diagnostics: status=%d calls=%d", w.Code, reader.calls)
	}
}

func TestViewerCanReadSafeDiagnostics(t *testing.T) {
	now := time.Date(2026, time.August, 16, 10, 0, 0, 0, time.UTC)
	reader := &diagnosticsStub{summary: diagnostics.Summary{
		OrganizationID:         "123e4567-e89b-12d3-a456-426614174000",
		WorkloadsTotal:         3,
		WorkloadsHealthy:       2,
		RegistrationsPending:   1,
		RegistrationsConverged: 2,
		PoliciesActive:         1,
		AuditEvents:            7,
		SecurityEvents:         1,
		LatestAuditAt:          &now,
	}}
	h := newDiagnosticsHandler(t, operatorauth.RoleViewer, reader)
	r := authenticatedRequest(http.MethodGet, "/v1/diagnostics?organization_id=123e4567-e89b-12d3-a456-426614174000")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK || reader.calls != 1 {
		t.Fatalf("viewer diagnostics: status=%d calls=%d body=%s", w.Code, reader.calls, w.Body.String())
	}
	body := w.Body.String()
	for _, required := range []string{"workloads_total", "registrations_pending", "policies_active", "audit_events"} {
		if !strings.Contains(body, required) {
			t.Fatalf("diagnostics response missing %q: %s", required, body)
		}
	}
	for _, forbidden := range []string{"selectors", "parent_spiffe_id", "source_spiffe_id", "destination_spiffe_id", "change_reason", "metadata"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("diagnostics response exposed %q: %s", forbidden, body)
		}
	}
}

func TestDiagnosticsRejectsInvalidOrganizationID(t *testing.T) {
	reader := &diagnosticsStub{}
	h := newDiagnosticsHandler(t, operatorauth.RoleViewer, reader)
	r := authenticatedRequest(http.MethodGet, "/v1/diagnostics?organization_id=bad")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest || reader.calls != 0 {
		t.Fatalf("invalid organization: status=%d calls=%d", w.Code, reader.calls)
	}
}

func TestDiagnosticsMapsMissingOrganization(t *testing.T) {
	reader := &diagnosticsStub{err: diagnostics.ErrNotFound}
	h := newDiagnosticsHandler(t, operatorauth.RoleViewer, reader)
	r := authenticatedRequest(http.MethodGet, "/v1/diagnostics?organization_id=123e4567-e89b-12d3-a456-426614174000")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), `"code":"organization_not_found"`) {
		t.Fatalf("missing organization: status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestDiagnosticsRepositoryErrorIsGeneric(t *testing.T) {
	reader := &diagnosticsStub{err: errors.New("postgres password=do-not-leak")}
	h := newDiagnosticsHandler(t, operatorauth.RoleViewer, reader)
	r := authenticatedRequest(http.MethodGet, "/v1/diagnostics?organization_id=123e4567-e89b-12d3-a456-426614174000")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	for _, forbidden := range []string{"postgres", "password", "do-not-leak"} {
		if strings.Contains(w.Body.String(), forbidden) {
			t.Fatalf("diagnostics error leaked %q: %s", forbidden, w.Body.String())
		}
	}
}

func newDiagnosticsHandler(t *testing.T, role operatorauth.Role, reader diagnostics.Reader) http.Handler {
	t.Helper()
	authenticator, err := operatorauth.NewStaticBearer(testOperatorToken, "diagnostics-test", role)
	if err != nil {
		t.Fatalf("NewStaticBearer() error = %v", err)
	}
	server, err := New(Dependencies{
		Readiness:             readinessStub{},
		Workloads:             workloadStub{},
		Diagnostics:           reader,
		RegistrationMutations: &mutationStub{},
		PolicyManager:         policyManagerStub{},
		Authenticator:         authenticator,
		Authorizer:            operatorauth.RBAC{},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return server.Handler()
}

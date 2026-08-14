package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"time"

	"github.com/Imad-cpp/workload-trust-platform/internal/operatorauth"
	"github.com/Imad-cpp/workload-trust-platform/internal/registration"
	"github.com/Imad-cpp/workload-trust-platform/internal/workload"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

type ReadinessChecker interface {
	Ping(ctx context.Context) error
}

type Dependencies struct {
	Readiness             ReadinessChecker
	Workloads             workload.Lister
	RegistrationMutations registration.Mutator
	Authenticator         operatorauth.Authenticator
	Authorizer            operatorauth.Authorizer
}

type Server struct {
	readiness             ReadinessChecker
	workloads             workload.Lister
	registrationMutations registration.Mutator
	authenticator         operatorauth.Authenticator
	authorizer            operatorauth.Authorizer
	handler               http.Handler
}

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

type workloadListResponse struct {
	Data []workload.Workload `json:"data"`
}

func New(deps Dependencies) (*Server, error) {
	if deps.Readiness == nil {
		return nil, errors.New("readiness checker is required")
	}
	if deps.Workloads == nil {
		return nil, errors.New("workload lister is required")
	}
	if deps.RegistrationMutations == nil {
		return nil, errors.New("registration mutation service is required")
	}
	if deps.Authenticator == nil {
		return nil, errors.New("operator authenticator is required")
	}
	if deps.Authorizer == nil {
		return nil, errors.New("operator authorizer is required")
	}

	s := &Server{
		readiness:             deps.Readiness,
		workloads:             deps.Workloads,
		registrationMutations: deps.RegistrationMutations,
		authenticator:         deps.Authenticator,
		authorizer:            deps.Authorizer,
	}

	apiMux := http.NewServeMux()
	apiMux.Handle("/v1/workloads", s.requirePermission(operatorauth.PermissionWorkloadsRead, getOnly(s.listWorkloads)))
	apiMux.Handle("POST /v1/registration-rules", s.requirePermission(operatorauth.PermissionRegistrationsWrite, http.HandlerFunc(s.createRegistrationRule)))
	apiMux.Handle("PATCH /v1/registration-rules/{id}", s.requirePermission(operatorauth.PermissionRegistrationsWrite, http.HandlerFunc(s.replaceRegistrationRule)))
	apiMux.HandleFunc("/", s.notFound)

	rootMux := http.NewServeMux()
	rootMux.Handle("/healthz", getOnly(s.health))
	rootMux.Handle("/readyz", getOnly(s.ready))
	rootMux.Handle("/v1/", s.requireOperator(apiMux))
	rootMux.HandleFunc("/", s.notFound)

	s.handler = securityHeaders(requestIDMiddleware(rootMux))
	return s, nil
}

func (s *Server) Handler() http.Handler {
	return s.handler
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.readiness.Ping(ctx); err != nil {
		writeError(w, r, http.StatusServiceUnavailable, "not_ready", "control plane is not ready")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) listWorkloads(w http.ResponseWriter, r *http.Request) {
	organizationID := r.URL.Query().Get("organization_id")
	if !uuidPattern.MatchString(organizationID) {
		writeError(w, r, http.StatusBadRequest, "invalid_organization_id", "organization_id must be a UUID")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	items, err := s.workloads.ListByOrganization(ctx, organizationID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "internal_error", "request could not be completed")
		return
	}
	writeJSON(w, http.StatusOK, workloadListResponse{Data: items})
}

func (s *Server) requireOperator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := s.authenticator.Authenticate(r)
		if !ok {
			w.Header().Set("WWW-Authenticate", `Bearer realm="workload-trust-platform"`)
			writeError(w, r, http.StatusUnauthorized, "unauthorized", "operator authentication required")
			return
		}
		ctx := context.WithValue(r.Context(), operatorPrincipalContextKey{}, principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) requirePermission(permission operatorauth.Permission, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := operatorPrincipalFrom(r)
		if !ok || !s.authorizer.Allowed(principal, permission) {
			writeError(w, r, http.StatusForbidden, "forbidden", "operator is not authorized for this action")
			return
		}
		next.ServeHTTP(w, r)
	})
}

type operatorPrincipalContextKey struct{}

func operatorPrincipalFrom(r *http.Request) (operatorauth.Principal, bool) {
	principal, ok := r.Context().Value(operatorPrincipalContextKey{}).(operatorauth.Principal)
	return principal, ok
}

func (s *Server) notFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, http.StatusNotFound, "not_found", "route not found")
}

func getOnly(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		next(w, r)
	})
}

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := newRequestID()
		ctx := context.WithValue(r.Context(), requestIDContextKey{}, requestID)
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type requestIDContextKey struct{}

func newRequestID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "request-id-unavailable"
	}
	return hex.EncodeToString(raw[:])
}

func requestIDFrom(r *http.Request) string {
	value, _ := r.Context().Value(requestIDContextKey{}).(string)
	return value
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeJSONStatus(w, status, errorEnvelope{Error: errorBody{Code: code, Message: message, RequestID: requestIDFrom(r)}})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	writeJSONStatus(w, status, payload)
}

func writeJSONStatus(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

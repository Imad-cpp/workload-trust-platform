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

	"github.com/Imad-cpp/workload-trust-platform/internal/workload"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

type ReadinessChecker interface {
	Ping(ctx context.Context) error
}

type Dependencies struct {
	Readiness ReadinessChecker
	Workloads workload.Lister
}

type Server struct {
	readiness ReadinessChecker
	workloads workload.Lister
	handler   http.Handler
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

	s := &Server{readiness: deps.Readiness, workloads: deps.Workloads}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", getOnly(s.health))
	mux.HandleFunc("/readyz", getOnly(s.ready))
	mux.HandleFunc("/v1/workloads", getOnly(s.listWorkloads))
	mux.HandleFunc("/", s.notFound)
	s.handler = securityHeaders(requestIDMiddleware(mux))
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

func (s *Server) notFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, http.StatusNotFound, "not_found", "route not found")
}

func getOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		next(w, r)
	}
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

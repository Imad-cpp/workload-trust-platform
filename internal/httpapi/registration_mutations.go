package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"time"

	"github.com/Imad-cpp/workload-trust-platform/internal/registration"
)

const maxMutationRequestBytes int64 = 64 * 1024

var (
	errUnsupportedJSONMediaType = errors.New("unsupported JSON media type")
	errMutationRequestTooLarge  = errors.New("mutation request is too large")
	errMalformedJSONRequest     = errors.New("malformed JSON request")
)

type createRegistrationRequest struct {
	WorkloadID         string   `json:"workload_id"`
	Selectors          []string `json:"selectors"`
	ParentSPIFFEID     string   `json:"parent_spiffe_id"`
	X509SVIDTTLSeconds int32    `json:"x509_svid_ttl_seconds"`
}

type replaceRegistrationRequest struct {
	ExpectedRevision   int64    `json:"expected_revision"`
	DesiredState       string   `json:"desired_state"`
	Selectors          []string `json:"selectors"`
	ParentSPIFFEID     string   `json:"parent_spiffe_id"`
	X509SVIDTTLSeconds int32    `json:"x509_svid_ttl_seconds"`
}

type registrationMutationResponse struct {
	Data registration.MutationResult `json:"data"`
}

func (s *Server) createRegistrationRule(w http.ResponseWriter, r *http.Request) {
	var request createRegistrationRequest
	if err := decodeStrictJSON(w, r, &request); err != nil {
		writeMutationDecodeError(w, r, err)
		return
	}
	principal, ok := operatorPrincipalFrom(r)
	if !ok {
		writeError(w, r, http.StatusForbidden, "forbidden", "operator is not authorized for this action")
		return
	}

	ctx, cancel := contextWithTimeout(r, 5*time.Second)
	defer cancel()
	result, err := s.registrationMutations.CreateDesired(ctx, registration.MutationActor{
		Type: principal.ActorType,
		ID:   principal.ActorID,
		Role: string(principal.Role),
	}, requestIDFrom(r), registration.CreateDesiredInput{
		WorkloadID:         request.WorkloadID,
		Selectors:          request.Selectors,
		ParentSPIFFEID:     request.ParentSPIFFEID,
		X509SVIDTTLSeconds: request.X509SVIDTTLSeconds,
	})
	if err != nil {
		writeRegistrationMutationError(w, r, err)
		return
	}

	w.Header().Set("Location", "/v1/registration-rules/"+result.ID)
	writeJSON(w, http.StatusCreated, registrationMutationResponse{Data: result})
}

func (s *Server) replaceRegistrationRule(w http.ResponseWriter, r *http.Request) {
	ruleID := r.PathValue("id")
	if !uuidPattern.MatchString(ruleID) {
		writeError(w, r, http.StatusBadRequest, "invalid_registration_rule_id", "registration rule ID must be a UUID")
		return
	}
	var request replaceRegistrationRequest
	if err := decodeStrictJSON(w, r, &request); err != nil {
		writeMutationDecodeError(w, r, err)
		return
	}
	principal, ok := operatorPrincipalFrom(r)
	if !ok {
		writeError(w, r, http.StatusForbidden, "forbidden", "operator is not authorized for this action")
		return
	}

	ctx, cancel := contextWithTimeout(r, 5*time.Second)
	defer cancel()
	result, err := s.registrationMutations.ReplaceDesired(ctx, registration.MutationActor{
		Type: principal.ActorType,
		ID:   principal.ActorID,
		Role: string(principal.Role),
	}, requestIDFrom(r), ruleID, registration.ReplaceDesiredInput{
		ExpectedRevision:   request.ExpectedRevision,
		DesiredState:       request.DesiredState,
		Selectors:          request.Selectors,
		ParentSPIFFEID:     request.ParentSPIFFEID,
		X509SVIDTTLSeconds: request.X509SVIDTTLSeconds,
	})
	if err != nil {
		writeRegistrationMutationError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, registrationMutationResponse{Data: result})
}

func decodeStrictJSON(w http.ResponseWriter, r *http.Request, target any) error {
	contentType := r.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != "application/json" {
		return errUnsupportedJSONMediaType
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxMutationRequestBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return errMutationRequestTooLarge
		}
		return errMalformedJSONRequest
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errMalformedJSONRequest
	}
	return nil
}

func writeMutationDecodeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, errUnsupportedJSONMediaType):
		writeError(w, r, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json")
	case errors.Is(err, errMutationRequestTooLarge):
		writeError(w, r, http.StatusRequestEntityTooLarge, "request_too_large", "request body exceeds the allowed size")
	default:
		writeError(w, r, http.StatusBadRequest, "invalid_json", "request body must be a single valid JSON object")
	}
}

func writeRegistrationMutationError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, registration.ErrInvalidInput):
		writeError(w, r, http.StatusBadRequest, "invalid_registration", "registration desired state is invalid")
	case errors.Is(err, registration.ErrNotFound):
		writeError(w, r, http.StatusNotFound, "registration_target_not_found", "registration mutation target was not found")
	case errors.Is(err, registration.ErrConflict):
		writeError(w, r, http.StatusConflict, "revision_conflict", "registration rule revision no longer matches")
	default:
		writeError(w, r, http.StatusInternalServerError, "internal_error", "request could not be completed")
	}
}

func contextWithTimeout(r *http.Request, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), timeout)
}

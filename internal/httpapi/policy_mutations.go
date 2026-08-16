package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/Imad-cpp/workload-trust-platform/internal/policy"
)

type policyRuleRequest struct {
	SourceSPIFFEID      string `json:"source_spiffe_id"`
	DestinationSPIFFEID string `json:"destination_spiffe_id"`
	Action              string `json:"action"`
	Effect              string `json:"effect"`
	ChangeReason        string `json:"change_reason"`
}

type createPolicyRequest struct {
	OrganizationID string            `json:"organization_id"`
	Name           string            `json:"name"`
	InitialVersion policyRuleRequest `json:"initial_version"`
}

type appendPolicyVersionRequest struct {
	ExpectedRevision   int64  `json:"expected_revision"`
	SourceSPIFFEID      string `json:"source_spiffe_id"`
	DestinationSPIFFEID string `json:"destination_spiffe_id"`
	Action              string `json:"action"`
	Effect              string `json:"effect"`
	ChangeReason        string `json:"change_reason"`
}

type activatePolicyRequest struct {
	ExpectedRevision int64  `json:"expected_revision"`
	VersionID        string `json:"version_id"`
}

type policyCreateResponse struct {
	Data policy.CreateResult `json:"data"`
}

type policyVersionResponse struct {
	Data policy.VersionResult `json:"data"`
}

type policyMutationResponse struct {
	Data policy.PolicyResult `json:"data"`
}

func (s *Server) createPolicy(w http.ResponseWriter, r *http.Request) {
	var request createPolicyRequest
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
	result, err := s.policyManager.Create(ctx, policy.MutationActor{
		Type: principal.ActorType,
		ID:   principal.ActorID,
		Role: string(principal.Role),
	}, requestIDFrom(r), policy.CreateInput{
		OrganizationID: request.OrganizationID,
		Name:           request.Name,
		InitialVersion: policy.RuleInput{
			SourceSPIFFEID:      request.InitialVersion.SourceSPIFFEID,
			DestinationSPIFFEID: request.InitialVersion.DestinationSPIFFEID,
			Action:              request.InitialVersion.Action,
			Effect:              request.InitialVersion.Effect,
			ChangeReason:        request.InitialVersion.ChangeReason,
		},
	})
	if err != nil {
		writePolicyMutationError(w, r, err)
		return
	}

	w.Header().Set("Location", "/v1/policies/"+result.Policy.ID)
	writeJSON(w, http.StatusCreated, policyCreateResponse{Data: result})
}

func (s *Server) appendPolicyVersion(w http.ResponseWriter, r *http.Request) {
	policyID := r.PathValue("id")
	if !uuidPattern.MatchString(policyID) {
		writeError(w, r, http.StatusBadRequest, "invalid_policy_id", "policy ID must be a UUID")
		return
	}
	var request appendPolicyVersionRequest
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
	result, err := s.policyManager.AppendVersion(ctx, policy.MutationActor{
		Type: principal.ActorType,
		ID:   principal.ActorID,
		Role: string(principal.Role),
	}, requestIDFrom(r), policyID, policy.AppendVersionInput{
		ExpectedRevision: request.ExpectedRevision,
		Version: policy.RuleInput{
			SourceSPIFFEID:      request.SourceSPIFFEID,
			DestinationSPIFFEID: request.DestinationSPIFFEID,
			Action:              request.Action,
			Effect:              request.Effect,
			ChangeReason:        request.ChangeReason,
		},
	})
	if err != nil {
		writePolicyMutationError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, policyVersionResponse{Data: result})
}

func (s *Server) activatePolicy(w http.ResponseWriter, r *http.Request) {
	policyID := r.PathValue("id")
	if !uuidPattern.MatchString(policyID) {
		writeError(w, r, http.StatusBadRequest, "invalid_policy_id", "policy ID must be a UUID")
		return
	}
	var request activatePolicyRequest
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
	result, err := s.policyManager.Activate(ctx, policy.MutationActor{
		Type: principal.ActorType,
		ID:   principal.ActorID,
		Role: string(principal.Role),
	}, requestIDFrom(r), policyID, policy.ActivateInput{
		ExpectedRevision: request.ExpectedRevision,
		VersionID:        request.VersionID,
	})
	if err != nil {
		writePolicyMutationError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, policyMutationResponse{Data: result})
}

func writePolicyMutationError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, policy.ErrForbidden):
		writeError(w, r, http.StatusForbidden, "forbidden", "operator is not authorized for this action")
	case errors.Is(err, policy.ErrInvalidInput):
		writeError(w, r, http.StatusBadRequest, "invalid_policy", "policy state is invalid")
	case errors.Is(err, policy.ErrNotFound):
		writeError(w, r, http.StatusNotFound, "policy_target_not_found", "policy mutation target was not found")
	case errors.Is(err, policy.ErrConflict):
		writeError(w, r, http.StatusConflict, "revision_conflict", "policy revision no longer matches")
	default:
		writeError(w, r, http.StatusInternalServerError, "internal_error", "request could not be completed")
	}
}

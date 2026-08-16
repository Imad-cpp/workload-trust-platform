package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/Imad-cpp/workload-trust-platform/internal/diagnostics"
)

type diagnosticsResponse struct {
	Data diagnostics.Summary `json:"data"`
}

func (s *Server) organizationDiagnostics(w http.ResponseWriter, r *http.Request) {
	organizationID := r.URL.Query().Get("organization_id")
	if !uuidPattern.MatchString(organizationID) {
		writeError(w, r, http.StatusBadRequest, "invalid_organization_id", "organization_id must be a UUID")
		return
	}

	ctx, cancel := contextWithTimeout(r, 3*time.Second)
	defer cancel()
	summary, err := s.diagnostics.SummaryByOrganization(ctx, organizationID)
	if errors.Is(err, diagnostics.ErrNotFound) {
		writeError(w, r, http.StatusNotFound, "organization_not_found", "organization was not found")
		return
	}
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "internal_error", "request could not be completed")
		return
	}
	writeJSON(w, http.StatusOK, diagnosticsResponse{Data: summary})
}

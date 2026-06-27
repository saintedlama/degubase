package jobs

import (
	"net/http"

	httplib "github.com/saintedlama/degubase/internal/infrastructure/http"
	"github.com/saintedlama/degubase/internal/models"
)

// Handler serves the job run endpoints.
type Handler struct {
	Store Store
}

// ListJobRuns returns paginated job runs, newest first.
func (h *Handler) ListJobRuns(w http.ResponseWriter, r *http.Request) {
	paging := httplib.ParsePaging(r)

	runs, err := h.Store.ListJobRuns(r.Context(), paging.PageSize, paging.Offset())
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}

	total, err := h.Store.CountJobRuns(r.Context())
	if err != nil {
		httplib.InternalErr(w, err)
		return
	}

	result := models.NewPagedResult(runs, total, paging.Page, paging.PageSize)
	httplib.Respond(w, http.StatusOK, result)
}

package jobs

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mirra-ai/mirra/backend/internal/store"
	"github.com/mirra-ai/mirra/backend/pkg/config"
	appmiddleware "github.com/mirra-ai/mirra/backend/pkg/middleware"
	"github.com/mirra-ai/mirra/backend/pkg/response"
)

type Handler struct {
	cfg  *config.Config
	jobs store.JobStore
}

func NewHandler(cfg *config.Config, jobs store.JobStore) *Handler {
	return &Handler{cfg: cfg, jobs: jobs}
}

// List returns all jobs for the authenticated user.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := appmiddleware.GetUserID(r)

	jobs, err := h.jobs.ListByOwner(r.Context(), userID)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch jobs")
		return
	}

	if jobs == nil {
		jobs = []*store.Job{}
	}

	response.JSON(w, http.StatusOK, jobs)
}

// Get returns a single job by ID.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID := appmiddleware.GetUserID(r)
	id := chi.URLParam(r, "id")

	job, err := h.jobs.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			response.Err(w, http.StatusNotFound, "JOB_NOT_FOUND", "Job not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch job")
		return
	}

	if job.OwnerID != userID {
		response.Err(w, http.StatusNotFound, "JOB_NOT_FOUND", "Job not found")
		return
	}

	response.JSON(w, http.StatusOK, job)
}

package jobs

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mirra-ai/mirra/backend/pkg/config"
	"github.com/mirra-ai/mirra/backend/pkg/response"
)

// Handler handles job status endpoints.
type Handler struct {
	cfg *config.Config
}

func NewHandler(cfg *config.Config) *Handler {
	return &Handler{cfg: cfg}
}

// Job represents an async processing job.
type Job struct {
	ID          string `json:"id"`
	PersonaID   string `json:"personaId"`
	Status      string `json:"status"` // queued | running | done | failed
	CurrentStep string `json:"currentStep"` // ingest | extract | distill | score
	ErrorLog    string `json:"errorLog,omitempty"`
}

// List returns all jobs for the authenticated user.
// TODO: query from database
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, []Job{})
}

// Get returns a single job by ID.
// TODO: query from database, check ownership
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	response.Err(w, http.StatusNotFound, "JOB_NOT_FOUND", "Job "+id+" not found")
}

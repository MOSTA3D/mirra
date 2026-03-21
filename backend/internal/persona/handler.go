package persona

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/mirra-ai/mirra/backend/pkg/config"
	appmiddleware "github.com/mirra-ai/mirra/backend/pkg/middleware"
	"github.com/mirra-ai/mirra/backend/pkg/response"
)

// Handler handles persona CRUD endpoints.
type Handler struct {
	cfg *config.Config
}

func NewHandler(cfg *config.Config) *Handler {
	return &Handler{cfg: cfg}
}

// Persona represents a distilled persona entity.
type Persona struct {
	ID          string            `json:"id"`
	OwnerID     string            `json:"ownerId"`
	Name        string            `json:"name"`
	Slug        string            `json:"slug"`
	Visibility  string            `json:"visibility"` // private | public
	Status      string            `json:"status"`     // draft | processing | ready
	Disclaimer  string            `json:"disclaimer"`
	Confidence  map[string]float64 `json:"confidence"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

// Source represents a data source attached to a persona.
type Source struct {
	ID        string    `json:"id"`
	PersonaID string    `json:"personaId"`
	Type      string    `json:"type"` // url | pdf | text
	Content   string    `json:"content"`
	Status    string    `json:"status"` // pending | processed | failed
	CreatedAt time.Time `json:"createdAt"`
}

type createPersonaRequest struct {
	Name       string `json:"name"`
	Visibility string `json:"visibility"`
}

type addSourceRequest struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// List returns all personas owned by the authenticated user.
// TODO: query from database
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := appmiddleware.GetUserID(r)
	_ = userID
	response.JSON(w, http.StatusOK, []Persona{})
}

// Create creates a new persona.
// TODO: persist to database
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID := appmiddleware.GetUserID(r)

	var req createPersonaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	if req.Name == "" {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "Name is required")
		return
	}

	if req.Visibility == "" {
		req.Visibility = "private"
	}

	now := time.Now().UTC()
	persona := &Persona{
		ID:         uuid.NewString(),
		OwnerID:    userID,
		Name:       req.Name,
		Slug:       slugify(req.Name),
		Visibility: req.Visibility,
		Status:     "draft",
		Disclaimer: "This is an AI-generated approximation. Not affiliated with or endorsed by the real person.",
		Confidence: map[string]float64{},
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	response.JSON(w, http.StatusCreated, persona)
}

// Get returns a single persona by ID.
// TODO: query from database, check ownership
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	response.Err(w, http.StatusNotFound, "PERSONA_NOT_FOUND", "Persona "+id+" not found")
}

// Update updates a persona's metadata.
// TODO: persist to database
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	response.Err(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Coming soon")
}

// Delete removes a persona and all its sources.
// TODO: cascade delete in database
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	response.Err(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Coming soon")
}

// AddSource attaches a data source to a persona and queues it for processing.
// TODO: persist source, enqueue processing job
func (h *Handler) AddSource(w http.ResponseWriter, r *http.Request) {
	personaID := chi.URLParam(r, "id")

	var req addSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	validTypes := map[string]bool{"url": true, "pdf": true, "text": true}
	if !validTypes[req.Type] {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "Type must be one of: url, pdf, text")
		return
	}

	if req.Content == "" {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "Content is required")
		return
	}

	source := &Source{
		ID:        uuid.NewString(),
		PersonaID: personaID,
		Type:      req.Type,
		Content:   req.Content,
		Status:    "pending",
		CreatedAt: time.Now().UTC(),
	}

	response.JSON(w, http.StatusCreated, source)
}

// Export generates a persona export in the requested format.
// TODO: trigger export pipeline
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	response.Err(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Export coming soon")
}

func slugify(name string) string {
	slug := ""
	for _, c := range name {
		switch {
		case c >= 'a' && c <= 'z':
			slug += string(c)
		case c >= 'A' && c <= 'Z':
			slug += string(c + 32)
		case c == ' ' || c == '-':
			slug += "-"
		}
	}
	return slug
}

package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/mirra-ai/mirra/backend/internal/store"
	"github.com/mirra-ai/mirra/backend/pkg/config"
	"github.com/mirra-ai/mirra/backend/pkg/response"
	"golang.org/x/crypto/bcrypt"
)

// Handler handles authentication endpoints.
type Handler struct {
	cfg   *config.Config
	users store.UserStore
}

func NewHandler(cfg *config.Config, users store.UserStore) *Handler {
	return &Handler{cfg: cfg, users: users}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type tokenResponse struct {
	AccessToken string `json:"accessToken"`
	ExpiresIn   int    `json:"expiresIn"`
	UserID      string `json:"userId"`
}

// Register creates a new user account.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "Email and password are required")
		return
	}

	if len(req.Password) < 8 {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "Password must be at least 8 characters")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to process password")
		return
	}

	now := time.Now().UTC()
	user := &store.User{
		ID:           uuid.NewString(),
		Email:        req.Email,
		PasswordHash: string(hash),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := h.users.Create(r.Context(), user); err != nil {
		if errors.Is(err, store.ErrAlreadyExists) {
			response.Err(w, http.StatusConflict, "EMAIL_TAKEN", "An account with this email already exists")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create account")
		return
	}

	tokens, err := h.generateTokens(user.ID)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to generate tokens")
		return
	}

	response.JSON(w, http.StatusCreated, tokens)
}

// Login authenticates an existing user.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "Email and password are required")
		return
	}

	user, err := h.users.GetByEmail(r.Context(), req.Email)
	if err != nil {
		// Return same error for not found and wrong password — don't leak which one
		response.Err(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		response.Err(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
		return
	}

	tokens, err := h.generateTokens(user.ID)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to generate tokens")
		return
	}

	response.JSON(w, http.StatusOK, tokens)
}

// Refresh — placeholder for refresh token flow
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	response.Err(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Refresh endpoint coming soon")
}

func (h *Handler) generateTokens(userID string) (*tokenResponse, error) {
	expiresIn := 3600 * 24 // 24 hours

	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiresIn) * time.Second)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(h.cfg.JWTSecret))
	if err != nil {
		return nil, err
	}

	return &tokenResponse{
		AccessToken: signed,
		ExpiresIn:   expiresIn,
		UserID:      userID,
	}, nil
}

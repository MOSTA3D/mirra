package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/mirra-ai/mirra/backend/internal/store"
	"github.com/mirra-ai/mirra/backend/internal/verification"
	"github.com/mirra-ai/mirra/backend/pkg/config"
	"github.com/mirra-ai/mirra/backend/pkg/response"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	cfg          *config.Config
	users        store.UserStore
	verification *verification.Store
	reset        *verification.ResetStore
}

func NewHandler(cfg *config.Config, users store.UserStore, vs *verification.Store, rs *verification.ResetStore) *Handler {
	return &Handler{cfg: cfg, users: users, verification: vs, reset: rs}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type sendCodeRequest struct {
	Email string `json:"email"`
}

type verifyCodeRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type tokenResponse struct {
	AccessToken string `json:"accessToken"`
	ExpiresIn   int    `json:"expiresIn"`
	UserID      string `json:"userId"`
}

// SendVerificationCode sends a code to the given email.
func (h *Handler) SendVerificationCode(w http.ResponseWriter, r *http.Request) {
	var req sendCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	if req.Email == "" {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "Email is required")
		return
	}

	if _, err := h.verification.Issue(r.Context(), req.Email); err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to send verification code")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"message": "Verification code sent to " + req.Email,
	})
}

// VerifyCode validates a code submitted by the user.
func (h *Handler) VerifyCode(w http.ResponseWriter, r *http.Request) {
	var req verifyCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	if req.Email == "" || req.Code == "" {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "Email and code are required")
		return
	}

	if err := h.verification.Verify(r.Context(), req.Email, req.Code); err != nil {
		var ve *verification.VerificationError
		if errors.As(err, &ve) {
			response.Err(w, http.StatusBadRequest, ve.Code, ve.Message)
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Verification failed")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Email verified successfully"})
}

// Register creates a new user account (requires prior email verification).
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

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	response.Err(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Refresh endpoint coming soon")
}

// ForgotPassword sends a password reset code to the given email.
// Always returns 200 even if email not found — prevents user enumeration.
func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "Email is required")
		return
	}

	// Check if user exists (silently — don't leak)
	if _, err := h.users.GetByEmail(r.Context(), req.Email); err == nil {
		// Only send code if account exists
		h.reset.Issue(r.Context(), req.Email)
	}

	// Always return success to prevent user enumeration
	response.JSON(w, http.StatusOK, map[string]string{
		"message": "If an account exists for this email, a reset code has been sent.",
	})
}

// VerifyResetCode validates the code without changing the password yet.
func (h *Handler) VerifyResetCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	if req.Email == "" || req.Code == "" {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "Email and code are required")
		return
	}

	if err := h.reset.Verify(r.Context(), req.Email, req.Code); err != nil {
		var ve *verification.VerificationError
		if errors.As(err, &ve) {
			response.Err(w, http.StatusBadRequest, ve.Code, ve.Message)
			return
		}
		response.Err(w, http.StatusBadRequest, "INVALID_CODE", "Invalid or expired code")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Code verified"})
}

// ResetPassword sets a new password after verifying the reset code.
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Code     string `json:"code"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}
	if req.Email == "" || req.Code == "" || req.Password == "" {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "Email, code and password are required")
		return
	}
	if len(req.Password) < 8 {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "Password must be at least 8 characters")
		return
	}

	// Verify code
	if err := h.reset.Verify(r.Context(), req.Email, req.Code); err != nil {
		var ve *verification.VerificationError
		if errors.As(err, &ve) {
			response.Err(w, http.StatusBadRequest, ve.Code, ve.Message)
			return
		}
		response.Err(w, http.StatusBadRequest, "INVALID_CODE", "Invalid or expired code")
		return
	}

	// Get user
	user, err := h.users.GetByEmail(r.Context(), req.Email)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_CODE", "Invalid or expired code")
		return
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to process password")
		return
	}

	user.PasswordHash = string(hash)
	user.UpdatedAt = time.Now().UTC()

	if err := h.users.Update(r.Context(), user); err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update password")
		return
	}

	// Consume the reset code so it can't be reused
	h.reset.Consume(req.Email)

	response.JSON(w, http.StatusOK, map[string]string{"message": "Password updated successfully"})
}

func (h *Handler) generateTokens(userID string) (*tokenResponse, error) {
	expiresIn := 3600 * 24
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
	return &tokenResponse{AccessToken: signed, ExpiresIn: expiresIn, UserID: userID}, nil
}

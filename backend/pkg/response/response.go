package response

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Envelope is the standard API response wrapper.
// Every response — success or error — uses this shape.
type Envelope struct {
	Data  any    `json:"data"`
	Error *Error `json:"error"`
	Meta  Meta   `json:"meta"`
}

// Error represents a structured API error.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Meta contains request metadata attached to every response.
type Meta struct {
	RequestID string `json:"requestId"`
	Timestamp string `json:"timestamp"`
}

// JSON writes a JSON response with the given status code and data.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Envelope{
		Data:  data,
		Error: nil,
		Meta:  newMeta(),
	})
}

// Err writes a structured error response.
func Err(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Envelope{
		Data: nil,
		Error: &Error{
			Code:    code,
			Message: message,
		},
		Meta: newMeta(),
	})
}

func newMeta() Meta {
	return Meta{
		RequestID: uuid.NewString(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

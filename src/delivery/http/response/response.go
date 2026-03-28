package response

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gothi/vouchrs/src/internal/domain/apperror"
	"github.com/gothi/vouchrs/src/pkg/pagination"
)

type envelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *apiError   `json:"error,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

type apiError struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// Response is the standard JSON envelope returned by every endpoint.
// Used only as a swag type reference — the actual serialisation uses envelope above.
type Response struct {
	Success bool      `json:"success" example:"true"`
	Data    any       `json:"data,omitempty"`
	Error   *APIError `json:"error,omitempty"`
	Meta    any       `json:"meta,omitempty"`
}

// APIError is the error detail embedded in a failed Response.
type APIError struct {
	Message string `json:"message" example:"listing not found"`
	Code    string `json:"code,omitempty" example:"NOT_FOUND"`
}

// JSON writes a JSON response with the given status code and data.
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{Success: true, Data: data})
}

// Paginated writes a paginated JSON response.
func Paginated(w http.ResponseWriter, status int, data interface{}, meta pagination.Meta) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{Success: true, Data: data, Meta: meta})
}

// Error maps a domain error to an HTTP status code and writes the response.
func Error(w http.ResponseWriter, err error) {
	status, code := statusFromError(err)
	if status == http.StatusInternalServerError {
		slog.Error("unhandled error", "error", err)
	}
	msg := err.Error()

	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		msg = appErr.Message
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{
		Success: false,
		Error:   &apiError{Message: msg, Code: code},
	})
}

func statusFromError(err error) (int, string) {
	switch {
	case errors.Is(err, apperror.ErrNotFound):
		return http.StatusNotFound, "NOT_FOUND"
	case errors.Is(err, apperror.ErrUnauthorized):
		return http.StatusUnauthorized, "UNAUTHORIZED"
	case errors.Is(err, apperror.ErrForbidden):
		return http.StatusForbidden, "FORBIDDEN"
	case errors.Is(err, apperror.ErrConflict), errors.Is(err, apperror.ErrDuplicateCard):
		return http.StatusConflict, "CONFLICT"
	case errors.Is(err, apperror.ErrBadRequest):
		return http.StatusBadRequest, "BAD_REQUEST"
	case errors.Is(err, apperror.ErrListingLocked):
		return http.StatusConflict, "LISTING_LOCKED"
	case errors.Is(err, apperror.ErrListingNotLive):
		return http.StatusConflict, "LISTING_NOT_AVAILABLE"
	case errors.Is(err, apperror.ErrVerificationFailed):
		return http.StatusUnprocessableEntity, "VERIFICATION_FAILED"
	case errors.Is(err, apperror.ErrCardTampered):
		return http.StatusUnprocessableEntity, "CARD_TAMPERED"
	case errors.Is(err, apperror.ErrOTPInvalid):
		return http.StatusUnauthorized, "OTP_INVALID"
	case errors.Is(err, apperror.ErrOTPTooManyAttempts):
		return http.StatusTooManyRequests, "OTP_RATE_LIMITED"
	case errors.Is(err, apperror.ErrListingLimitReached):
		return http.StatusUnprocessableEntity, "LISTING_LIMIT_REACHED"
	case errors.Is(err, apperror.ErrUnprocessable):
		return http.StatusUnprocessableEntity, "UNPROCESSABLE"
	default:
		return http.StatusInternalServerError, "INTERNAL_ERROR"
	}
}

package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gothi/vouchrs/src/internal/domain/apperror"
	"github.com/gothi/vouchrs/src/delivery/http/response"
)

// Recover catches panics and returns a 500 response.
func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic recovered",
						"panic", rec,
						"stack", string(debug.Stack()),
					)
					response.Error(w, apperror.ErrInternal)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

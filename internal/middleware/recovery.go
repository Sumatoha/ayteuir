package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/ayteuir/backend/internal/pkg/logger"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error().
					Interface("error", err).
					Str("stack", string(debug.Stack())).
					Str("path", r.URL.Path).
					Str("method", r.Method).
					Msg("panic recovered")

				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":"internal server error"}`))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

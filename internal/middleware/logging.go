package middleware

import (
	"net/http"
	"time"

	"github.com/ayteuir/backend/internal/pkg/logger"
	"github.com/go-chi/chi/v5/middleware"
)

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		defer func() {
			logger.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("remote_addr", r.RemoteAddr).
				Int("status", ww.Status()).
				Int("bytes", ww.BytesWritten()).
				Dur("latency", time.Since(start)).
				Str("request_id", middleware.GetReqID(r.Context())).
				Msg("request completed")
		}()

		next.ServeHTTP(ww, r)
	})
}

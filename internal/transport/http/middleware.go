package httptransport

import (
	"log/slog"
	"net/http"
	"time"

	// "github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func LoggingMiddleware(log *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()

			defer func ()  {
				duration := time.Since(start)

				attrs := []any{
					"method", r.Method,
					"path", r.URL.Path,
					"status", ww.Status(),
					"size", ww.BytesWritten(),
					"duration_ms", duration.Milliseconds(),
					"request_id", middleware.GetReqID(r.Context()),
				}
				log.Info("http_request", attrs...)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}
package middleware

import (
	"net/http"
	"rtcs/internal/metrics"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a custom response writer to capture the status code
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Get the route template
		route := mux.CurrentRoute(r)
		var endpoint string
		if route != nil {
			if pathTemplate, err := route.GetPathTemplate(); err == nil {
				endpoint = pathTemplate
			} else {
				endpoint = r.URL.Path
			}
		} else {
			endpoint = r.URL.Path
		}

		// Call the next handler
		next.ServeHTTP(rw, r)

		// Record metrics
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(rw.statusCode)

		metrics.HttpRequestsTotal.WithLabelValues(r.Method, endpoint, status).Inc()
		metrics.HttpRequestDuration.WithLabelValues(r.Method, endpoint).Observe(duration)
	})
}

// responseWriter is a custom response writer that captures the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

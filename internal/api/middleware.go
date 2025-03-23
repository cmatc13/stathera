// internal/api/middleware.go
package api

import (
	"net/http"
	"time"

	"github.com/cmatc13/stathera/pkg/logging"
	"github.com/cmatc13/stathera/pkg/metrics"
	"github.com/go-chi/chi/v5/middleware"
)

// MetricsMiddleware creates middleware that records request metrics
func MetricsMiddleware(metricsCollector *metrics.Metrics, serviceName string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a response writer wrapper to capture the status code
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// Track in-flight requests
			metricsCollector.RequestInFlight.WithLabelValues(serviceName).Inc()
			defer metricsCollector.RequestInFlight.WithLabelValues(serviceName).Dec()

			// Call the next handler
			next.ServeHTTP(ww, r)

			// Record metrics after the request is processed
			duration := time.Since(start)
			status := ww.Status()
			if status == 0 {
				status = http.StatusOK // Default to 200 if status not explicitly set
			}

			// Record request metrics
			metricsCollector.RecordRequest(
				serviceName,
				r.Method,
				r.URL.Path,
				status,
				duration,
			)
		})
	}
}

// LoggingMiddleware creates middleware that logs requests using structured logging
func LoggingMiddleware(logger *logging.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a response writer wrapper to capture the status code
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// Get request ID from context if available
			requestID := middleware.GetReqID(r.Context())

			// Log request start
			logger.Info("Request started",
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
				"request_id", requestID,
			)

			// Call the next handler
			next.ServeHTTP(ww, r)

			// Log request completion
			duration := time.Since(start)
			status := ww.Status()
			if status == 0 {
				status = http.StatusOK // Default to 200 if status not explicitly set
			}

			// Determine log level based on status code
			if status >= 500 {
				logger.Error("Request completed with server error",
					"method", r.Method,
					"path", r.URL.Path,
					"status", status,
					"duration_ms", duration.Milliseconds(),
					"request_id", requestID,
				)
			} else if status >= 400 {
				logger.Warn("Request completed with client error",
					"method", r.Method,
					"path", r.URL.Path,
					"status", status,
					"duration_ms", duration.Milliseconds(),
					"request_id", requestID,
				)
			} else {
				logger.Info("Request completed successfully",
					"method", r.Method,
					"path", r.URL.Path,
					"status", status,
					"duration_ms", duration.Milliseconds(),
					"request_id", requestID,
				)
			}
		})
	}
}

// RecovererWithMetrics is a middleware that recovers from panics, logs the panic,
// and records it as a metric
func RecovererWithMetrics(logger *logging.Logger, metricsCollector *metrics.Metrics, serviceName string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					// Log the panic
					logger.Error("Panic recovered",
						"error", rvr,
						"method", r.Method,
						"path", r.URL.Path,
						"request_id", middleware.GetReqID(r.Context()),
					)

					// Record the panic as a metric
					metricsCollector.RecordError(serviceName, "panic", "PANIC")

					// Return a 500 Internal Server Error
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

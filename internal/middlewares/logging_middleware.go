package middlewares

import (
	"log"
	"net/http"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a custom response writer to capture the status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			status:         200,
		}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Log the request details
		log.Printf(
			"Method: %s | Path: %s | Status: %d | Latency: %v | IP: %s | User-Agent: %s",
			r.Method,
			r.URL.Path,
			wrapped.status,
			time.Since(start),
			r.RemoteAddr,
			r.UserAgent(),
		)
	})
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.status = code
		rw.ResponseWriter.WriteHeader(code)
		rw.wroteHeader = true
	}
}

func (rw *responseWriter) Write(buf []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(buf)
}

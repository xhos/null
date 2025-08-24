package middleware

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/charmbracelet/log"
)

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type contextKey string

const (
	UserContextKey  contextKey = "user"
	InternalAuthKey contextKey = "internal_auth"
	UserIDKey       contextKey = "user_id"
)

// responseWriter wraps http.ResponseWriter to capture response data
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:           &bytes.Buffer{},
	}
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	rw.body.Write(data)
	return rw.ResponseWriter.Write(data)
}

func getUserFromContext(r *http.Request) (*User, bool) {
	user, ok := r.Context().Value(UserContextKey).(*User)
	return user, ok
}

func isInternalRequest(r *http.Request) bool {
	internal, ok := r.Context().Value(InternalAuthKey).(bool)
	return ok && internal
}

func Logging(logger *log.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			
			// Capture request body for debug logging
			var requestBody []byte
			if r.Body != nil {
				requestBody, _ = io.ReadAll(r.Body)
				r.Body = io.NopCloser(bytes.NewBuffer(requestBody))
			}

			// Wrap response writer to capture response
			rw := newResponseWriter(w)

			logFields := []interface{}{
				"method", r.Method,
				"path", r.URL.Path,
				"user_agent", r.Header.Get("User-Agent"),
				"content_type", r.Header.Get("Content-Type"),
			}

			if user, ok := getUserFromContext(r); ok {
				logFields = append(logFields, "user_id", user.ID, "user_email", user.Email)
			} else if isInternalRequest(r) {
				logFields = append(logFields, "auth_type", "internal")
			}

			logger.Debug("incoming request", append(logFields, "request_body", string(requestBody))...)

			// Process request
			next.ServeHTTP(rw, r)

			// Log response
			duration := time.Since(start)
			responseFields := append(logFields,
				"status_code", rw.statusCode,
				"duration_ms", duration.Milliseconds(),
			)

			responseBody := rw.body.String()
			
			// Always log errors (4xx/5xx status codes)
			if rw.statusCode >= 400 {
				logger.Error("API error response", append(responseFields, "response_body", responseBody)...)
			} else {
				// Debug log successful responses
				logger.Debug("API response", append(responseFields, "response_body", responseBody)...)
			}
		})
	}
}

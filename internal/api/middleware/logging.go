package middleware

import (
	"net/http"

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
)

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

			logger.Debug("incoming request", logFields...)
			next.ServeHTTP(w, r)
		})
	}
}

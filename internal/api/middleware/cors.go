package middleware

import (
	"net/http"

	"github.com/rs/cors"
)

func CORS() Middleware {
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, // TODO: harden
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"}, // TODO: harden
		ExposedHeaders:   []string{"Connect-Protocol-Version"},
		AllowCredentials: true,
	})

	return func(next http.Handler) http.Handler {
		return c.Handler(next)
	}
}

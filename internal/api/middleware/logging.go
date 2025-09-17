package middleware

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"github.com/charmbracelet/log"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
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

// ConnectLoggingInterceptor creates a Connect unary interceptor for structured logging
func ConnectLoggingInterceptor(logger *log.Logger) connect.UnaryInterceptorFunc {
	marshaler := protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: false,
		Indent:          "",
	}

	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			start := time.Now()
			procedure := req.Spec().Procedure

			// Extract HTTP info from headers
			userAgent := req.Header().Get("User-Agent")
			contentType := req.Header().Get("Content-Type")

			// Extract user info from context if available
			var logFields []any
			if user, ok := ctx.Value(UserContextKey).(*User); ok {
				logFields = append(logFields, "user_id", user.ID, "user_email", user.Email)
			} else if internal, ok := ctx.Value(InternalAuthKey).(bool); ok && internal {
				logFields = append(logFields, "auth_type", "internal")
			}

			// Log incoming request with comprehensive info
			requestFields := append([]any{
				"timestamp", start.Format(time.RFC3339),
				"procedure", procedure,
				"user_agent", userAgent,
				"content_type", contentType,
			}, logFields...)

			if reqMsg, ok := req.Any().(proto.Message); ok {
				if jsonBytes, err := marshaler.Marshal(reqMsg); err == nil {
					requestFields = append(requestFields, "request", string(jsonBytes))
				}
			}

			logger.Debug("connect request", requestFields...)

			// Call the actual handler
			resp, err := next(ctx, req)
			duration := time.Since(start)

			// Log response or error with comprehensive info
			responseFields := append([]any{
				"timestamp", time.Now().Format(time.RFC3339),
				"procedure", procedure,
				"user_agent", userAgent,
				"content_type", contentType,
				"duration_ms", duration.Milliseconds(),
			}, logFields...)

			if err != nil {
				// Log error responses with Connect error details
				responseFields = append(responseFields, "error", err.Error())
				if connectErr := new(connect.Error); errors.As(err, &connectErr) {
					responseFields = append(responseFields,
						"code", connectErr.Code().String(),
					)
				}
				logger.Error("connect error", responseFields...)
				return nil, err
			}

			// Log successful response with status info
			responseFields = append(responseFields, "status", "success")
			if respMsg, ok := resp.Any().(proto.Message); ok {
				if jsonBytes, err := marshaler.Marshal(respMsg); err == nil {
					responseFields = append(responseFields, "response", string(jsonBytes))
				}
			}

			logger.Debug("connect response", responseFields...)
			return resp, nil
		}
	})
}

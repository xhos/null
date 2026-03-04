package middleware

import (
	"context"

	"connectrpc.com/connect"
	"github.com/charmbracelet/log"
)

type UserEnsurer interface {
	EnsureExists(ctx context.Context, id, email, name string) error
}

func EnsureUserInterceptor(ensurer UserEnsurer, logger *log.Logger) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			user, ok := ctx.Value(UserContextKey).(*User)
			if !ok || user == nil {
				return next(ctx, req)
			}

			if err := ensurer.EnsureExists(ctx, user.ID, user.Email, user.Name); err != nil {
				logger.Error("failed to ensure user exists", "user_id", user.ID, "error", err)
				return nil, connect.NewError(connect.CodeInternal, err)
			}

			return next(ctx, req)
		}
	}
}

package api

import (
	"ariand/internal/api/middleware"
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"
)

func getUserID(ctx context.Context) (uuid.UUID, error) {
	userID, ok := ctx.Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated or user_id not found"))
	}
	return userID, nil
}

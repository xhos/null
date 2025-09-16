package service

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrPermission    = errors.New("permission denied")
	ErrConflict      = errors.New("conflict")
	ErrValidation    = errors.New("validation failed")
	ErrUnimplemented = errors.New("unimplemented")
)

func wrapErr(op string, err error) error {
	if err == nil {
		return nil
	}

	knownErrors := []error{
		ErrNotFound,
		ErrPermission,
		ErrConflict,
		ErrValidation,
		ErrUnimplemented,
	}

	for _, knownErr := range knownErrors {
		if errors.Is(err, knownErr) {
			return fmt.Errorf("%s: %w", op, knownErr)
		}
	}

	return fmt.Errorf("%s: %w", op, err)
}

package handler

import "github.com/google/uuid"

type ApiClient interface {
	Call(pairID uuid.UUID) error
}

type ApiError struct {
	err error
}

func (e *ApiError) Error() string {
	return e.err.Error()
}
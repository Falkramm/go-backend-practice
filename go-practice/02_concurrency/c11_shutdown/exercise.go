package c11

import (
	"context"
	"errors"
)

var (
	ErrClosed         = errors.New("service is closed")
	errNotImplemented = errors.New("not implemented")
)

type Service struct {
}

func NewService(workers, queueSize int, handle func(context.Context, int) error) (*Service, error) {
	return nil, errNotImplemented
}

func (*Service) Submit(ctx context.Context, id int) error {
	return errNotImplemented
}

func (*Service) Shutdown(ctx context.Context) error {
	return errNotImplemented
}

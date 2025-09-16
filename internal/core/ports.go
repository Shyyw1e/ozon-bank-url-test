package core

import "context"

type Store interface {
	GetByOriginal(ctx context.Context, original string) (code string, found bool, err error)
	GetByCode(ctx context.Context, code string) (original string, found bool, err error)
    Create(ctx context.Context, code, original string) error
}


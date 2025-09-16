package core

import "errors"

var (
	ErrInvalidURL = errors.New("invalid url")
	ErrNotFound   = errors.New("not found")
	ErrConflict   = errors.New("conflict") // исчерпали попытки генерации
	
	ErrDupCode    = errors.New("duplicate code")
    ErrDupOrigin  = errors.New("duplicate original")
)

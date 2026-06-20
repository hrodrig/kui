package store

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrDuplicateEmail     = errors.New("email already exists")
	ErrLastAdmin          = errors.New("cannot remove the last admin")
)

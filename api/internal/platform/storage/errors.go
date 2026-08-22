package storage

import "errors"

var (
	ErrObjectNotFound       = errors.New("object not found")
	ErrObjectAlreadyExists  = errors.New("object already exists")
	ErrInvalidObjectKey     = errors.New("invalid object key")
	ErrUnsupportedOperation = errors.New("unsupported operation")
	ErrAuthentication       = errors.New("authentication failed")
	ErrPermissionDenied     = errors.New("permission denied")
	ErrInvalidConfig        = errors.New("invalid provider config")
)

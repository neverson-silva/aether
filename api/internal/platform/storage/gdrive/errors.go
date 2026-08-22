package gdrive

import (
	"errors"
	"fmt"

	"google.golang.org/api/googleapi"

	"aether/internal/platform/storage"
)

func mapError(err error) error {
	var gerr *googleapi.Error
	if !errors.As(err, &gerr) {
		return err
	}
	switch gerr.Code {
	case 401:
		return fmt.Errorf("%w: %v", storage.ErrAuthentication, err)
	case 403:
		return fmt.Errorf("%w: %v", storage.ErrPermissionDenied, err)
	case 404:
		return fmt.Errorf("%w: %v", storage.ErrObjectNotFound, err)
	case 409:
		return fmt.Errorf("%w: %v", storage.ErrObjectAlreadyExists, err)
	}
	return err
}

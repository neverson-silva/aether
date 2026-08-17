package storage

import "strings"

func NormalizeKey(key string) (string, error) {
	if key == "" {
		return "", ErrInvalidObjectKey
	}
	if strings.ContainsRune(key, '\x00') {
		return "", ErrInvalidObjectKey
	}
	key = strings.TrimPrefix(key, "/")
	if key == "" {
		return "", ErrInvalidObjectKey
	}
	for _, part := range strings.Split(key, "/") {
		if part == "" || part == "." || part == ".." {
			return "", ErrInvalidObjectKey
		}
	}
	return key, nil
}

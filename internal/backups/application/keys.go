package application

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"aether/internal/backups/domain"
)

func StorageKey(prefix string, dbID uuid.UUID, engine string, backupID uuid.UUID, ts time.Time, format string) (string, error) {
	clean := sanitizePrefix(prefix)
	if clean == "" {
		return "", domain.ErrValidation
	}
	stamp := ts.UTC().Format("20060102T150405Z")
	return fmt.Sprintf("%s/%s/%s/backup-%s-%s.%s", clean, engine, dbID, stamp, backupID, format), nil
}

func SafeToken(s string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "database"
	}
	return b.String()
}

func sanitizePrefix(prefix string) string {
	p := strings.TrimSpace(prefix)
	p = strings.Trim(p, "/")
	p = strings.ReplaceAll(p, "\\", "/")
	if p == "" || p == "." {
		return ""
	}
	for _, part := range strings.Split(p, "/") {
		if part == "" || part == "." || part == ".." {
			return ""
		}
	}
	return p
}
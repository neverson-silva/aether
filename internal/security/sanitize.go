package security

import (
	"bufio"
	"io"
	"regexp"
	"sync"
)

// SanitizeLog mascarada valores sensíveis em qualquer string de log.
// Patterns cobertos: Authorization/Bearer/JWT, senhas, tokens, keys,
// connection strings, cookies, session IDs, etc.
func SanitizeLog(s string) string {
	if s == "" {
		return s
	}
	for _, re := range logSanitizers {
		s = re.ReplaceAllString(s, "[REDACTED]")
	}
	return s
}

// SanitizingWriter envolve um io.Writer e redige segredos linha a linha.
// Use com log.SetOutput(security.NewSanitizingWriter(os.Stderr)).
type SanitizingWriter struct {
	mu    sync.Mutex
	inner io.Writer
	w     *bufio.Writer
}

func NewSanitizingWriter(inner io.Writer) *SanitizingWriter {
	return &SanitizingWriter{inner: inner, w: bufio.NewWriter(inner)}
}

func (s *SanitizingWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	clean := SanitizeLog(string(p))
	if _, err := s.w.WriteString(clean); err != nil {
		return len(p), err
	}
	return len(p), s.w.Flush()
}

var logSanitizers = []*regexp.Regexp{
	// Authorization / Bearer / Basic / Token headers
	regexp.MustCompile(`(?i)(authorization\s*[:=]\s*)(?:bearer|basic|token|apikey)?\s*[A-Za-z0-9._\-+/=]{8,}`),
	regexp.MustCompile(`(?i)(\bbearer\s+)[A-Za-z0-9._\-+/=]{8,}`),
	// JWT (eyJ... segments)
	regexp.MustCompile(`(?i)eyJ[A-Za-z0-9_\-]+\.[A-Za-z0-9_\-]+\.[A-Za-z0-9_\-]+`),
	// passwords / secrets / tokens / keys (key=value)
	regexp.MustCompile(`(?i)(password|passwd|senha|secret|token|apikey|api[_-]?key|access[_-]?key|secret[_-]?key|private[_ ]?key|openai[_-]?key|smtp[_-]?pass|client[_-]?secret|refresh[_-]?token|access[_-]?token|webhook[_-]?secret|auth[_-]?token)\s*[:=]\s*["']?[^\s,;&"']{4,}`),
	// connection strings (postgres:///redis:///mysql:///mongodb:// com credenciais)
	regexp.MustCompile(`(?i)([a-z]+)://[^/@\s]+:[^/@\s]+@`),
	// cookies / session IDs
	regexp.MustCompile(`(?i)(cookie|session[_-]?id|sid)\s*[:=]\s*["']?[A-Za-z0-9+/=_-]{8,}`),
	// genérico: chaves longas base64 (>=32 chars) em contexto de credencial
	regexp.MustCompile(`(?i)\b(a[ks][k-s]|[a-z0-9_]{3,})["']?\s*[:=]\s*["']?[A-Za-z0-9+/=_-]{32,}`),
}

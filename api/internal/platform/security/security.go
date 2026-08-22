package security

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidKey     = errors.New("invalid key")
	ErrInvalidToken   = errors.New("invalid token")
	ErrPasswordTooOld = errors.New("password hash format not supported")
)

type Secrets struct {
	kek    []byte // master key (32 bytes) — nunca no banco
	crypto *EnvelopeCrypto
}

// LoadSecrets carrega (ou gera) a master key e prepara a camada de criptografia.
// A master key vive apenas no disco com permissão 0600; nunca no banco.
func LoadSecrets(keysDir string) (*Secrets, error) {
	kekPath := keysDir + "/master.key"
	kek, err := os.ReadFile(kekPath)
	if err != nil {
		kek, err = GenerateMasterKey()
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(keysDir, 0o700); err != nil {
			return nil, err
		}
		if err := os.WriteFile(kekPath, kek, 0o600); err != nil {
			return nil, err
		}
	}
	if len(kek) != encMasterKeyLen {
		return nil, ErrInvalidKey
	}
	crypto, err := NewEnvelopeCrypto(kek)
	if err != nil {
		return nil, err
	}
	return &Secrets{kek: kek, crypto: crypto}, nil
}

func (s *Secrets) Encrypt(data []byte) ([]byte, error) {
	return s.crypto.Encrypt(data)
}

func (s *Secrets) Decrypt(data []byte) ([]byte, error) {
	return s.crypto.Decrypt(data)
}

func (s *Secrets) EncryptString(v string) (string, error) {
	return s.crypto.EncryptString(v)
}

func (s *Secrets) DecryptString(v string) (string, error) {
	return s.crypto.DecryptString(v)
}

func (s *Secrets) EncryptFile(data []byte, path string) error {
	enc, err := s.Encrypt(data)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, enc, 0o600)
}

func (s *Secrets) DecryptFile(path string) ([]byte, error) {
	enc, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return s.Decrypt(enc)
}

type PasswordHasher struct{}

func (PasswordHasher) Hash(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)
	enc := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=19$m=65536,t=3,p=4$%s$%s",
		enc.EncodeToString(salt), enc.EncodeToString(hash)), nil
}

func (PasswordHasher) Verify(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, ErrPasswordTooOld
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}
	got := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

type TokenManager struct {
	secret []byte
}

func NewTokenManager(secrets *Secrets) (*TokenManager, error) {
	if len(secrets.kek) < 32 {
		return nil, ErrInvalidKey
	}
	h := sha256.New()
	h.Write(secrets.kek)
	h.Write([]byte("aether:token:v1"))
	enc := h.Sum(nil)
	sum := sha256.Sum256(enc)
	return &TokenManager{secret: sum[:]}, nil
}

type Claims struct {
	Subject    string
	OrgID      string
	Role       string
	GlobalRole string
	Exp        int64
}

func (t *TokenManager) Sign(c Claims, ttl time.Duration) (string, error) {
	c.Exp = time.Now().Add(ttl).Unix()
	payload, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	enc := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmacSHA256(t.secret, []byte(enc))
	return enc + "." + base64.RawURLEncoding.EncodeToString(mac), nil
}

func (t *TokenManager) Verify(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, ErrInvalidToken
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, ErrInvalidToken
	}
	want := base64.RawURLEncoding.EncodeToString(hmacSHA256(t.secret, []byte(parts[0])))
	if subtle.ConstantTimeCompare([]byte(want), []byte(parts[1])) != 1 {
		return nil, ErrInvalidToken
	}
	var c Claims
	if err := json.Unmarshal(payload, &c); err != nil {
		return nil, ErrInvalidToken
	}
	if c.Exp < time.Now().Unix() {
		return nil, ErrInvalidToken
	}
	return &c, nil
}

func hmacSHA256(secret, data []byte) []byte {
	h := sha256.New()
	h.Write(secret)
	h.Write(data)
	return h.Sum(nil)
}

func HashAPIKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return base64.RawStdEncoding.EncodeToString(sum[:])
}

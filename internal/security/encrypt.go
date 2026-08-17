package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

// Envelope de criptografia em repouso — formato versionado com metadados.
//
// Layout binário (base64 na persistência):
//
//	byte 0      : version (1)
//	byte 1      : algorithm (1 = AES-256-GCM)
//	byte 2..5   : key_id (4 bytes = prefixo do SHA-256 da master key)
//	byte 6..21  : salt (16 bytes) — deriva a DEK por registro via HKDF
//	byte 22..33 : nonce (12 bytes)
//	byte 34..   : ciphertext + authentication tag (16 bytes)
//
// Cada registro possui sua própria DEK derivada (HKDF(master, salt)), permitindo
// rotação de chaves e múltiplas versões sem reescrever os dados existentes.
const (
	encVersionV1    byte = 1
	encAlgAESGCM    byte = 1
	encKeyIDLen     int  = 4
	encSaltLen      int  = 16
	encNonceLen     int  = 12
	encHeaderLen    int  = 34
	encMasterKeyLen int  = 32
)

var (
	ErrCipherInvalid = errors.New("corrupted ciphertext or invalid master key")
	ErrCipherVersion = errors.New("unsupported ciphertext version")
	ErrCipherKeyID   = errors.New("ciphertext uses another master key (rotation required)")
)

// EnvelopeCrypto é a única camada de criptografia simétrica da aplicação.
type EnvelopeCrypto struct {
	masterKey []byte
	keyID     []byte
}

// NewEnvelopeCrypto recebe a master key (32 bytes) e prepara o encryptor.
func NewEnvelopeCrypto(masterKey []byte) (*EnvelopeCrypto, error) {
	if len(masterKey) != encMasterKeyLen {
		return nil, ErrInvalidKey
	}
	sum := sha256.Sum256(masterKey)
	return &EnvelopeCrypto{masterKey: masterKey, keyID: sum[:encKeyIDLen]}, nil
}

// GenerateMasterKey cria uma master key criptograficamente segura (32 bytes).
func GenerateMasterKey() ([]byte, error) {
	k := make([]byte, encMasterKeyLen)
	if _, err := rand.Read(k); err != nil {
		return nil, err
	}
	return k, nil
}

func (e *EnvelopeCrypto) deriveDEK(salt []byte) ([]byte, error) {
	dek := make([]byte, 32)
	rd := hkdf.New(sha256.New, e.masterKey, salt, []byte("aether:v1:record"))
	if _, err := io.ReadFull(rd, dek); err != nil {
		return nil, err
	}
	return dek, nil
}

// Encrypt protege os dados em um envelope versionado.
func (e *EnvelopeCrypto) Encrypt(plaintext []byte) ([]byte, error) {
	salt := make([]byte, encSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	dek, err := e.deriveDEK(salt)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	ct := gcm.Seal(nil, nonce, plaintext, nil)

	out := make([]byte, 0, encHeaderLen+len(ct))
	out = append(out, encVersionV1, encAlgAESGCM)
	out = append(out, e.keyID...)
	out = append(out, salt...)
	out = append(out, nonce...)
	out = append(out, ct...)
	return out, nil
}

// Decrypt reverte o envelope. Suporta múltiplas versões (futuro) e detecta
// ciphertext de outra master key (rotação).
func (e *EnvelopeCrypto) Decrypt(data []byte) ([]byte, error) {
	if len(data) < encHeaderLen {
		return nil, ErrCipherInvalid
	}
	version := data[0]
	switch version {
	case encVersionV1:
		// ok
	default:
		return nil, ErrCipherVersion
	}
	alg := data[1]
	if alg != encAlgAESGCM {
		return nil, ErrCipherVersion
	}
	keyID := data[2:6]
	if !bytesEqual(keyID, e.keyID) {
		return nil, ErrCipherKeyID
	}
	salt := data[6:22]
	nonce := data[22:34]
	ct := data[34:]

	dek, err := e.deriveDEK(salt)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, ErrCipherInvalid
	}
	return pt, nil
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := range a {
		v |= a[i] ^ b[i]
	}
	return v == 0
}

// Encode/Decode base64 helpers para persistência em colunas TEXT.

// EncodeCiphertext serializa em base64 para armazenamento.
func EncodeCiphertext(enc []byte) string {
	return base64.StdEncoding.EncodeToString(enc)
}

// DecodeCiphertext reverte o base64.
func DecodeCiphertext(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

func (e *EnvelopeCrypto) EncryptString(v string) (string, error) {
	enc, err := e.Encrypt([]byte(v))
	if err != nil {
		return "", err
	}
	return EncodeCiphertext(enc), nil
}

func (e *EnvelopeCrypto) DecryptString(v string) (string, error) {
	enc, err := DecodeCiphertext(v)
	if err != nil {
		return "", ErrCipherInvalid
	}
	pt, err := e.Decrypt(enc)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

// Describe retorna metadados do envelope (auditoria/diagnóstico).
func (e *EnvelopeCrypto) Describe(data []byte) (map[string]any, error) {
	if len(data) < encHeaderLen {
		return nil, ErrCipherInvalid
	}
	return map[string]any{
		"version":   int(data[0]),
		"algorithm": "AES-256-GCM",
		"key_id":    fmt.Sprintf("%x", data[2:6]),
	}, nil
}

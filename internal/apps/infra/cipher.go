package infra

import (
	"aether/internal/apps/domain"
	"aether/internal/security"
)

type secretCipher struct {
	crypto *security.EnvelopeCrypto
}

func NewSecretCipher(masterKey []byte) (domain.SecretCipher, error) {
	crypto, err := security.NewEnvelopeCrypto(masterKey)
	if err != nil {
		return nil, err
	}
	return secretCipher{crypto: crypto}, nil
}

func (c secretCipher) Encrypt(plain string) (string, error) {
	return c.crypto.EncryptString(plain)
}

func (c secretCipher) Decrypt(ciphertext string) (string, error) {
	return c.crypto.DecryptString(ciphertext)
}

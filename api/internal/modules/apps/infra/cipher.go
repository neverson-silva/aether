package infra

import (
	"aether/internal/modules/apps/domain"
	"aether/internal/platform/security"
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
	plain, err := c.crypto.DecryptString(ciphertext)
	if err == nil {
		return plain, nil
	}
	if !security.IsEnvelopeCiphertext(ciphertext) {
		return ciphertext, nil
	}
	return "", err
}

package infra

import "aether/internal/platform/security"

type passwordCipher struct {
	crypto *security.EnvelopeCrypto
}

func NewPasswordCipher(masterKey []byte) (passwordCipher, error) {
	crypto, err := security.NewEnvelopeCrypto(masterKey)
	if err != nil {
		return passwordCipher{}, err
	}
	return passwordCipher{crypto: crypto}, nil
}

func (c passwordCipher) Encrypt(plain string) (string, error) {
	return c.crypto.EncryptString(plain)
}

func (c passwordCipher) Decrypt(ciphertext string) (string, error) {
	return c.crypto.DecryptString(ciphertext)
}

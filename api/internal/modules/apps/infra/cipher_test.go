package infra

import "testing"

func TestSecretCipherDecryptsEnvelopeAndLegacyPlaintext(t *testing.T) {
	cipher, err := NewSecretCipher([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("create cipher: %v", err)
	}

	encrypted, err := cipher.Encrypt("supersecret")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	decrypted, err := cipher.Decrypt(encrypted)
	if err != nil || decrypted != "supersecret" {
		t.Fatalf("decrypt envelope: %v %q", err, decrypted)
	}

	legacy, err := cipher.Decrypt("legacy-secret")
	if err != nil || legacy != "legacy-secret" {
		t.Fatalf("decrypt legacy value: %v %q", err, legacy)
	}
}

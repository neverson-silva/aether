package security

import (
	"encoding/json"
	"strings"
	"testing"
)

func testCrypto(t *testing.T) *EnvelopeCrypto {
	mk, err := GenerateMasterKey()
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewEnvelopeCrypto(mk)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestEnvelopeRoundTrip(t *testing.T) {
	c := testCrypto(t)
	plain := []byte("hello world")
	enc, err := c.Encrypt(plain)
	if err != nil {
		t.Fatal(err)
	}
	if len(enc) <= encHeaderLen {
		t.Fatalf("envelope curto: %d", len(enc))
	}
	// formato: version + alg + keyID + salt + nonce
	if enc[0] != 1 || enc[1] != 1 {
		t.Fatalf("header: %d %d", enc[0], enc[1])
	}
	dec, err := c.Decrypt(enc)
	if err != nil {
		t.Fatal(err)
	}
	if string(dec) != string(plain) {
		t.Fatalf("roundtrip: %q", dec)
	}
}

func TestEnvelopeDeterministicNonce(t *testing.T) {
	c := testCrypto(t)
	a, _ := c.Encrypt([]byte("same"))
	b, _ := c.Encrypt([]byte("same"))
	// nonces diferentes → ciphertexts diferentes (GCM random nonce)
	if strings.EqualFold(string(a[encHeaderLen:]), string(b[encHeaderLen:])) {
		t.Fatal("ciphertexts iguais — nonce reutilizado?")
	}
}

func TestEnvelopeUnicodeAndJSON(t *testing.T) {
	c := testCrypto(t)
	payload := map[string]any{"url": "postgres://user:senha@host:5432/db", "emoji": "🚀", "chave": "ç ã õ"}
	raw, _ := json.Marshal(payload)
	enc, _ := c.Encrypt(raw)
	dec, err := c.Decrypt(enc)
	if err != nil {
		t.Fatal(err)
	}
	var back map[string]any
	if err := json.Unmarshal(dec, &back); err != nil {
		t.Fatal(err)
	}
	if back["emoji"] != "🚀" || back["chave"] != "ç ã õ" {
		t.Fatalf("unicode perdido: %v", back)
	}
}

func TestEnvelopeEmptyAndLong(t *testing.T) {
	c := testCrypto(t)
	for _, s := range []string{"", "a", strings.Repeat("x", 1024*1024)} {
		enc, err := c.Encrypt([]byte(s))
		if err != nil {
			t.Fatal(err)
		}
		dec, err := c.Decrypt(enc)
		if err != nil {
			t.Fatal(err)
		}
		if string(dec) != s {
			t.Fatalf("len %d mismatch", len(s))
		}
	}
}

func TestEnvelopeTamperDetection(t *testing.T) {
	c := testCrypto(t)
	enc, _ := c.Encrypt([]byte("segredo"))
	// corrompe o último byte (tag)
	enc[len(enc)-1] ^= 0xff
	if _, err := c.Decrypt(enc); err == nil {
		t.Fatal("ciphertext corrompido deveria falhar")
	}
	// corrompe um byte do ciphertext
	enc2, _ := c.Encrypt([]byte("dados"))
	enc2[encHeaderLen+5] ^= 0x01
	if _, err := c.Decrypt(enc2); err == nil {
		t.Fatal("tamper no ciphertext deveria falhar")
	}
}

func TestEnvelopeWrongKey(t *testing.T) {
	c1 := testCrypto(t)
	c2 := testCrypto(t) // outra master key
	enc, _ := c1.Encrypt([]byte("x"))
	if _, err := c2.Decrypt(enc); err == nil {
		t.Fatal("outra key deveria falhar")
	}
}

func TestEnvelopeTruncated(t *testing.T) {
	c := testCrypto(t)
	enc, _ := c.Encrypt([]byte("dados"))
	for _, n := range []int{0, 5, encHeaderLen - 1, encHeaderLen} {
		if _, err := c.Decrypt(enc[:n]); err == nil {
			t.Fatalf("truncado len %d deveria falhar", n)
		}
	}
}

func TestEnvelopeStringAPI(t *testing.T) {
	c := testCrypto(t)
	for _, s := range []string{"segredo", "com espaço", "unicode🚀", ""} {
		enc, err := c.EncryptString(s)
		if err != nil {
			t.Fatal(err)
		}
		if enc == s && s != "" {
			t.Fatal("string não deve ficar em claro")
		}
		dec, err := c.DecryptString(enc)
		if err != nil {
			t.Fatal(err)
		}
		if dec != s {
			t.Fatalf("mismatch: %q", dec)
		}
	}
}

func TestMasterKeyLength(t *testing.T) {
	mk, _ := GenerateMasterKey()
	if len(mk) != 32 {
		t.Fatalf("master key len %d", len(mk))
	}
}

func TestSecretsLoadPersists(t *testing.T) {
	dir := t.TempDir()
	s1, err := LoadSecrets(dir)
	if err != nil {
		t.Fatal(err)
	}
	enc, _ := s1.EncryptString("valor")
	// segunda carga reutiliza a MESMA master key (não gera nova)
	s2, err := LoadSecrets(dir)
	if err != nil {
		t.Fatal(err)
	}
	dec, err := s2.DecryptString(enc)
	if err != nil || dec != "valor" {
		t.Fatalf("persistência: %q %v", dec, err)
	}
}

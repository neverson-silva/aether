package security

import (
	"bytes"
	"testing"
)

func TestEnvelopeRotation(t *testing.T) {
	mk1, _ := GenerateMasterKey()
	mk2, _ := GenerateMasterKey()
	c1, _ := NewEnvelopeCrypto(mk1)
	c2, _ := NewEnvelopeCrypto(mk2)
	enc, _ := c1.Encrypt([]byte("segredo"))
	if _, err := c2.Decrypt(enc); err != ErrCipherKeyID {
		t.Fatalf("esperado ErrCipherKeyID, got %v", err)
	}
	// key_id distintos entre masters
	d1, _ := c1.Describe(enc)
	if d1["key_id"] == "" {
		t.Fatal("key_id vazio")
	}
	// ciphertext com outra key não se mistura
	if bytes.Equal(c1.keyID, c2.keyID) {
		t.Fatal("key_id deveriam diferir")
	}
}

func TestEnvelopeDescribeMetadata(t *testing.T) {
	c := testCrypto(t)
	enc, _ := c.Encrypt([]byte("x"))
	meta, err := c.Describe(enc)
	if err != nil {
		t.Fatal(err)
	}
	if meta["version"] != 1 || meta["algorithm"] != "AES-256-GCM" {
		t.Fatalf("meta: %v", meta)
	}
	if _, ok := meta["key_id"].(string); !ok || meta["key_id"].(string) == "" {
		t.Fatalf("key_id ausente: %v", meta)
	}
}

func TestInvalidMasterKey(t *testing.T) {
	if _, err := NewEnvelopeCrypto(make([]byte, 16)); err == nil {
		t.Fatal("master key de 16 bytes deveria falhar")
	}
	if _, err := NewEnvelopeCrypto(nil); err == nil {
		t.Fatal("nil deveria falhar")
	}
}

func TestSanitizeLog(t *testing.T) {
	cases := []struct{ in, mustNotContain string }{
		{`Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxIn0.signature`, "eyJhbGci"},
		{`authorization: Basic dXNlcjpwYXNz`, "dXNlcjpwYXNz"},
		{`password=supersecret123 x`, "supersecret123"},
		{`DB_PASSWORD = 'x9fQ$!1aa' rest`, "x9fQ$!1aa"},
		{`token: ghp_1234567890abcdefghij`, "ghp_1234567890abcdefghij"},
		{`postgres://admin:senha123@db:5432/app`, "senha123"},
		{`X-API-Key: sk-openai-ABCDEF123456`, "sk-openai-ABCDEF123456"},
		{`x-auth-token="lkj98987"`, "lkj98987"},
	}
	for _, c := range cases {
		out := SanitizeLog(c.in)
		if containsSubstring(out, c.mustNotContain) {
			t.Fatalf("vazou secret em log\n in: %q\nout: %q", c.in, out)
		}
	}
	// texto normal não deve ser destruído
	normal := "deploy app 12 finished in 3s"
	if SanitizeLog(normal) != normal {
		t.Fatalf("texto normal alterado: %q", SanitizeLog(normal))
	}
}

func TestSanitizingWriter(t *testing.T) {
	var buf bytes.Buffer
	w := NewSanitizingWriter(&buf)
	w.Write([]byte("senha=abc123xyz fica bem"))
	if containsSubstring(buf.String(), "abc123xyz") {
		t.Fatalf("writer vazou: %q", buf.String())
	}
	if !containsSubstring(buf.String(), "fica bem") {
		t.Fatalf("writer perdeu texto: %q", buf.String())
	}
}

func containsSubstring(s, sub string) bool {
	return bytes.Contains([]byte(s), []byte(sub))
}

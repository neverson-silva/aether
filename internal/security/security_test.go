package security

import (
	"testing"
	"time"
)

func TestSecretsRoundTrip(t *testing.T) {
	s, err := LoadSecrets(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	plain := "valor-secreto-com-ç-accent"
	enc, err := s.EncryptString(plain)
	if err != nil {
		t.Fatal(err)
	}
	dec, err := s.DecryptString(enc)
	if err != nil {
		t.Fatal(err)
	}
	if dec != plain {
		t.Fatalf("roundtrip divergiu: %q != %q", dec, plain)
	}
	keyPath := t.TempDir() + "/keys"
	if err := s.EncryptFile([]byte("arquivo"), keyPath+"/x"); err != nil {
		t.Fatal(err)
	}
	data, err := s.DecryptFile(keyPath + "/x")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "arquivo" {
		t.Fatal("arquivo divergiu")
	}
}

func TestSecretsWrongKeyFails(t *testing.T) {
	s1, _ := LoadSecrets(t.TempDir())
	enc, _ := s1.EncryptString("x")
	s2, _ := LoadSecrets(t.TempDir())
	if _, err := s2.DecryptString(enc); err == nil {
		t.Fatal("deveria falhar com chave diferente")
	}
}

func TestPasswordHashAndVerify(t *testing.T) {
	var h PasswordHasher
	hash, err := h.Hash("senha-forte-123")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "senha-forte-123" {
		t.Fatal("hash não pode ser o texto")
	}
	ok, err := h.Verify("senha-forte-123", hash)
	if err != nil || !ok {
		t.Fatal("verificação deveria passar")
	}
	ok, _ = h.Verify("senha-errada", hash)
	if ok {
		t.Fatal("verificação deveria falhar")
	}
}

func TestTokenSignVerify(t *testing.T) {
	s, _ := LoadSecrets(t.TempDir())
	tm, err := NewTokenManager(s)
	if err != nil {
		t.Fatal(err)
	}
	token, err := tm.Sign(Claims{Subject: "u1", OrgID: "o1", Role: "owner"}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := tm.Verify(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "u1" || claims.OrgID != "o1" || claims.Role != "owner" {
		t.Fatalf("claims divergiram: %+v", claims)
	}
	if _, err := tm.Verify(token + "x"); err == nil {
		t.Fatal("token adulterado deveria falhar")
	}
	expired, _ := tm.Sign(Claims{Subject: "u1"}, -time.Minute)
	if _, err := tm.Verify(expired); err == nil {
		t.Fatal("token expirado deveria falhar")
	}
}

func TestAPIKeyHash(t *testing.T) {
	key := "ak_abc123"
	if HashAPIKey(key) == key {
		t.Fatal("hash não pode ser o texto")
	}
	if HashAPIKey(key) != HashAPIKey(key) {
		t.Fatal("hash deve ser determinístico")
	}
	if HashAPIKey(key) == HashAPIKey(key+"x") {
		t.Fatal("hashes diferentes para chaves diferentes")
	}
}

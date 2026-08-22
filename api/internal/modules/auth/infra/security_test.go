package infra

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"aether/internal/modules/auth/domain"
)

func TestSignerRoundTrip(t *testing.T) {
	s := NewSigner("unit-test-secret-0123456789abcdef0123456789abcdef")
	ctx := context.Background()
	sub := uuid.New()
	org := uuid.New()
	token, err := s.Sign(ctx, sub, org, domain.RoleAdmin, "admin", time.Hour)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	got, err := s.Verify(ctx, token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if got.Subject != sub || got.OrgID != org || got.Role != domain.RoleAdmin || got.Global != "admin" {
		t.Fatalf("payload divergente: %+v", got)
	}
	if !got.Expires.After(time.Now()) {
		t.Fatalf("expiração no passado")
	}
}

func TestSignerRejectsTampered(t *testing.T) {
	s := NewSigner("unit-test-secret-0123456789abcdef0123456789abcdef")
	ctx := context.Background()
	token, err := s.Sign(ctx, uuid.New(), uuid.New(), domain.RoleOwner, "", time.Hour)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if _, err := s.Verify(ctx, token[:len(token)-2]+"aa"); err == nil {
		t.Fatalf("assinatura adulterada aceita")
	}
}

func TestSignerRejectsExpired(t *testing.T) {
	s := NewSigner("unit-test-secret-0123456789abcdef0123456789abcdef")
	ctx := context.Background()
	token, err := s.Sign(ctx, uuid.New(), uuid.New(), domain.RoleMember, "", -time.Minute)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if _, err := s.Verify(ctx, token); err != domain.ErrUnauthorized {
		t.Fatalf("token expirado não rejeitado: %v", err)
	}
}

func TestHasher(t *testing.T) {
	h := NewHasher()
	ctx := context.Background()
	hash, err := h.Hash(ctx, "senha-secreta-123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !h.Verify(ctx, "senha-secreta-123", hash) {
		t.Fatalf("senha correta rejeitada")
	}
	if h.Verify(ctx, "senha-errada", hash) {
		t.Fatalf("senha errada aceita")
	}
}

func TestVerifyArgon2ID(t *testing.T) {
	h := &Hasher{}
	hash := "$argon2id$v=19$m=65536,t=3,p=4$xFfgiFgYqkBTWZyYWfskKg$COjaBYq67VHLEUXvbHa1G8iLkewScRlV7PDT/yhO5sY"
	if !h.Verify(context.Background(), "Never@1212", hash) {
		t.Fatalf("argon2id deveria verificar")
	}
	if h.Verify(context.Background(), "wrong", hash) {
		t.Fatalf("senha errada não deveria passar")
	}
}

func TestVerifyBcrypt(t *testing.T) {
	h := &Hasher{}
	hash, err := h.Hash(context.Background(), "senha12345")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !h.Verify(context.Background(), "senha12345", hash) {
		t.Fatalf("bcrypt deveria verificar")
	}
}

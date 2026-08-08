package cert

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"aether/internal/config"
	"aether/internal/db"
	"aether/internal/security"
)

type fakeACME struct {
	mu           sync.Mutex
	challengeSrv string
	authzs       map[string]string
	orders       map[string]string
	certDER      []byte
	nonce        int
	base         string
	accountThumb string
	accountJWK   map[string]any
}

func (f *fakeACME) baseURL() string { return f.base }

func (f *fakeACME) nonceHeader() string {
	f.nonce++
	return "nonce-" + string(rune('0'+f.nonce%10))
}

func (f *fakeACME) writeNonce(w http.ResponseWriter) {
	w.Header().Set("Replay-Nonce", f.nonceHeader())
}

func (f *fakeACME) jwkThumbprint(protectedB64 string) string {
	protectedRaw, err := base64.RawURLEncoding.DecodeString(protectedB64)
	if err != nil {
		return ""
	}
	var prot struct {
		JWK map[string]any `json:"jwk"`
	}
	if err := json.Unmarshal(protectedRaw, &prot); err != nil {
		return ""
	}
	crv, _ := prot.JWK["crv"].(string)
	kty, _ := prot.JWK["kty"].(string)
	x, _ := prot.JWK["x"].(string)
	y, _ := prot.JWK["y"].(string)
	canonical := `{"crv":"` + crv + `","kty":"` + kty + `","x":"` + x + `","y":"` + y + `"}`
	sum := sha256.Sum256([]byte(canonical))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (f *fakeACME) handler(t *testing.T) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/dir", func(w http.ResponseWriter, r *http.Request) {
		f.writeNonce(w)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"newNonce":   f.baseURL() + "/new-nonce",
			"newAccount": f.baseURL() + "/new-account",
			"newOrder":   f.baseURL() + "/new-order",
			"revokeCert": f.baseURL() + "/revoke-cert",
		})
	})
	mux.HandleFunc("/new-nonce", func(w http.ResponseWriter, r *http.Request) {
		f.writeNonce(w)
		w.WriteHeader(200)
	})
	mux.HandleFunc("/new-account", func(w http.ResponseWriter, r *http.Request) {
		f.writeNonce(w)
		var req struct {
			Protected string `json:"protected"`
			Payload   string `json:"payload"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		payloadRaw, _ := base64.RawURLEncoding.DecodeString(req.Payload)
		onlyExisting := strings.Contains(string(payloadRaw), "onlyReturnExisting")
		var jwk map[string]any
		if !onlyExisting {
			f.mu.Lock()
			f.accountThumb = f.jwkThumbprint(req.Protected)
			f.accountJWK = f.parseJWK(req.Protected)
			f.mu.Unlock()
			jwk = f.accountJWK
			w.Header().Set("Location", f.baseURL()+"/acct/1")
			w.WriteHeader(201)
		} else {
			f.mu.Lock()
			jwk = f.accountJWK
			f.mu.Unlock()
			w.WriteHeader(200)
		}
		if jwk == nil {
			jwk = map[string]any{}
		}
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "valid",
			"contact": []string{},
			"orders":  f.baseURL() + "/list-orders",
			"key":     jwk,
		})
	})
	mux.HandleFunc("/acct/1", func(w http.ResponseWriter, r *http.Request) {
		f.writeNonce(w)
		f.mu.Lock()
		jwk := f.accountJWK
		f.mu.Unlock()
		if jwk == nil {
			jwk = map[string]any{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "valid",
			"contact": []string{},
			"orders":  f.baseURL() + "/list-orders",
			"key":     jwk,
		})
	})
	mux.HandleFunc("/new-order", func(w http.ResponseWriter, r *http.Request) {
		f.writeNonce(w)
		f.mu.Lock()
		f.orders["order-1"] = "pending"
		f.mu.Unlock()
		w.Header().Set("Location", f.baseURL()+"/order/order-1")
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(map[string]any{
			"status":         "pending",
			"authorizations": []string{f.baseURL() + "/authz/authz-1"},
			"finalize":       f.baseURL() + "/finalize/order-1",
			"identifiers":    []map[string]string{{"type": "dns", "value": "spike.example.com"}},
			"expires":        time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
		})
	})
	mux.HandleFunc("/authz/authz-1", func(w http.ResponseWriter, r *http.Request) {
		f.writeNonce(w)
		w.Header().Set("Content-Type", "application/json")
		status := "pending"
		f.mu.Lock()
		if s, ok := f.authzs["authz-1"]; ok {
			status = s
		}
		f.mu.Unlock()
		json.NewEncoder(w).Encode(map[string]any{
			"identifier": map[string]string{"type": "dns", "value": "spike.example.com"},
			"status":     status,
			"challenges": []map[string]string{{
				"type": "http-01", "url": f.baseURL() + "/challenge/token-abc", "token": "token-abc", "status": status,
			}},
			"expires": time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
		})
	})
	mux.HandleFunc("/challenge/token-abc", func(w http.ResponseWriter, r *http.Request) {
		f.writeNonce(w)
		f.mu.Lock()
		keyAuth := "token-abc." + f.accountThumb
		f.mu.Unlock()
		fmt.Fprintf(os.Stderr, "DEBUG challenge thumb=%q\n", f.accountThumb)
		ok := f.validate(keyAuth)
		fmt.Fprintf(os.Stderr, "DEBUG validate=%v\n", ok)
		f.mu.Lock()
		if ok {
			f.authzs["authz-1"] = "valid"
			f.orders["order-1"] = "ready"
		}
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		status := "pending"
		if ok {
			status = "valid"
		}
		json.NewEncoder(w).Encode(map[string]string{
			"type": "http-01", "url": f.baseURL() + "/challenge/token-abc", "token": "token-abc", "status": status,
		})
	})
	mux.HandleFunc("/order/order-1", func(w http.ResponseWriter, r *http.Request) {
		f.writeNonce(w)
		f.mu.Lock()
		status := f.orders["order-1"]
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":         status,
			"authorizations": []string{f.baseURL() + "/authz/authz-1"},
			"finalize":       f.baseURL() + "/finalize/order-1",
			"identifiers":    []map[string]string{{"type": "dns", "value": "spike.example.com"}},
		})
	})
	mux.HandleFunc("/finalize/order-1", func(w http.ResponseWriter, r *http.Request) {
		f.writeNonce(w)
		var req struct {
			Payload string `json:"payload"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		payloadRaw, _ := base64.RawURLEncoding.DecodeString(req.Payload)
		var finalize struct {
			CSR string `json:"csr"`
		}
		if err := json.Unmarshal(payloadRaw, &finalize); err != nil {
			w.WriteHeader(400)
			return
		}
		csrDER, err := base64.RawURLEncoding.DecodeString(finalize.CSR)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		csr, err := x509.ParseCertificateRequest(csrDER)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		f.mu.Lock()
		f.certDER = f.issueFromCSR(csr)
		f.orders["order-1"] = "valid"
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":      "valid",
			"certificate": f.baseURL() + "/cert/1",
		})
	})
	mux.HandleFunc("/cert/1", func(w http.ResponseWriter, r *http.Request) {
		f.writeNonce(w)
		w.Header().Set("Content-Type", "application/pem-certificate-chain")
		pem.Encode(w, &pem.Block{Type: "CERTIFICATE", Bytes: f.certDER})
	})
	return mux
}

func (f *fakeACME) parseJWK(protectedB64 string) map[string]any {
	protectedRaw, err := base64.RawURLEncoding.DecodeString(protectedB64)
	if err != nil {
		return nil
	}
	var prot struct {
		JWK map[string]any `json:"jwk"`
	}
	if err := json.Unmarshal(protectedRaw, &prot); err != nil {
		return nil
	}
	return prot.JWK
}

func (f *fakeACME) validate(keyAuth string) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	url := "http://" + f.challengeSrv + "/.well-known/acme-challenge/token-abc"
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	buf := make([]byte, 4096)
	n, _ := resp.Body.Read(buf)
	return strings.TrimSpace(string(buf[:n])) == keyAuth
}

func (f *fakeACME) issueFromCSR(csr *x509.CertificateRequest) []byte {
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      csr.Subject,
		DNSNames:     csr.DNSNames,
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(90 * 24 * time.Hour),
	}
	caKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, csr.PublicKey, caKey)
	if err != nil {
		panic(err)
	}
	return der
}

func (f *fakeACME) makeCert(host string) []byte {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: host},
		DNSNames:     []string{host},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(90 * 24 * time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	return der
}

func TestEngineIssueHTTP01(t *testing.T) {
	challengePort := freePort(t)
	fake := &fakeACME{challengeSrv: "127.0.0.1:" + challengePort, authzs: map[string]string{}, orders: map[string]string{}}
	srv := httptest.NewServer(fake.handler(t))
	defer srv.Close()
	fake.base = srv.URL
	fake.certDER = fake.makeCert("spike.example.com")

	stateDir := t.TempDir()
	cfg := &config.Config{
		StateDir:      stateDir,
		CertsDir:      stateDir + "/certs",
		KeysDir:       stateDir + "/keys",
		ChallengeAddr: "127.0.0.1:" + challengePort,
		CertEmail:     "ops@example.com",
		ACMEDirectory: srv.URL + "/dir",
		LogsDir:       stateDir + "/logs",
		BuildsDir:     stateDir + "/builds",
		CacheDir:      stateDir + "/cache",
		DataDir:       stateDir + "/data",
	}
	if err := cfg.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	secrets, err := security.LoadSecrets(cfg.KeysDir)
	if err != nil {
		t.Fatal(err)
	}
	sqldb := db.OpenTest(t)
	store := db.NewStore(sqldb)
	engine := NewEngine(cfg, secrets, store)
	var challengeHosts []string
	engine.SetChallenge = func(host string) { challengeHosts = append(challengeHosts, host) }
	engine.ClearChallenge = func(host string) {}
	if err := engine.StartChallengeServer(); err != nil {
		t.Fatal(err)
	}
	defer engine.StopChallengeServer()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := engine.Issue(ctx, "spike.example.com"); err != nil {
		t.Fatal(err)
	}
	if len(challengeHosts) != 1 || challengeHosts[0] != "spike.example.com" {
		t.Fatalf("challenge route não registrada: %v", challengeHosts)
	}
	row, err := store.GetCert("spike.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if !row.NotAfter.After(time.Now().Add(60 * 24 * time.Hour)) {
		t.Fatalf("certificado com validade curta: %v", row.NotAfter)
	}
	keyPEM, err := secrets.DecryptFile(row.KeyPath)
	if err != nil || !strings.Contains(string(keyPEM), "PRIVATE KEY") {
		t.Fatalf("chave privada não recuperável: %v", err)
	}
	if _, err := engine.LoadTLSConfig("spike.example.com"); err != nil {
		t.Fatalf("tls config: %v", err)
	}
}

func TestEngineEnsureSkipsValidCert(t *testing.T) {
	stateDir := t.TempDir()
	cfg := &config.Config{
		StateDir:      stateDir,
		CertsDir:      stateDir + "/certs",
		KeysDir:       stateDir + "/keys",
		ChallengeAddr: "127.0.0.1:" + freePort(t),
		CertEmail:     "ops@example.com",
		ACMEDirectory: "http://127.0.0.1:1/dir",
	}
	cfg.EnsureDirs()
	secrets, _ := security.LoadSecrets(cfg.KeysDir)
	sqldb := db.OpenTest(t)
	store := db.NewStore(sqldb)
	engine := NewEngine(cfg, secrets, store)
	engine.SetChallenge = func(string) {}
	engine.ClearChallenge = func(string) {}
	if err := store.SaveCert("ok.example.com", "/c/cert", "/c/key", time.Now().Add(60*24*time.Hour).UTC().Format(time.RFC3339), "test"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := engine.Ensure(ctx, "ok.example.com"); err != nil {
		t.Fatalf("Ensure deveria pular emissão com cert válido: %v", err)
	}
}

func freePort(t *testing.T) string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return strings.TrimPrefix(ln.Addr().String(), "127.0.0.1:")
}

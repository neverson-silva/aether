package cert

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/acme"

	"aether/internal/config"
	"aether/internal/db"
	"aether/internal/security"
)

type Engine struct {
	cfg            *config.Config
	secrets        *security.Secrets
	store          *db.Store
	challengeMu    sync.RWMutex
	challenges     map[string]string
	challengeSrv   *http.Server
	accountMu      sync.Mutex
	accounts       map[string]*acme.Client
	SetChallenge   func(host string)
	ClearChallenge func(host string)
	lockFn         func(name string, fn func())
}

func (e *Engine) SetLockFn(fn func(name string, fn func())) {
	e.lockFn = fn
}

func NewEngine(cfg *config.Config, secrets *security.Secrets, store *db.Store) *Engine {
	e := &Engine{
		cfg:        cfg,
		secrets:    secrets,
		store:      store,
		challenges: map[string]string{},
		accounts:   map[string]*acme.Client{},
	}
	return e
}

func (e *Engine) StartChallengeServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", e.serveChallenge)
	srv := &http.Server{Handler: mux}
	ln, err := net.Listen("tcp", e.cfg.ChallengeAddr)
	if err != nil {
		return err
	}
	e.challengeSrv = srv
	go srv.Serve(ln)
	return nil
}

func (e *Engine) StopChallengeServer() {
	if e.challengeSrv != nil {
		e.challengeSrv.Close()
	}
}

func (e *Engine) serveChallenge(w http.ResponseWriter, r *http.Request) {
	segments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(segments) == 0 {
		http.NotFound(w, r)
		return
	}
	token := segments[len(segments)-1]
	e.challengeMu.RLock()
	keyAuth, ok := e.challenges[token]
	e.challengeMu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, keyAuth)
}

func (e *Engine) putChallenge(token, keyAuth string) {
	e.challengeMu.Lock()
	e.challenges[token] = keyAuth
	e.challengeMu.Unlock()
}

func (e *Engine) dropChallenge(token string) {
	e.challengeMu.Lock()
	delete(e.challenges, token)
	e.challengeMu.Unlock()
}

func (e *Engine) clientFor(email string) (*acme.Client, error) {
	if email == "" {
		email = "admin@aether.local"
	}
	e.accountMu.Lock()
	defer e.accountMu.Unlock()
	if c, ok := e.accounts[email]; ok {
		return c, nil
	}
	keyPath := filepath.Join(e.cfg.KeysDir, "accounts", email+".key.enc")
	keyBytes, err := os.ReadFile(keyPath)
	if err == nil {
		raw, err := e.secrets.Decrypt(keyBytes)
		if err != nil {
			return nil, err
		}
		key, err := parseECKey(raw)
		if err != nil {
			return nil, err
		}
		client := &acme.Client{Key: key, DirectoryURL: e.cfg.ACMEDirectory}
		if err := e.ensureAccountRegistered(client, email); err != nil {
			return nil, err
		}
		e.accounts[email] = client
		return client, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	raw, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	enc, err := e.secrets.Encrypt(raw)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(keyPath, enc, 0o600); err != nil {
		return nil, err
	}
	client := &acme.Client{Key: key, DirectoryURL: e.cfg.ACMEDirectory}
	if _, err := client.Register(ctxNoTimeout(), &acme.Account{Contact: []string{"mailto:" + email}}, acme.AcceptTOS); err != nil {
		return nil, err
	}
	e.accounts[email] = client
	return client, nil
}

func (e *Engine) ensureAccountRegistered(client *acme.Client, email string) error {
	_, err := client.Register(ctxNoTimeout(), &acme.Account{Contact: []string{"mailto:" + email}}, acme.AcceptTOS)
	if err != nil && !errors.Is(err, acme.ErrAccountAlreadyExists) {
		return err
	}
	return nil
}

func parseECKey(raw []byte) (*ecdsa.PrivateKey, error) {
	if block, _ := pem.Decode(raw); block != nil {
		return x509.ParseECPrivateKey(block.Bytes)
	}
	return x509.ParseECPrivateKey(raw)
}

func ctxNoTimeout() context.Context {
	return context.Background()
}

func (e *Engine) Ensure(ctx context.Context, host string) error {
	if e.cfg.CertEmail == "" && e.cfg.ACMEDirectory == "" {
		return nil
	}
	row, err := e.store.GetCert(host)
	if err == nil && row.NotAfter.After(time.Now().Add(15*24*time.Hour)) {
		return nil
	}
	return e.Issue(ctx, host)
}

func (e *Engine) Issue(ctx context.Context, host string) error {
	email := e.cfg.CertEmail
	if email == "" {
		email = "admin@aether.local"
	}
	client, err := e.clientFor(email)
	if err != nil {
		return err
	}
	order, err := client.AuthorizeOrder(ctx, []acme.AuthzID{{Type: "dns", Value: host}})
	if err != nil {
		return fmt.Errorf("create order: %w", err)
	}
	if len(order.AuthzURLs) == 0 {
		return fmt.Errorf("order sem autorizações para %s", host)
	}
	authz, err := client.GetAuthorization(ctx, order.AuthzURLs[0])
	if err != nil {
		return fmt.Errorf("get authorization: %w", err)
	}
	var httpChal *acme.Challenge
	for _, ch := range authz.Challenges {
		if ch.Type == "http-01" {
			httpChal = ch
			break
		}
	}
	if httpChal == nil {
		return fmt.Errorf("no http-01 challenge for %s", host)
	}
	keyAuth, err := client.HTTP01ChallengeResponse(httpChal.Token)
	if err != nil {
		return err
	}
	if authz.Status != acme.StatusValid {
		e.putChallenge(httpChal.Token, keyAuth)
		if e.SetChallenge != nil {
			e.SetChallenge(host)
		}
		defer func() {
			if e.ClearChallenge != nil {
				e.ClearChallenge(host)
			}
			e.dropChallenge(httpChal.Token)
		}()
		if _, err := client.Accept(ctx, httpChal); err != nil {
			return fmt.Errorf("accept challenge: %w", err)
		}
		if _, err := client.WaitAuthorization(ctx, authz.URI); err != nil {
			return fmt.Errorf("wait authorization: %w", err)
		}
	}
	orderURI := order.URI
	if _, err := client.WaitOrder(ctx, orderURI); err != nil {
		return fmt.Errorf("wait order: %w", err)
	}
	order, err = client.GetOrder(ctx, orderURI)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}
	certKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	csrDER, err := makeCSR(certKey, host)
	if err != nil {
		return err
	}
	certs, _, err := client.CreateOrderCert(ctx, order.FinalizeURL, csrDER, true)
	if err != nil {
		order2, werr := client.WaitOrder(ctx, orderURI)
		if werr != nil {
			return fmt.Errorf("create order cert: %w", err)
		}
		if order2.CertURL == "" {
			return fmt.Errorf("create order cert: %w", err)
		}
		certs, err = client.FetchCert(ctx, order2.CertURL, true)
		if err != nil {
			return fmt.Errorf("fetch cert: %w", err)
		}
	}
	privDER, err := x509.MarshalPKCS8PrivateKey(certKey)
	if err != nil {
		return err
	}
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER})
	leaf, err := x509.ParseCertificate(certs[0])
	if err != nil {
		return errors.New("resposta de certificado inválida")
	}
	var chainPEM bytes.Buffer
	for _, der := range certs {
		pem.Encode(&chainPEM, &pem.Block{Type: "CERTIFICATE", Bytes: der})
	}
	dir := filepath.Join(e.cfg.CertsDir, host)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	keyPath := filepath.Join(dir, "privkey.pem.enc")
	if err := e.secrets.EncryptFile(privPEM, keyPath); err != nil {
		return err
	}
	certPath := filepath.Join(dir, "fullchain.pem")
	if err := os.WriteFile(certPath, chainPEM.Bytes(), 0o640); err != nil {
		return err
	}
	notAfter := leaf.NotAfter.UTC().Format(time.RFC3339)
	if err := e.store.SaveCert(host, certPath, keyPath, notAfter, "letsencrypt"); err != nil {
		return err
	}
	return nil
}

func makeCSR(key *ecdsa.PrivateKey, host string) ([]byte, error) {
	tmpl := &x509.CertificateRequest{
		Subject:  pkix.Name{CommonName: host},
		DNSNames: []string{host},
	}
	return x509.CreateCertificateRequest(rand.Reader, tmpl, key)
}

func (e *Engine) LoadTLSConfig(host string) (*tls.Config, error) {
	row, err := e.store.GetCert(host)
	if err != nil {
		return nil, err
	}
	keyPEM, err := e.secrets.DecryptFile(row.KeyPath)
	if err != nil {
		return nil, err
	}
	certPEM, err := os.ReadFile(row.CertPath)
	if err != nil {
		return nil, err
	}
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}
	return &tls.Config{Certificates: []tls.Certificate{cert}}, nil
}

func (e *Engine) RenewLoop(ctx context.Context, interval time.Duration) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(interval):
				certs, err := e.store.ListCerts()
				if err != nil {
					continue
				}
				for _, c := range certs {
					if c.NotAfter.Before(time.Now().Add(30 * 24 * time.Hour)) {
						issue := func() {
							if err := e.Issue(ctx, c.Host); err != nil {
								log.Printf("[cert] renovação de %s falhou: %v", c.Host, err)
							} else {
								log.Printf("[cert] renovado %s", c.Host)
							}
						}
						if e.lockFn != nil {
							e.lockFn("lock:cert:"+c.Host, issue)
						} else {
							issue()
						}
					}
				}
			}
		}
	}()
}

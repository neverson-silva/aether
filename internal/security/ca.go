package security

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

type CA struct {
	cert *x509.Certificate
	key  *rsa.PrivateKey
}

func LoadCA(certsDir string) (*CA, error) {
	certPath := filepath.Join(certsDir, "ca.pem")
	keyPath := filepath.Join(certsDir, "ca.key")
	if _, err := os.Stat(certPath); err != nil {
		return nil, errors.New("CA ausente: " + certPath)
	}
	if _, err := os.Stat(keyPath); err != nil {
		return nil, errors.New("CA key ausente: " + keyPath)
	}
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, errors.New("invalid ca.pem")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	kblock, _ := pem.Decode(keyPEM)
	if kblock == nil {
		return nil, errors.New("invalid ca.key")
	}
	key, err := x509.ParsePKCS1PrivateKey(kblock.Bytes)
	if err != nil {
		return nil, err
	}
	return &CA{cert: cert, key: key}, nil
}

func EnsureCA(certsDir string) (*CA, error) {
	ca, err := LoadCA(certsDir)
	if err == nil {
		return ca, nil
	}
	if err := os.MkdirAll(certsDir, 0o750); err != nil {
		return nil, err
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	tpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "aether-ca"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.AddDate(10, 0, 0),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(certsDir, "ca.pem"), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o640); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(certsDir, "ca.key"), pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}), 0o600); err != nil {
		return nil, err
	}
	return LoadCA(certsDir)
}

type AgentIdentity struct {
	CertPEM []byte
	KeyPEM  []byte
	CAPEM   []byte
}

func (c *CA) SignAgent(serverID string, days int) (*AgentIdentity, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	tpl := &x509.Certificate{
		SerialNumber: big.NewInt(now.Unix()),
		Subject:      pkix.Name{CommonName: "agent:" + serverID, Organization: []string{"aether"}},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.AddDate(0, 0, days),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, c.cert, &key.PublicKey, c.key)
	if err != nil {
		return nil, err
	}
	return &AgentIdentity{
		CertPEM: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		KeyPEM:  pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}),
		CAPEM:   pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: c.cert.Raw}),
	}, nil
}

func (c *CA) ServerTLS() (tls.Certificate, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}
	host, _ := os.Hostname()
	now := time.Now()
	tpl := &x509.Certificate{
		SerialNumber: big.NewInt(now.Unix()),
		Subject:      pkix.Name{CommonName: "aether-core", Organization: []string{"aether"}},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.AddDate(10, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost", host},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, c.cert, &key.PublicKey, c.key)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.X509KeyPair(
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}),
	)
}

func (c *CA) ClientPool() *x509.CertPool {
	pool := x509.NewCertPool()
	pool.AddCert(c.cert)
	return pool
}

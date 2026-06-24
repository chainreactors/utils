package cert

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"os"
	"sync"
)

type TLSOption func(*tls.Config)

var standardCipherSuites = []uint16{
	tls.TLS_CHACHA20_POLY1305_SHA256,
	tls.TLS_AES_128_GCM_SHA256,
	tls.TLS_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
}

// NewTLSConfig creates a *tls.Config with sensible defaults:
// TLS 1.2+, standard cipher suites. Use TLSOption to customize.
func NewTLSConfig(cert tls.Certificate, opts ...TLSOption) *tls.Config {
	cfg := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
		CipherSuites: standardCipherSuites,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

func TLSMinVersion(v uint16) TLSOption {
	return func(c *tls.Config) { c.MinVersion = v }
}

func TLSCipherSuites(suites []uint16) TLSOption {
	return func(c *tls.Config) { c.CipherSuites = suites }
}

func TLSInsecureSkipVerify() TLSOption {
	return func(c *tls.Config) { c.InsecureSkipVerify = true }
}

func TLSMutualAuth(clientCAs *x509.CertPool) TLSOption {
	return func(c *tls.Config) {
		c.ClientAuth = tls.RequireAndVerifyClientCert
		c.ClientCAs = clientCAs
	}
}

// TLSPinFingerprint sets InsecureSkipVerify=true and installs a custom
// VerifyPeerCertificate callback that verifies the peer certificate chain
// against roots, checks for the required ExtKeyUsage, and matches the
// leaf certificate's SHA-256 fingerprint.
func TLSPinFingerprint(roots *x509.CertPool, expectedFP string, requiredUsage x509.ExtKeyUsage) TLSOption {
	return func(c *tls.Config) {
		c.InsecureSkipVerify = true
		c.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return fmt.Errorf("no peer certificates")
			}
			cert, err := x509.ParseCertificate(rawCerts[0])
			if err != nil {
				return fmt.Errorf("parse peer certificate: %w", err)
			}

			opts := x509.VerifyOptions{Roots: roots}
			if requiredUsage != 0 {
				opts.KeyUsages = []x509.ExtKeyUsage{requiredUsage}
			}
			if _, err := cert.Verify(opts); err != nil {
				return fmt.Errorf("verify peer certificate chain: %w", err)
			}

			if requiredUsage != 0 && !HasExtKeyUsage(cert, requiredUsage) {
				return fmt.Errorf("peer certificate missing required %s usage", ExtKeyUsageName(requiredUsage))
			}

			if expectedFP != "" && NormalizeFingerprint(Fingerprint(cert)) != NormalizeFingerprint(expectedFP) {
				return fmt.Errorf("peer certificate fingerprint mismatch: got %s", Fingerprint(cert))
			}

			return nil
		}
	}
}

func TLSKeyLog() TLSOption {
	return func(c *tls.Config) {
		if w := KeyLogWriter(); w != nil {
			c.KeyLogWriter = w
		}
	}
}

func TLSServerName(name string) TLSOption {
	return func(c *tls.Config) { c.ServerName = name }
}

// --- Cert Pool ---

func LoadCertPool(caCertPEM []byte) (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCertPEM) {
		return nil, fmt.Errorf("failed to parse CA certificate PEM")
	}
	return pool, nil
}

func LoadCertPoolFromFile(path string) (*x509.CertPool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return LoadCertPool(data)
}

// --- Key Log ---

var (
	keyLogOnce   sync.Once
	keyLogWriter io.Writer
)

func KeyLogWriter() io.Writer {
	keyLogOnce.Do(func() {
		path := os.Getenv("SSLKEYLOGFILE")
		if path == "" {
			return
		}
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			return
		}
		keyLogWriter = f
	})
	return keyLogWriter
}

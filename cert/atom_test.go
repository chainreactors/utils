package cert

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"testing"
	"time"
)

// --- key.go ---

func TestGenerateRSAKey(t *testing.T) {
	key, err := generateRSAKey(2048)
	if err != nil {
		t.Fatal(err)
	}
	if key.N.BitLen() != 2048 {
		t.Fatalf("expected 2048 bits, got %d", key.N.BitLen())
	}
}

func TestGenerateRSAKeyRandom(t *testing.T) {
	key, err := generateRSAKey(0)
	if err != nil {
		t.Fatal(err)
	}
	bits := key.N.BitLen()
	if bits != 2048 && bits != 4096 {
		t.Fatalf("expected 2048 or 4096, got %d", bits)
	}
}

// --- template.go ---

func TestNewTemplate_Defaults(t *testing.T) {
	tmpl, err := NewTemplate()
	if err != nil {
		t.Fatal(err)
	}
	if tmpl.SerialNumber == nil {
		t.Fatal("serial number should be set")
	}
	if tmpl.IsCA {
		t.Fatal("default should not be CA")
	}
}

func TestNewTemplate_AsCA(t *testing.T) {
	tmpl, _ := NewTemplate(AsCA())
	if !tmpl.IsCA {
		t.Fatal("should be CA")
	}
	if tmpl.KeyUsage&x509.KeyUsageCertSign == 0 {
		t.Fatal("should have CertSign usage")
	}
}

func TestNewTemplate_WithSubjectCN(t *testing.T) {
	tmpl, _ := NewTemplate(WithSubjectCN("example.com"))
	if tmpl.Subject.CommonName != "example.com" {
		t.Fatalf("expected CN=example.com, got %s", tmpl.Subject.CommonName)
	}
	if len(tmpl.DNSNames) != 1 || tmpl.DNSNames[0] != "example.com" {
		t.Fatalf("expected DNS SAN, got %v", tmpl.DNSNames)
	}
}

func TestNewTemplate_WithSubjectCN_IP(t *testing.T) {
	tmpl, _ := NewTemplate(WithSubjectCN("10.0.0.1"))
	if len(tmpl.IPAddresses) != 1 {
		t.Fatal("expected IP SAN")
	}
}

func TestNewTemplate_WithRandomSubject(t *testing.T) {
	tmpl, _ := NewTemplate(WithRandomSubject(""))
	if tmpl.Subject.CommonName == "" {
		t.Fatal("should auto-generate CN")
	}
}

func TestNewTemplate_WithValidityWindow(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	tmpl, _ := NewTemplate(WithValidityWindow(start, 24*time.Hour))
	if !tmpl.NotBefore.Equal(start) {
		t.Fatal("NotBefore mismatch")
	}
	if !tmpl.NotAfter.Equal(start.Add(24 * time.Hour)) {
		t.Fatal("NotAfter mismatch")
	}
}

// --- sign.go ---

func TestSelfSign(t *testing.T) {
	key, _ := generateRSAKey(2048)
	tmpl, _ := NewTemplate(AsCA(), WithSubjectCN("test-ca"))
	der, err := SelfSign(tmpl, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, _ := x509.ParseCertificate(der)
	if !cert.IsCA {
		t.Fatal("should be CA")
	}
}

func TestSignWith(t *testing.T) {
	caKey, _ := generateRSAKey(2048)
	caTmpl, _ := NewTemplate(AsCA(), WithSubjectCN("test-ca"))
	caDER, _ := SelfSign(caTmpl, caKey)
	caCert, _ := x509.ParseCertificate(caDER)

	childKey, _ := generateRSAKey(2048)
	childTmpl, _ := NewTemplate(
		WithSubjectCN("child.example.com"),
		WithExtKeyUsages(x509.ExtKeyUsageServerAuth),
	)
	childDER, err := SignWith(childTmpl, caCert, childKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	childCert, _ := x509.ParseCertificate(childDER)

	pool := x509.NewCertPool()
	pool.AddCert(caCert)
	if _, err := childCert.Verify(x509.VerifyOptions{
		Roots:     pool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}); err != nil {
		t.Fatalf("chain verify failed: %v", err)
	}
}

func TestSignWith_KeyReuse(t *testing.T) {
	caKey, _ := generateRSAKey(2048)
	caTmpl, _ := NewTemplate(AsCA(), WithSubjectCN("test-ca"))
	caDER, _ := SelfSign(caTmpl, caKey)
	caCert, _ := x509.ParseCertificate(caDER)

	leafTmpl, _ := NewTemplate(
		WithSubjectCN("leaf.example.com"),
		WithExtKeyUsages(x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth),
	)
	leafDER, err := SignWith(leafTmpl, caCert, caKey, caKey)
	if err != nil {
		t.Fatal(err)
	}

	tlsCert := &tls.Certificate{Certificate: [][]byte{leafDER}, PrivateKey: caKey}
	if len(tlsCert.Certificate) != 1 {
		t.Fatal("should have 1 cert")
	}
	leafCert, _ := x509.ParseCertificate(leafDER)
	if leafCert.Subject.CommonName != "leaf.example.com" {
		t.Fatal("CN mismatch")
	}
}

// --- fingerprint.go ---

func TestFingerprint(t *testing.T) {
	key, _ := generateRSAKey(2048)
	tmpl, _ := NewTemplate(WithSubjectCN("fp-test"))
	der, _ := SelfSign(tmpl, key)
	cert, _ := x509.ParseCertificate(der)
	fp := Fingerprint(cert)
	if len(fp) != 64 {
		t.Fatalf("expected 64-char hex, got len=%d", len(fp))
	}
}

func TestNormalizeFingerprint(t *testing.T) {
	tests := []struct{ in, want string }{
		{"AA:BB:CC", "aabbcc"},
		{"aa bb cc", "aabbcc"},
		{"AABBCC", "aabbcc"},
	}
	for _, tt := range tests {
		if got := NormalizeFingerprint(tt.in); got != tt.want {
			t.Errorf("NormalizeFingerprint(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// --- verify.go ---

func TestHasExtKeyUsage(t *testing.T) {
	certPEM, _, _ := GenerateSelfSignedCert(2048)
	cert, _ := ParseCertificatePEM(certPEM)
	if !HasExtKeyUsage(cert, x509.ExtKeyUsageServerAuth) {
		t.Fatal("should have ServerAuth")
	}
	if HasExtKeyUsage(cert, x509.ExtKeyUsageCodeSigning) {
		t.Fatal("should not have CodeSigning")
	}
}

func TestExtKeyUsageName(t *testing.T) {
	if ExtKeyUsageName(x509.ExtKeyUsageServerAuth) != "ServerAuth" {
		t.Fatal("wrong name")
	}
}

// --- tls.go ---

func TestNewTLSConfig_Defaults(t *testing.T) {
	certPEM, keyPEM, _ := GenerateSelfSignedCert(2048)
	tlsCert, _ := tls.X509KeyPair(certPEM, keyPEM)
	cfg := NewTLSConfig(tlsCert)
	if cfg.MinVersion != tls.VersionTLS12 {
		t.Fatal("should require TLS 1.2+")
	}
	if len(cfg.CipherSuites) == 0 {
		t.Fatal("should have cipher suites")
	}
}

func TestNewTLSConfig_WithOptions(t *testing.T) {
	certPEM, keyPEM, _ := GenerateSelfSignedCert(2048)
	tlsCert, _ := tls.X509KeyPair(certPEM, keyPEM)
	cfg := NewTLSConfig(tlsCert,
		TLSMinVersion(tls.VersionTLS13),
		TLSInsecureSkipVerify(),
		TLSServerName("test.example.com"),
	)
	if cfg.MinVersion != tls.VersionTLS13 {
		t.Fatal("should be TLS 1.3")
	}
	if !cfg.InsecureSkipVerify {
		t.Fatal("should skip verify")
	}
	if cfg.ServerName != "test.example.com" {
		t.Fatal("server name mismatch")
	}
}

func TestNewTLSConfig_MutualAuth(t *testing.T) {
	certPEM, keyPEM, _ := GenerateSelfSignedCert(2048)
	tlsCert, _ := tls.X509KeyPair(certPEM, keyPEM)
	pool, _ := LoadCertPool(certPEM)
	cfg := NewTLSConfig(tlsCert, TLSMutualAuth(pool))
	if cfg.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Fatal("should require client cert")
	}
}

func TestNewTLSConfig_PinFingerprint(t *testing.T) {
	caCertPEM, caKeyPEM, _ := GenerateCACert(2048)
	caCert, _ := ParseCertificatePEM(caCertPEM)
	caKey, _ := ParseKeyPEM(caKeyPEM)
	pool, _ := LoadCertPool(caCertPEM)

	serverPEM, serverKeyPEM, _ := GenerateChildCert(2048, caCert, caKey,
		WithRandomSubject("server"),
		WithExtKeyUsages(x509.ExtKeyUsageServerAuth),
		WithAutoSAN("server"),
	)
	serverTLS, _ := tls.X509KeyPair(serverPEM, serverKeyPEM)
	serverX509, _ := ParseCertificatePEM(serverPEM)
	fp := Fingerprint(serverX509)

	cfg := NewTLSConfig(serverTLS, TLSPinFingerprint(pool, fp, x509.ExtKeyUsageServerAuth))
	if cfg.VerifyPeerCertificate == nil {
		t.Fatal("should have VerifyPeerCertificate callback")
	}
}

func TestLoadCertPool(t *testing.T) {
	certPEM, _, _ := GenerateCACert(2048)
	pool, err := LoadCertPool(certPEM)
	if err != nil {
		t.Fatal(err)
	}
	if pool == nil {
		t.Fatal("pool should not be nil")
	}
}

// --- pem.go ---

func TestParsePrivateKeyPEM_RSA(t *testing.T) {
	_, keyPEM, _ := GenerateCACert(2048)
	signer, err := ParsePrivateKeyPEM(keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	if signer == nil {
		t.Fatal("signer should not be nil")
	}
}

// --- full atom composition: mitmproxy DummyCert pattern ---

func TestAtomComposition_DummyCert(t *testing.T) {
	caKey, _ := generateRSAKey(2048)
	caTmpl, _ := NewTemplate(
		AsCA(),
		WithFullSubject(pkix.Name{CommonName: "mitmproxy", Organization: []string{"mitmproxy"}}),
		WithValidityWindow(time.Now().Add(-48*time.Hour), 3*365*24*time.Hour),
	)
	caDER, _ := SelfSign(caTmpl, caKey)
	caCert, _ := x509.ParseCertificate(caDER)

	leafTmpl, _ := NewTemplate(
		WithFullSubject(pkix.Name{CommonName: "example.com", Organization: []string{"mitmproxy"}}),
		WithValidityWindow(time.Now().Add(-48*time.Hour), 365*24*time.Hour),
		WithExtKeyUsages(x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth),
		WithAutoSAN("example.com"),
	)
	leafDER, err := SignWith(leafTmpl, caCert, caKey, caKey)
	if err != nil {
		t.Fatal(err)
	}

	leafCert, _ := x509.ParseCertificate(leafDER)
	if leafCert.Subject.CommonName != "example.com" {
		t.Fatal("CN mismatch")
	}
	if len(leafCert.DNSNames) == 0 || leafCert.DNSNames[0] != "example.com" {
		t.Fatal("SAN mismatch")
	}
}

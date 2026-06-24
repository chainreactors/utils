package cert

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGenerateCACert(t *testing.T) {
	certPEM, keyPEM, err := GenerateCACert(2048)
	if err != nil {
		t.Fatal(err)
	}
	cert, _ := ParseCertificatePEM(certPEM)
	if !cert.IsCA {
		t.Fatal("expected CA cert")
	}
	if cert.Subject.CommonName == "" {
		t.Fatal("expected auto-generated subject")
	}
	key, _ := ParseKeyPEM(keyPEM)
	if key.N.BitLen() != 2048 {
		t.Fatalf("expected 2048-bit key, got %d", key.N.BitLen())
	}
}

func TestGenerateCACertWithSubject(t *testing.T) {
	subject := pkix.Name{CommonName: "custom-ca", Organization: []string{"TestOrg"}}
	certPEM, _, err := GenerateCACert(2048, WithFullSubject(subject))
	if err != nil {
		t.Fatal(err)
	}
	cert, _ := ParseCertificatePEM(certPEM)
	if cert.Subject.Organization[0] != "TestOrg" {
		t.Fatalf("expected org=TestOrg, got %s", cert.Subject.Organization[0])
	}
}

func TestGenerateChildCert(t *testing.T) {
	caCertPEM, caKeyPEM, _ := GenerateCACert(2048)
	caCert, _ := ParseCertificatePEM(caCertPEM)
	caKey, _ := ParseKeyPEM(caKeyPEM)

	serverPEM, _, err := GenerateChildCert(2048, caCert, caKey,
		WithRandomSubject("server.example.com"),
		WithExtKeyUsages(x509.ExtKeyUsageServerAuth),
		WithAutoSAN("server.example.com"),
	)
	if err != nil {
		t.Fatal(err)
	}
	serverCert, _ := ParseCertificatePEM(serverPEM)
	if serverCert.IsCA {
		t.Fatal("child cert should not be CA")
	}
	found := false
	for _, dns := range serverCert.DNSNames {
		if dns == "server.example.com" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected server.example.com in DNSNames")
	}
}

func TestGenerateChildCert_ClientAuth(t *testing.T) {
	caCertPEM, caKeyPEM, _ := GenerateCACert(2048)
	caCert, _ := ParseCertificatePEM(caCertPEM)
	caKey, _ := ParseKeyPEM(caKeyPEM)

	clientPEM, _, _ := GenerateChildCert(2048, caCert, caKey,
		WithRandomSubject("client"),
		WithExtKeyUsages(x509.ExtKeyUsageClientAuth),
	)
	clientCert, _ := ParseCertificatePEM(clientPEM)
	if !HasExtKeyUsage(clientCert, x509.ExtKeyUsageClientAuth) {
		t.Fatal("should have ClientAuth usage")
	}
}

func TestGenerateChildCert_IP(t *testing.T) {
	caCertPEM, caKeyPEM, _ := GenerateCACert(2048)
	caCert, _ := ParseCertificatePEM(caCertPEM)
	caKey, _ := ParseKeyPEM(caKeyPEM)

	certPEM, _, _ := GenerateChildCert(2048, caCert, caKey,
		WithSubjectCN("192.168.1.1"),
		WithExtKeyUsages(x509.ExtKeyUsageServerAuth),
	)
	cert, _ := ParseCertificatePEM(certPEM)
	found := false
	for _, ip := range cert.IPAddresses {
		if ip.Equal(net.ParseIP("192.168.1.1")) {
			found = true
		}
	}
	if !found {
		t.Fatal("expected 192.168.1.1 in IPAddresses")
	}
}

func TestGenerateSelfSignedCert(t *testing.T) {
	certPEM, keyPEM, err := GenerateSelfSignedCert(2048, WithSubjectCN("test.example.com"))
	if err != nil {
		t.Fatal(err)
	}
	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	if len(tlsCert.Certificate) != 1 {
		t.Fatal("expected exactly 1 cert in chain")
	}
	cert, _ := ParseCertificatePEM(certPEM)
	if cert.IsCA {
		t.Fatal("self-signed leaf should not be CA")
	}
}

func TestGenerateSelfSignedCert_RandomCN(t *testing.T) {
	certPEM, _, err := GenerateSelfSignedCert(0)
	if err != nil {
		t.Fatal(err)
	}
	cert, _ := ParseCertificatePEM(certPEM)
	if cert.Subject.CommonName == "" {
		t.Fatal("expected auto-generated CN")
	}
}

func TestCAChildChainVerification(t *testing.T) {
	caCertPEM, caKeyPEM, _ := GenerateCACert(2048)
	caCert, _ := ParseCertificatePEM(caCertPEM)
	caKey, _ := ParseKeyPEM(caKeyPEM)

	childPEM, _, _ := GenerateChildCert(2048, caCert, caKey,
		WithRandomSubject("child"),
		WithExtKeyUsages(x509.ExtKeyUsageServerAuth),
	)
	childCert, _ := ParseCertificatePEM(childPEM)

	pool := x509.NewCertPool()
	pool.AddCert(caCert)
	_, err := childCert.Verify(x509.VerifyOptions{
		Roots:     pool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	})
	if err != nil {
		t.Fatalf("child cert should verify against CA: %v", err)
	}
}

func TestRandomKeySize(t *testing.T) {
	seen := map[int]bool{}
	for i := 0; i < 100; i++ {
		s := RandomKeySize()
		if s != 2048 && s != 4096 {
			t.Fatalf("unexpected key size: %d", s)
		}
		seen[s] = true
	}
	if !seen[2048] || !seen[4096] {
		t.Fatal("expected both key sizes in 100 iterations")
	}
}

func TestPEMRoundtrip(t *testing.T) {
	certPEM, keyPEM, _ := GenerateCACert(2048, WithFullSubject(pkix.Name{CommonName: "roundtrip"}))
	cert, _ := ParseCertificatePEM(certPEM)
	if cert.Subject.CommonName != "roundtrip" {
		t.Fatal("roundtrip failed for cert")
	}
	key, _ := ParseKeyPEM(keyPEM)
	_ = key.Public().(*rsa.PublicKey)
}

func TestSaveToPEMFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.pem")
	data := []byte("test PEM data")
	if err := SaveToPEMFile(path, data); err != nil {
		t.Fatal(err)
	}
	read, _ := os.ReadFile(path)
	if string(read) != "test PEM data" {
		t.Fatal("content mismatch")
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatalf("expected 0600, got %o", info.Mode().Perm())
	}
}

func TestWithValidityOverridesDefault(t *testing.T) {
	start := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	certPEM, _, _ := GenerateSelfSignedCert(2048,
		WithSubjectCN("test"),
		WithValidityWindow(start, 30*24*time.Hour),
	)
	cert, _ := ParseCertificatePEM(certPEM)
	if !cert.NotBefore.Equal(start) {
		t.Fatalf("expected NotBefore=%v, got %v", start, cert.NotBefore)
	}
	if cert.NotAfter.Sub(cert.NotBefore) != 30*24*time.Hour {
		t.Fatal("validity mismatch")
	}
}

package cert

import (
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"fmt"
	"math/big"
	mathrand "math/rand"
	"time"
)

func randomSerialNumber() (*big.Int, error) {
	return rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
}

func randomValidFor() time.Duration {
	days := 365 + mathrand.Intn(730)
	return time.Duration(days) * 24 * time.Hour
}

func randomBackdate() time.Time {
	days := mathrand.Intn(365)
	return time.Now().AddDate(0, 0, -days)
}

func SelfSign(template *x509.Certificate, signer crypto.Signer) ([]byte, error) {
	der, err := x509.CreateCertificate(rand.Reader, template, template, signer.Public(), signer)
	if err != nil {
		return nil, fmt.Errorf("self-sign certificate: %w", err)
	}
	return der, nil
}

func SignWith(template, parent *x509.Certificate, pub crypto.Signer, parentKey crypto.Signer) ([]byte, error) {
	der, err := x509.CreateCertificate(rand.Reader, template, parent, pub.Public(), parentKey)
	if err != nil {
		return nil, fmt.Errorf("sign certificate: %w", err)
	}
	return der, nil
}

package cert

import (
	"crypto"
	"crypto/x509"
)

// GenerateCACert generates a self-signed CA certificate.
// keySize 0 means random (2048/4096).
// Defaults applied before opts: AsCA, random validity, ServerAuth+ClientAuth EKU.
// If no subject is set via opts, a random subject is generated.
func GenerateCACert(keySize int, opts ...TemplateOption) (certPEM, keyPEM []byte, err error) {
	key, err := generateRSAKey(keySize)
	if err != nil {
		return nil, nil, err
	}

	defaults := []TemplateOption{
		AsCA(),
		WithRandomValidity(),
		WithExtKeyUsages(x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth),
	}
	tmpl, err := NewTemplate(append(defaults, opts...)...)
	if err != nil {
		return nil, nil, err
	}
	ensureSubject(tmpl)

	der, err := SelfSign(tmpl, key)
	if err != nil {
		return nil, nil, err
	}
	return EncodeCertPEM(der), mustEncodeKey(key), nil
}

// GenerateChildCert generates a certificate signed by a CA.
// keySize 0 means random. caKey must implement crypto.Signer.
// Defaults applied before opts: random validity.
// Caller should provide EKU via WithExtKeyUsages and subject via WithRandomSubject/WithFullSubject.
func GenerateChildCert(keySize int, caCert *x509.Certificate, caKey crypto.Signer, opts ...TemplateOption) (certPEM, keyPEM []byte, err error) {
	key, err := generateRSAKey(keySize)
	if err != nil {
		return nil, nil, err
	}

	defaults := []TemplateOption{WithRandomValidity()}
	tmpl, err := NewTemplate(append(defaults, opts...)...)
	if err != nil {
		return nil, nil, err
	}
	ensureSubject(tmpl)

	der, err := SignWith(tmpl, caCert, key, caKey)
	if err != nil {
		return nil, nil, err
	}
	return EncodeCertPEM(der), mustEncodeKey(key), nil
}

// GenerateSelfSignedCert generates a self-signed leaf certificate.
// keySize 0 means random.
// Defaults applied before opts: random validity, ServerAuth+ClientAuth EKU.
// If no subject is set via opts, a random subject is generated.
func GenerateSelfSignedCert(keySize int, opts ...TemplateOption) (certPEM, keyPEM []byte, err error) {
	key, err := generateRSAKey(keySize)
	if err != nil {
		return nil, nil, err
	}

	defaults := []TemplateOption{
		WithRandomValidity(),
		WithExtKeyUsages(x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth),
	}
	tmpl, err := NewTemplate(append(defaults, opts...)...)
	if err != nil {
		return nil, nil, err
	}
	ensureSubject(tmpl)

	der, err := SelfSign(tmpl, key)
	if err != nil {
		return nil, nil, err
	}
	return EncodeCertPEM(der), mustEncodeKey(key), nil
}

func ensureSubject(tmpl *x509.Certificate) {
	if tmpl.Subject.CommonName == "" && len(tmpl.Subject.Organization) == 0 {
		WithRandomSubject("")(tmpl)
	}
}

func mustEncodeKey(key interface{}) []byte {
	pem, err := EncodeKeyPEM(key)
	if err != nil {
		panic(err)
	}
	return pem
}

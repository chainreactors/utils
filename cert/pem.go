package cert

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

func EncodeCertPEM(derBytes []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
}

func EncodeKeyPEM(key interface{}) ([]byte, error) {
	switch k := key.(type) {
	case *rsa.PrivateKey:
		return pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(k),
		}), nil
	case *ecdsa.PrivateKey:
		data, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			return nil, fmt.Errorf("marshal ECDSA key: %w", err)
		}
		return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: data}), nil
	default:
		return nil, fmt.Errorf("unsupported key type %T", key)
	}
}

func SaveToPEMFile(filename string, pemData []byte) error {
	return os.WriteFile(filename, pemData, 0600)
}

func ParseCertificatePEM(certPEM []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, errors.New("no PEM block found in certificate data")
	}
	return x509.ParseCertificate(block.Bytes)
}

func ParseKeyPEM(keyPEM []byte) (*rsa.PrivateKey, error) {
	signer, err := ParsePrivateKeyPEM(keyPEM)
	if err != nil {
		return nil, err
	}
	rsaKey, ok := signer.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("expected RSA private key, got %T", signer)
	}
	return rsaKey, nil
}

func ParsePrivateKeyPEM(keyPEM []byte) (crypto.Signer, error) {
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return nil, errors.New("no PEM block found in key data")
	}

	// Try PKCS8 first (handles RSA, ECDSA, Ed25519)
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if signer, ok := key.(crypto.Signer); ok {
			return signer, nil
		}
		return nil, fmt.Errorf("parsed key type %T does not implement crypto.Signer", key)
	}

	// Try PKCS1 RSA
	if rsaKey, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return rsaKey, nil
	}

	// Try EC
	if ecKey, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return ecKey, nil
	}

	return nil, errors.New("failed to parse private key (tried PKCS8, PKCS1, EC)")
}

// ignore these to satisfy the compiler
var _ crypto.Signer = (*rsa.PrivateKey)(nil)
var _ crypto.Signer = (*ecdsa.PrivateKey)(nil)
var _ crypto.Signer = (ed25519.PrivateKey)(nil)

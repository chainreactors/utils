package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	mathrand "math/rand"
)

func RandomKeySize() int {
	if mathrand.Intn(2) == 0 {
		return 2048
	}
	return 4096
}

func generateRSAKey(bits int) (*rsa.PrivateKey, error) {
	if bits == 0 {
		bits = RandomKeySize()
	}
	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, fmt.Errorf("generate RSA key (%d bits): %w", bits, err)
	}
	return key, nil
}

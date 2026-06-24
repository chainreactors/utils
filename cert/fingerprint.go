package cert

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"strings"
)

func Fingerprint(cert *x509.Certificate) string {
	sum := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(sum[:])
}

func NormalizeFingerprint(fp string) string {
	fp = strings.ToLower(fp)
	fp = strings.ReplaceAll(fp, ":", "")
	fp = strings.ReplaceAll(fp, " ", "")
	return fp
}

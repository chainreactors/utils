package cert

import (
	"crypto/x509"
	"fmt"
)

func HasExtKeyUsage(cert *x509.Certificate, usage x509.ExtKeyUsage) bool {
	for _, u := range cert.ExtKeyUsage {
		if u == usage {
			return true
		}
	}
	return false
}

func ExtKeyUsageName(usage x509.ExtKeyUsage) string {
	switch usage {
	case x509.ExtKeyUsageAny:
		return "Any"
	case x509.ExtKeyUsageServerAuth:
		return "ServerAuth"
	case x509.ExtKeyUsageClientAuth:
		return "ClientAuth"
	case x509.ExtKeyUsageCodeSigning:
		return "CodeSigning"
	case x509.ExtKeyUsageEmailProtection:
		return "EmailProtection"
	case x509.ExtKeyUsageTimeStamping:
		return "TimeStamping"
	case x509.ExtKeyUsageOCSPSigning:
		return "OCSPSigning"
	default:
		return fmt.Sprintf("ExtKeyUsage(%d)", usage)
	}
}

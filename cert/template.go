package cert

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"net"
	"time"
)

type TemplateOption func(*x509.Certificate)

func NewTemplate(opts ...TemplateOption) (*x509.Certificate, error) {
	serial, err := randomSerialNumber()
	if err != nil {
		return nil, err
	}

	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
	}

	for _, opt := range opts {
		opt(tmpl)
	}

	return tmpl, nil
}

func AsCA() TemplateOption {
	return func(t *x509.Certificate) {
		t.IsCA = true
		t.KeyUsage |= x509.KeyUsageCertSign
	}
}

func WithSubjectCN(cn string) TemplateOption {
	return func(t *x509.Certificate) {
		t.Subject.CommonName = cn
		applyAutoSAN(t, cn)
	}
}

func WithFullSubject(name pkix.Name) TemplateOption {
	return func(t *x509.Certificate) {
		t.Subject = name
	}
}

func WithRandomSubject(cn string) TemplateOption {
	return func(t *x509.Certificate) {
		s := RandomSubject(cn)
		t.Subject = *s
		applyAutoSAN(t, s.CommonName)
	}
}

func WithAutoSAN(cn string) TemplateOption {
	return func(t *x509.Certificate) {
		applyAutoSAN(t, cn)
	}
}

func applyAutoSAN(t *x509.Certificate, cn string) {
	if cn == "" {
		return
	}
	if ip := net.ParseIP(cn); ip != nil {
		t.IPAddresses = append(t.IPAddresses, ip)
	} else {
		t.DNSNames = append(t.DNSNames, cn)
	}
}

func WithDNSNames(names ...string) TemplateOption {
	return func(t *x509.Certificate) {
		t.DNSNames = append(t.DNSNames, names...)
	}
}

func WithIPAddresses(ips ...net.IP) TemplateOption {
	return func(t *x509.Certificate) {
		t.IPAddresses = append(t.IPAddresses, ips...)
	}
}

func WithValidityWindow(notBefore time.Time, validFor time.Duration) TemplateOption {
	return func(t *x509.Certificate) {
		t.NotBefore = notBefore
		t.NotAfter = notBefore.Add(validFor)
	}
}

func WithRandomValidity() TemplateOption {
	return func(t *x509.Certificate) {
		t.NotBefore = randomBackdate()
		t.NotAfter = t.NotBefore.Add(randomValidFor())
	}
}

func WithKeyUsages(usage x509.KeyUsage) TemplateOption {
	return func(t *x509.Certificate) {
		t.KeyUsage = usage
	}
}

func WithExtKeyUsages(usages ...x509.ExtKeyUsage) TemplateOption {
	return func(t *x509.Certificate) {
		t.ExtKeyUsage = usages
	}
}

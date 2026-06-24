package cert

import (
	"crypto/x509/pkix"
	"fmt"
	"math/rand"
)

type GeoEntry struct {
	Country  string
	Province string
	Locality string
	Streets  []string
}

var DefaultGeoData = []GeoEntry{
	// US
	{"US", "California", "San Francisco", []string{"1000 Market St", "2000 Mission St"}},
	{"US", "California", "San Jose", []string{"606 Market St", "707 First St"}},
	{"US", "California", "Los Angeles", []string{"202 Hollywood Blvd", "303 Sunset Blvd"}},
	{"US", "California", "San Diego", []string{"404 Beach Ave", "505 Ocean Blvd"}},
	{"US", "New York", "New York", []string{"123 Broadway", "456 Wall St"}},
	{"US", "New York", "Buffalo", []string{"2021 Main St", "2223 Elmwood Ave"}},
	{"US", "Texas", "Austin", nil},
	{"US", "Texas", "Houston", nil},
	{"US", "Washington", "Seattle", []string{"123 Pike Pl", "456 Rainier Ave"}},
	{"US", "Florida", "Miami", []string{"123 South Beach Blvd", "456 Ocean Dr"}},
	{"US", "Florida", "Orlando", nil},
	{"US", "Illinois", "Chicago", []string{"123 Michigan Ave", "456 State St"}},
	{"US", "Massachusetts", "Boston", []string{"123 Beacon St", "456 Commonwealth Ave"}},
	{"US", "Colorado", "Denver", []string{"123 Downtown St", "456 Mountain Ave"}},
	{"US", "Ohio", "Columbus", nil},
	{"US", "Michigan", "Detroit", nil},
	{"US", "Minnesota", "Minneapolis", nil},
	{"US", "New Jersey", "Newark", nil},
	{"US", "North Carolina", "Charlotte", nil},
	{"US", "Indiana", "Indianapolis", nil},
	{"US", "Connecticut", "New Haven", nil},
	{"US", "Arizona", "Phoenix", []string{"123 Main St", "456 Elm St"}},
	{"US", "Oregon", "Portland", nil},
	{"US", "Pennsylvania", "Philadelphia", nil},
	// CA
	{"CA", "Alberta", "Calgary", []string{"1234 1st Ave", "5678 2nd Ave"}},
	{"CA", "Alberta", "Edmonton", nil},
	{"CA", "British Columbia", "Vancouver", []string{"123 Main St", "456 Granville St"}},
	{"CA", "British Columbia", "Victoria", nil},
	{"CA", "Ontario", "Toronto", nil},
	{"CA", "Ontario", "Ottawa", nil},
	{"CA", "Manitoba", "Winnipeg", nil},
	{"CA", "Quebec", "Montreal", nil},
	// JP
	{"JP", "Tokyo", "Chiyoda", nil},
	{"JP", "Tokyo", "Shibuya", nil},
	{"JP", "Aichi", "Nagoya", []string{"123-4567 Aza Kitayama", "789-0123 Aza Minamiyama"}},
	{"JP", "Chiba", "Chiba", nil},
}

type SubjectOption func(*subjectConfig)

type subjectConfig struct {
	geoData     []GeoEntry
	adjectives  []string
	nouns       []string
	orgSuffixes []string
}

func WithGeoData(entries []GeoEntry) SubjectOption {
	return func(c *subjectConfig) { c.geoData = entries }
}

func WithWordLists(adjectives, nouns []string) SubjectOption {
	return func(c *subjectConfig) {
		c.adjectives = adjectives
		c.nouns = nouns
	}
}

func WithOrgSuffixes(suffixes []string) SubjectOption {
	return func(c *subjectConfig) { c.orgSuffixes = suffixes }
}

func defaultSubjectConfig() *subjectConfig {
	return &subjectConfig{
		geoData:     DefaultGeoData,
		adjectives:  defaultAdjectives,
		nouns:       defaultNouns,
		orgSuffixes: defaultOrgSuffixes,
	}
}

func RandomSubject(cn string) *pkix.Name {
	return RandomSubjectWith(cn)
}

func RandomSubjectWith(cn string, opts ...SubjectOption) *pkix.Name {
	cfg := defaultSubjectConfig()
	for _, o := range opts {
		o(cfg)
	}

	if cn == "" {
		cn = randomCommonNameWith(cfg)
	}

	geo := cfg.geoData[rand.Intn(len(cfg.geoData))]
	org := randomOrganizationWith(cfg)

	name := &pkix.Name{
		CommonName:         cn,
		Organization:       org,
		OrganizationalUnit: randomOrganizationWith(cfg),
		Country:            []string{geo.Country},
		Province:           []string{geo.Province},
		Locality:           []string{geo.Locality},
	}

	if len(geo.Streets) > 0 {
		name.StreetAddress = []string{geo.Streets[rand.Intn(len(geo.Streets))]}
	}

	if rand.Intn(5) == 0 {
		name.PostalCode = []string{randomPostalCode(geo.Country)}
	}

	return name
}

func randomCommonName() string {
	return randomCommonNameWith(defaultSubjectConfig())
}

func randomCommonNameWith(cfg *subjectConfig) string {
	adj := cfg.adjectives[rand.Intn(len(cfg.adjectives))]
	noun := cfg.nouns[rand.Intn(len(cfg.nouns))]
	tld := defaultTLDs[rand.Intn(len(defaultTLDs))]
	return fmt.Sprintf("%s-%s.%s", adj, noun, tld)
}

func randomOrganization() []string {
	return randomOrganizationWith(defaultSubjectConfig())
}

func randomOrganizationWith(cfg *subjectConfig) []string {
	adj := cfg.adjectives[rand.Intn(len(cfg.adjectives))]
	noun := cfg.nouns[rand.Intn(len(cfg.nouns))]
	suffix := cfg.orgSuffixes[rand.Intn(len(cfg.orgSuffixes))]

	adjTitle := titleCase(adj)
	nounTitle := titleCase(noun)

	formats := []string{
		fmt.Sprintf("%s %s %s", adjTitle, nounTitle, suffix),
		fmt.Sprintf("%s%s %s", adjTitle, nounTitle, suffix),
		fmt.Sprintf("%s %s", adjTitle, suffix),
		fmt.Sprintf("%s %s", nounTitle, suffix),
		fmt.Sprintf("%s%s", adjTitle, nounTitle),
		fmt.Sprintf("%s & %s %s", adjTitle, nounTitle, suffix),
		fmt.Sprintf("%s %s %s", adjTitle, nounTitle, suffix),
		fmt.Sprintf("The %s %s", nounTitle, suffix),
	}

	return []string{formats[rand.Intn(len(formats))]}
}

func titleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	b := []byte(s)
	if b[0] >= 'a' && b[0] <= 'z' {
		b[0] -= 'a' - 'A'
	}
	return string(b)
}

func randomPostalCode(country string) string {
	switch country {
	case "US":
		return fmt.Sprintf("%05d", rand.Intn(100000))
	case "CA":
		letters := "ABCEGHJKLMNPRSTVXY"
		return fmt.Sprintf("%c%d%c %d%c%d",
			letters[rand.Intn(len(letters))],
			rand.Intn(10),
			'A'+byte(rand.Intn(26)),
			rand.Intn(10),
			'A'+byte(rand.Intn(26)),
			rand.Intn(10),
		)
	default:
		return fmt.Sprintf("%d", 100+rand.Intn(900))
	}
}


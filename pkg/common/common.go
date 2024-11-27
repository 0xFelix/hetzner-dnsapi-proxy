package common

const (
	KeyDNSRecord = "KeyDNSRecord"
)

type DNSRecord struct {
	FullName string
	Name     string
	Zone     string
	Value    string
	Type     string
}

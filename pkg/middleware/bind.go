package middleware

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/netip"
	"strings"

	"golang.org/x/net/publicsuffix"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/data"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/hetzner"
)

var errInvalidFQDN = errors.New("invalid fqdn")

const (
	failedParseRequestFmt = "failed to parse request: %v"
	maxRequestBodySize    = 1 << 10 // 1 KB
)

func BindPlain(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
		if err := r.ParseForm(); err != nil {
			log.Printf(failedParseRequestFmt, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		hostname := NormalizeFQDN(r.Form.Get("hostname"))
		ip := r.Form.Get("ip")
		if hostname == "" || ip == "" {
			http.Error(w, "hostname or ip address is missing", http.StatusBadRequest)
			return
		}

		addr, err := parseAddr(ip)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		recordType := config.RecordTypeA
		if !addr.Is4() {
			recordType = config.RecordTypeAAAA
		}

		name, zone, err := SplitFQDN(hostname)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		username, password, _ := r.BasicAuth()
		next.ServeHTTP(
			w, r.WithContext(
				data.NewContextWithReqData(
					r.Context(),
					&data.ReqData{
						FullName:  hostname,
						Name:      name,
						Zone:      zone,
						Value:     addr.String(),
						Type:      recordType,
						Username:  username,
						Password:  password,
						BasicAuth: true,
					},
				),
			),
		)
	})
}

func BindAcmeDNS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
		d := &struct {
			Subdomain string `json:"subdomain"`
			TXT       string `json:"txt"`
		}{}
		if err := json.UnmarshalRead(r.Body, d); err != nil {
			log.Printf(failedParseRequestFmt, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if d.Subdomain == "" || d.TXT == "" {
			http.Error(w, "subdomain or txt is missing", http.StatusBadRequest)
			return
		}

		fqdn := NormalizeFQDN(d.Subdomain)
		name, zone, err := SplitFQDN(fqdn)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// prepend prefix if not already given
		const prefixAcmeChallenge = "_acme-challenge."
		if !strings.HasPrefix(fqdn, prefixAcmeChallenge) {
			fqdn = prefixAcmeChallenge + fqdn
			// the name is empty if the subdomain is the zone apex
			name = strings.TrimSuffix(prefixAcmeChallenge+name, ".")
			// the prefix can make the name invalid, e.g. too long
			if err := ValidateFQDN(fqdn); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}

		next.ServeHTTP(
			w, r.WithContext(
				data.NewContextWithReqData(
					r.Context(),
					&data.ReqData{
						FullName:  fqdn,
						Name:      name,
						Zone:      zone,
						Value:     d.TXT,
						Type:      config.RecordTypeTXT,
						Username:  r.Header.Get("X-Api-User"),
						Password:  r.Header.Get("X-Api-Key"),
						BasicAuth: false,
					},
				),
			),
		)
	})
}

func BindHTTPReq(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
		d := &struct {
			FQDN  string `json:"fqdn"`
			Value string `json:"value"`
		}{}
		if err := json.UnmarshalRead(r.Body, d); err != nil {
			log.Printf(failedParseRequestFmt, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if d.FQDN == "" {
			http.Error(w, "fqdn is missing", http.StatusBadRequest)
			return
		}

		if d.Value == "" {
			http.Error(w, "value is missing", http.StatusBadRequest)
			return
		}

		d.FQDN = NormalizeFQDN(d.FQDN)
		name, zone, err := SplitFQDN(d.FQDN)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Values are appended, so that several challenges for the same name,
		// e.g. for example.com and *.example.com, can be present at once.
		// Cleanup removes only the given value.
		username, password, _ := r.BasicAuth()
		next.ServeHTTP(
			w, r.WithContext(
				data.NewContextWithReqData(
					r.Context(),
					&data.ReqData{
						FullName:  d.FQDN,
						Name:      name,
						Zone:      zone,
						Value:     d.Value,
						Type:      config.RecordTypeTXT,
						Username:  username,
						Password:  password,
						BasicAuth: true,
						Mode:      data.ModeAppend,
					},
				),
			),
		)
	})
}

// BindRecords binds requests of the generic records endpoint, which in
// addition to A, AAAA and TXT also supports CNAME and SRV records. Whether
// the type is allowed by cfg.RecordTypes is checked after authorization by
// NewRecordTypeAllowed, so unauthenticated clients cannot probe it.
func BindRecords(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
		d := &struct {
			FQDN    string `json:"fqdn"`
			Type    string `json:"type"`
			Value   string `json:"value"`
			Replace bool   `json:"replace"`
		}{}
		if err := json.UnmarshalRead(r.Body, d); err != nil {
			log.Printf(failedParseRequestFmt, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if d.FQDN == "" || d.Type == "" || d.Value == "" {
			http.Error(w, "fqdn, type or value is missing", http.StatusBadRequest)
			return
		}

		recordType := strings.ToUpper(d.Type)
		if !config.IsSupportedRecordType(recordType) {
			http.Error(w, config.UnsupportedRecordTypeMsg, http.StatusBadRequest)
			return
		}

		fqdn := NormalizeFQDN(d.FQDN)
		name, zone, err := SplitFQDN(fqdn)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if recordType == config.RecordTypeCNAME && name == "" {
			http.Error(w, "a CNAME record is not allowed at the zone apex", http.StatusBadRequest)
			return
		}

		value, err := normalizeRecordValue(d.Value, recordType)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// SRV records usually have several targets sharing one name, so they
		// are appended unless a replace is requested explicitly. A target of
		// '.' means that the service is not available and is valid only as
		// the sole record, so it always replaces.
		mode := data.ModeReplace
		if recordType == config.RecordTypeSRV && !d.Replace && !hetzner.SRVUnavailable(value) {
			mode = data.ModeAppend
		}

		username, password, _ := r.BasicAuth()
		next.ServeHTTP(
			w, r.WithContext(
				data.NewContextWithReqData(
					r.Context(),
					&data.ReqData{
						FullName:       fqdn,
						Name:           name,
						Zone:           zone,
						Value:          value,
						Type:           recordType,
						Username:       username,
						Password:       password,
						BasicAuth:      true,
						Mode:           mode,
						CheckConflicts: true,
					},
				),
			),
		)
	})
}

func BindDirectAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
		if err := r.ParseForm(); err != nil {
			log.Printf(failedParseRequestFmt, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		domain := r.Form.Get("domain")
		action := r.Form.Get("action")
		if domain == "" || action == "" {
			http.Error(w, "domain or action is missing", http.StatusBadRequest)
			return
		}

		if action != "add" {
			StatusOkDirectAdmin(next).ServeHTTP(w, r)
			return
		}

		recordType := r.Form.Get("type")
		if recordType != config.RecordTypeA && recordType != config.RecordTypeAAAA && recordType != config.RecordTypeTXT {
			http.Error(w, "type can only be A, AAAA or TXT", http.StatusBadRequest)
			return
		}

		value, err := normalizeRecordValue(r.Form.Get("value"), recordType)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		fqdn := domain
		if name := r.Form.Get("name"); name != "" {
			fqdn = name + "." + domain
		}
		fqdn = NormalizeFQDN(fqdn)

		name, zone, err := SplitFQDN(fqdn)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		username, password, _ := r.BasicAuth()
		next.ServeHTTP(
			w, r.WithContext(
				data.NewContextWithReqData(
					r.Context(),
					&data.ReqData{
						FullName:  fqdn,
						Name:      name,
						Zone:      zone,
						Value:     value,
						Type:      recordType,
						Username:  username,
						Password:  password,
						BasicAuth: true,
					},
				),
			),
		)
	})
}

// parseAddr parses an IP address without zone and returns it unmapped, so
// that IPv4-mapped IPv6 addresses are treated as IPv4 addresses.
func parseAddr(value string) (netip.Addr, error) {
	addr, err := netip.ParseAddr(value)
	if err != nil {
		return netip.Addr{}, errors.New("invalid ip address")
	}
	if addr.Zone() != "" {
		return netip.Addr{}, errors.New("ip address must not have a zone")
	}
	return addr.Unmap(), nil
}

// normalizeIP validates the value for the given address record type and
// returns it in its canonical form.
func normalizeIP(value, recordType string) (string, error) {
	addr, err := parseAddr(value)
	if err != nil {
		return "", err
	}
	if recordType == config.RecordTypeA && !addr.Is4() {
		return "", errors.New("invalid ipv4 address")
	}
	if recordType == config.RecordTypeAAAA && addr.Is4() {
		return "", errors.New("invalid ipv6 address")
	}
	return addr.String(), nil
}

// normalizeRecordValue validates the value for the given record type and
// returns it in the form expected by the API. IP addresses are canonicalized
// and hostnames in CNAME and SRV values are made absolute by appending a
// trailing dot.
func normalizeRecordValue(value, recordType string) (string, error) {
	switch recordType {
	case config.RecordTypeA, config.RecordTypeAAAA:
		return normalizeIP(value, recordType)
	case config.RecordTypeTXT:
		return value, nil
	case config.RecordTypeCNAME:
		target, err := hetzner.NormalizeHostname(value)
		if err != nil {
			return "", err
		}
		return target + ".", nil
	case config.RecordTypeSRV:
		return hetzner.NormalizeSRV(value)
	default:
		return "", errors.New(config.UnsupportedRecordTypeMsg)
	}
}

// NormalizeFQDN strips trailing dots and lowercases the name, so that
// comparisons against configured domains are case-insensitive.
func NormalizeFQDN(fqdn string) string {
	return strings.ToLower(strings.TrimRight(fqdn, "."))
}

// ValidateFQDN checks a normalized name against hostname rules (see
// hetzner.NormalizeHostname). A '*' is allowed as the leftmost label only, so
// that wildcard records can be set.
func ValidateFQDN(fqdn string) error {
	if len(fqdn) > hetzner.MaxHostnameLength {
		return errInvalidFQDN
	}
	if _, err := hetzner.NormalizeHostname(strings.TrimPrefix(fqdn, "*.")); err != nil {
		return errInvalidFQDN
	}
	return nil
}

// SplitFQDN validates a normalized name with ValidateFQDN and splits it into
// the name relative to its zone and the zone.
func SplitFQDN(fqdn string) (name, zone string, err error) {
	if vErr := ValidateFQDN(fqdn); vErr != nil {
		return "", "", vErr
	}

	zone, err = publicsuffix.EffectiveTLDPlusOne(fqdn)
	if err != nil || strings.Contains(zone, "*") {
		return "", "", fmt.Errorf("invalid fqdn: %s", fqdn)
	}

	if fqdn == zone {
		return "", zone, nil
	}

	name = strings.TrimSuffix(fqdn, "."+zone)
	return name, zone, nil
}

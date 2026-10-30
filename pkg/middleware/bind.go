package middleware

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/netip"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/publicsuffix"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/data"
)

const (
	recordTypeA           = "A"
	recordTypeAAAA        = "AAAA"
	recordTypeTXT         = "TXT"
	recordTypeCNAME       = "CNAME"
	recordTypeSRV         = "SRV"
	failedParseRequestFmt = "failed to parse request: %v"
	maxRequestBodySize    = 1 << 10 // 1 KB
	maxHostnameLength     = 253
	srvFieldCount         = 4
)

var hostnameRegexp = regexp.MustCompile(`^([a-z0-9_]([a-z0-9_-]{0,61}[a-z0-9_])?\.)*[a-z0-9_]([a-z0-9_-]{0,61}[a-z0-9_])?$`)

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

		addr, err := netip.ParseAddr(ip)
		if err != nil {
			http.Error(w, "invalid ip address", http.StatusBadRequest)
			return
		}

		recordType := recordTypeA
		if !addr.Unmap().Is4() {
			recordType = recordTypeAAAA
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
						Value:     ip,
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
						Type:      recordTypeTXT,
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
						Type:      recordTypeTXT,
						Username:  username,
						Password:  password,
						BasicAuth: true,
					},
				),
			),
		)
	})
}

// BindRecords binds requests of the generic records endpoint, which in
// addition to A, AAAA and TXT also supports CNAME and SRV records.
func BindRecords(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
		d := &struct {
			FQDN  string `json:"fqdn"`
			Type  string `json:"type"`
			Value string `json:"value"`
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
		fqdn := NormalizeFQDN(d.FQDN)
		name, zone, err := SplitFQDN(fqdn)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if recordType == recordTypeCNAME && name == "" {
			http.Error(w, "a CNAME record is not allowed at the zone apex", http.StatusBadRequest)
			return
		}

		value, err := normalizeRecordValue(d.Value, recordType)
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
		if recordType != recordTypeA && recordType != recordTypeAAAA && recordType != recordTypeTXT {
			http.Error(w, "type can only be A, AAAA or TXT", http.StatusBadRequest)
			return
		}

		value := r.Form.Get("value")
		if err := validateValue(value, recordType); err != nil {
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

func validateValue(value, recordType string) error {
	if recordType == recordTypeA || recordType == recordTypeAAAA {
		addr, err := netip.ParseAddr(value)
		if err != nil {
			return errors.New("invalid ip address")
		}
		if recordType == recordTypeA && !addr.Unmap().Is4() {
			return errors.New("invalid ipv4 address")
		}
		if recordType == recordTypeAAAA && addr.Unmap().Is4() {
			return errors.New("invalid ipv6 address")
		}
	}
	return nil
}

// normalizeRecordValue validates the value for the given record type and
// returns it in the form expected by the API. Hostnames in CNAME and SRV
// values are made absolute by appending a trailing dot.
func normalizeRecordValue(value, recordType string) (string, error) {
	switch recordType {
	case recordTypeA, recordTypeAAAA:
		return value, validateValue(value, recordType)
	case recordTypeTXT:
		return value, nil
	case recordTypeCNAME:
		target, err := normalizeHostname(value)
		if err != nil {
			return "", err
		}
		return target + ".", nil
	case recordTypeSRV:
		// priority weight port target
		fields := strings.Fields(value)
		if len(fields) != srvFieldCount {
			return "", errors.New("srv value must be 'priority weight port target'")
		}
		for i, f := range fields[:srvFieldCount-1] {
			n, err := strconv.ParseUint(f, 10, 16)
			if err != nil {
				return "", errors.New("srv priority, weight and port must be numbers between 0 and 65535")
			}
			// Canonicalize, e.g. strip leading zeros, so values compare equal
			fields[i] = strconv.FormatUint(n, 10)
		}
		// A target of '.' means that the service is not available.
		if fields[srvFieldCount-1] != "." {
			target, err := normalizeHostname(fields[srvFieldCount-1])
			if err != nil {
				return "", err
			}
			fields[srvFieldCount-1] = target + "."
		}
		return strings.Join(fields, " "), nil
	default:
		return "", errors.New("type can only be A, AAAA, TXT, CNAME or SRV")
	}
}

func normalizeHostname(host string) (string, error) {
	host = NormalizeFQDN(host)
	if len(host) > maxHostnameLength || !hostnameRegexp.MatchString(host) {
		return "", errors.New("invalid hostname")
	}
	return host, nil
}

// NormalizeFQDN strips trailing dots and lowercases the name, so that
// comparisons against configured domains are case-insensitive.
func NormalizeFQDN(fqdn string) string {
	return strings.ToLower(strings.TrimRight(fqdn, "."))
}

func SplitFQDN(fqdn string) (name, zone string, err error) {
	zone, err = publicsuffix.EffectiveTLDPlusOne(fqdn)
	if err != nil {
		return "", "", fmt.Errorf("invalid fqdn: %s", fqdn)
	}

	if fqdn == zone {
		return "", zone, nil
	}

	name = strings.TrimSuffix(fqdn, "."+zone)
	return name, zone, nil
}

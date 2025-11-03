package middleware

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/internal/model"
)

const (
	recordTypeA           = "A"
	recordTypeTXT         = "TXT"
	failedParseRequestFmt = "failed to parse request: %v"
)

func BindPlain(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			log.Printf(failedParseRequestFmt, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		hostname := r.Form.Get("hostname")
		ip := r.Form.Get("ip")
		if hostname == "" || ip == "" {
			http.Error(w, "hostname or ip address is missing", http.StatusBadRequest)
			return
		}

		name, zone, err := SplitFQDN(hostname)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		username, password, _ := r.BasicAuth()
		next.ServeHTTP(w, r.WithContext(
			model.NewContextWithReqData(r.Context(),
				&model.ReqData{
					FullName:  hostname,
					Name:      name,
					Zone:      zone,
					Value:     ip,
					Type:      recordTypeA,
					Username:  username,
					Password:  password,
					BasicAuth: true,
				},
			)),
		)
	})
}

func BindAcmeDNS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := &struct {
			Subdomain string `json:"subdomain"`
			TXT       string `json:"txt"`
		}{}
		if err := json.NewDecoder(r.Body).Decode(data); err != nil {
			log.Printf(failedParseRequestFmt, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if data.Subdomain == "" || data.TXT == "" {
			http.Error(w, "subdomain or txt is missing", http.StatusBadRequest)
			return
		}

		name, zone, err := SplitFQDN(data.Subdomain)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// prepend prefix if not already given
		const prefixAcmeChallenge = "_acme-challenge."
		if !strings.HasPrefix(data.Subdomain, prefixAcmeChallenge) {
			data.Subdomain = prefixAcmeChallenge + data.Subdomain
			name = prefixAcmeChallenge + name
		}

		next.ServeHTTP(w, r.WithContext(
			model.NewContextWithReqData(r.Context(),
				&model.ReqData{
					FullName:  data.Subdomain,
					Name:      name,
					Zone:      zone,
					Value:     data.TXT,
					Type:      recordTypeTXT,
					Username:  r.Header.Get("X-Api-User"),
					Password:  r.Header.Get("X-Api-Key"),
					BasicAuth: false,
				},
			)),
		)
	})
}

func BindHTTPReq(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := &struct {
			FQDN  string `json:"fqdn"`
			Value string `json:"value"`
		}{}
		if err := json.NewDecoder(r.Body).Decode(data); err != nil {
			log.Printf(failedParseRequestFmt, err)
			w.WriteHeader(http.StatusBadRequest)
		}

		if data.FQDN == "" {
			http.Error(w, "fqdn is missing", http.StatusBadRequest)
			return
		}

		if r.URL.Path == "/httpreq/present" && data.Value == "" {
			http.Error(w, "value is missing", http.StatusBadRequest)
			return
		}

		data.FQDN = strings.TrimRight(data.FQDN, ".")
		name, zone, err := SplitFQDN(data.FQDN)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		username, password, _ := r.BasicAuth()
		next.ServeHTTP(w, r.WithContext(
			model.NewContextWithReqData(r.Context(),
				&model.ReqData{
					FullName:  data.FQDN,
					Name:      name,
					Zone:      zone,
					Value:     data.Value,
					Type:      recordTypeTXT,
					Username:  username,
					Password:  password,
					BasicAuth: true,
				},
			)),
		)
	})
}

func BindDirectAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		if recordType != recordTypeA && recordType != recordTypeTXT {
			http.Error(w, "type can only be A or TXT", http.StatusBadRequest)
			return
		}

		fqdn := domain
		if name := r.Form.Get("name"); name != "" {
			fqdn = name + "." + domain
		}

		name, zone, err := SplitFQDN(fqdn)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		username, password, _ := r.BasicAuth()
		next.ServeHTTP(w, r.WithContext(
			model.NewContextWithReqData(r.Context(),
				&model.ReqData{
					FullName:  fqdn,
					Name:      name,
					Zone:      zone,
					Value:     r.Form.Get("value"),
					Type:      recordType,
					Username:  username,
					Password:  password,
					BasicAuth: true,
				},
			)),
		)
	})
}

func SplitFQDN(fqdn string) (name, zone string, err error) {
	parts := strings.Split(fqdn, ".")
	length := len(parts)

	const zoneParts = 2
	if length < zoneParts {
		return "", "", fmt.Errorf("invalid fqdn: %s", fqdn)
	}

	name = strings.Join(parts[:length-zoneParts], ".")
	zone = strings.Join(parts[length-zoneParts:], ".")

	return name, zone, nil
}

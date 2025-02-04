package update

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/data"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/hetzner"
)

const (
	headerAuthAPIToken = "Auth-API-Token" //#nosec G101
	requestTimeout     = 60
)

type Controller struct {
	cfg    *config.Config
	mutex  *sync.Mutex
	client *http.Client
}

func NewController(cfg *config.Config) *Controller {
	return &Controller{
		cfg,
		&sync.Mutex{},
		&http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		},
	}
}

func (d *Controller) CheckPermissions() gin.HandlerFunc {
	return func(c *gin.Context) {
		record := c.MustGet(data.KeyRecord).(*data.DNSRecord)

		if !CheckPermissions(record.FullName, c.ClientIP(), d.cfg.AllowedDomains) {
			log.Printf("Client '%s' is not allowed to update '%s' data of '%s' to '%s'\n", c.ClientIP(), record.Type, record.FullName, record.Value)
			c.AbortWithStatus(http.StatusForbidden)
		}
	}
}

func (d *Controller) UpdateDNS() gin.HandlerFunc {
	return func(c *gin.Context) {
		record := c.MustGet(data.KeyRecord).(*data.DNSRecord)
		log.Printf("Received request to update '%s' data of '%s' to '%s'\n", record.Type, record.FullName, record.Value)

		if err := d.do(record); err != nil {
			log.Printf("Update failed: %v", err)
			_ = c.AbortWithError(http.StatusInternalServerError, err)
		}
	}
}

func (d *Controller) do(record *data.DNSRecord) error {
	// Ensure only one simultaneous update sequence
	d.mutex.Lock()
	defer d.mutex.Unlock()

	zIDs, err := d.getZoneIds()
	if err != nil {
		return err
	}

	zID := zIDs[record.Zone]
	if zID == "" {
		return fmt.Errorf("could not find zone id for record %s", record.FullName)
	}

	rIDs, err := d.getRecordIds(zID, record.Type)
	if err != nil {
		return err
	}

	r := hetzner.Record{
		Name:   record.Name,
		TTL:    d.cfg.RecordTTL,
		Type:   record.Type,
		Value:  record.Value,
		ZoneID: zID,
	}

	if rID, ok := rIDs[record.Name]; ok {
		r.ID = rID
		return d.updateRecord(&r)
	}

	return d.createRecord(&r)
}

func (d *Controller) getRequest(url string) (body []byte, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Add(headerAuthAPIToken, d.cfg.Token)

	res, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil {
			err = closeErr
		}
	}()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get request failed with statuscode %d", res.StatusCode)
	}

	body, err = io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return
}

func (d *Controller) jsonRequest(method, url string, body []byte) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add(headerAuthAPIToken, d.cfg.Token)

	res, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil {
			err = closeErr
		}
	}()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%s request failed with statuscode %d", method, res.StatusCode)
	}

	return
}

func (d *Controller) getZoneIds() (map[string]string, error) {
	res, err := d.getRequest(d.cfg.BaseURL + "/zones")
	if err != nil {
		return nil, err
	}

	z := hetzner.Zones{}
	if err := json.Unmarshal(res, &z); err != nil {
		return nil, err
	}

	ids := map[string]string{}
	for _, zone := range z.Zones {
		ids[zone.Name] = zone.ID
	}

	return ids, nil
}

func (d *Controller) getRecordIds(zoneID, recordType string) (map[string]string, error) {
	res, err := d.getRequest(d.cfg.BaseURL + "/records?zone_id=" + zoneID)
	if err != nil {
		return nil, err
	}

	r := hetzner.Records{}
	if err := json.Unmarshal(res, &r); err != nil {
		return nil, err
	}

	ids := map[string]string{}
	for _, record := range r.Records {
		if record.Type == recordType {
			ids[record.Name] = record.ID
		}
	}

	return ids, nil
}

func (d *Controller) createRecord(record *hetzner.Record) error {
	body, err := json.Marshal(record)
	if err != nil {
		return err
	}

	return d.jsonRequest(http.MethodPost, d.cfg.BaseURL+"/records", body)
}

func (d *Controller) updateRecord(record *hetzner.Record) error {
	body, err := json.Marshal(record)
	if err != nil {
		return err
	}

	return d.jsonRequest(http.MethodPut, d.cfg.BaseURL+"/records/"+record.ID, body)
}

func CheckPermissions(fqdn, clientIP string, allowedDomains config.AllowedDomains) bool {
	for domain, ipNets := range allowedDomains {
		if fqdn != domain && !IsSubDomain(fqdn, domain) {
			continue
		}
		for _, ipNet := range ipNets {
			ip := net.ParseIP(clientIP)
			if ip != nil && ipNet.Contains(ip) {
				return true
			}
		}
	}

	return false
}

func IsSubDomain(sub, parent string) bool {
	subParts := strings.Split(sub, ".")
	parentParts := strings.Split(parent, ".")

	// Parent domain must be a wildcard domain
	// The subdomain must have at least the same amount of parts as the parent domain
	if parentParts[0] != "*" || len(subParts) < len(parentParts) {
		return false
	}

	// All domain parts up to the asterisk must match
	offset := len(subParts) - len(parentParts[1:])
	return slices.Equal(parentParts[1:], subParts[offset:])
}

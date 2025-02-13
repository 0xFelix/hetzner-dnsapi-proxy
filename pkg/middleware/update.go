package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/hetzner"
)

const (
	headerAuthAPIToken = "Auth-API-Token" //#nosec G101
	requestTimeout     = 60
	requestFailedFmt   = "%s request failed with status code %d"
)

type updater struct {
	cfg    *config.Config
	client http.Client
	m      sync.Mutex
}

func NewUpdater(cfg *config.Config) func(http.Handler) http.Handler {
	u := &updater{
		cfg: cfg,
		client: http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		},
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			data, err := reqDataFromContext(r.Context())
			if err != nil {
				log.Printf("%v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			log.Printf("received request to update '%s' data of '%s' to '%s'\n", data.Type, data.FullName, data.Value)
			if err := u.update(data); err != nil {
				log.Printf("failed to update record: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (u *updater) update(data *reqData) error {
	// Ensure only one simultaneous update sequence
	u.m.Lock()
	defer u.m.Unlock()

	zIDs, err := u.getZoneIds()
	if err != nil {
		return err
	}

	zID := zIDs[data.Zone]
	if zID == "" {
		return fmt.Errorf("could not find zone id for record %s", data.FullName)
	}

	rIDs, err := u.getRecordIds(zID, data.Type)
	if err != nil {
		return err
	}

	r := hetzner.Record{
		Name:   data.Name,
		TTL:    u.cfg.RecordTTL,
		Type:   data.Type,
		Value:  data.Value,
		ZoneID: zID,
	}

	if rID, ok := rIDs[data.Name]; ok {
		r.ID = rID
		return u.updateRecord(&r)
	}

	return u.createRecord(&r)
}

func (u *updater) getRequest(url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Add(headerAuthAPIToken, u.cfg.Token)

	res, err := u.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil {
			err = closeErr
		}
	}()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(requestFailedFmt, http.MethodGet, res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return body, err
}

func (u *updater) jsonRequest(method, url string, body []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Add(headerContentType, applicationJSON)
	req.Header.Add(headerAuthAPIToken, u.cfg.Token)

	res, err := u.client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil {
			err = closeErr
		}
	}()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf(requestFailedFmt, method, res.StatusCode)
	}

	return nil
}

func (u *updater) getZoneIds() (map[string]string, error) {
	res, err := u.getRequest(u.cfg.BaseURL + "/zones")
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

func (u *updater) getRecordIds(zoneID, recordType string) (map[string]string, error) {
	res, err := u.getRequest(u.cfg.BaseURL + "/records?zone_id=" + zoneID)
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

func (u *updater) createRecord(record *hetzner.Record) error {
	body, err := json.Marshal(record)
	if err != nil {
		return err
	}

	return u.jsonRequest(http.MethodPost, u.cfg.BaseURL+"/records", body)
}

func (u *updater) updateRecord(record *hetzner.Record) error {
	body, err := json.Marshal(record)
	if err != nil {
		return err
	}

	return u.jsonRequest(http.MethodPut, u.cfg.BaseURL+"/records/"+record.ID, body)
}

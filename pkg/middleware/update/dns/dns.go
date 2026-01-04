package dns

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/data"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/hetzner"
)

const (
	headerAuthAPIToken = "Auth-API-Token" //#nosec G101
	headerContentType  = "Content-Type"
	applicationJSON    = "application/json"
	requestFailedFmt   = "%s request failed with status code %d"
)

type updater struct {
	cfg    *config.Config
	client http.Client
	m      *sync.Mutex
}

func New(cfg *config.Config, m *sync.Mutex) *updater {
	return &updater{
		cfg: cfg,
		client: http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		},
		m: m,
	}
}

func (u *updater) Update(ctx context.Context, reqData *data.ReqData) error {
	// Ensure only one simultaneous update sequence
	u.m.Lock()
	defer u.m.Unlock()

	zIDs, err := u.getZoneIds(ctx)
	if err != nil {
		return err
	}

	zID := zIDs[reqData.Zone]
	if zID == "" {
		return fmt.Errorf("could not find zone id for record %s", reqData.FullName)
	}

	rIDs, err := u.getRecordIds(ctx, zID, reqData.Type)
	if err != nil {
		return err
	}

	r := hetzner.Record{
		Name:   reqData.Name,
		TTL:    u.cfg.RecordTTL,
		Type:   reqData.Type,
		Value:  reqData.Value,
		ZoneID: zID,
	}

	if rID, ok := rIDs[reqData.Name]; ok {
		r.ID = rID
		return u.updateRecord(ctx, &r)
	}

	return u.createRecord(ctx, &r)
}

func (u *updater) getRequest(ctx context.Context, url string) (body []byte, err error) {
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
		err = errors.Join(err, res.Body.Close())
	}()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(requestFailedFmt, http.MethodGet, res.StatusCode)
	}

	body, err = io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return body, err
}

func (u *updater) jsonRequest(ctx context.Context, method, url string, body []byte) (err error) {
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
		err = errors.Join(err, res.Body.Close())
	}()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf(requestFailedFmt, method, res.StatusCode)
	}

	return nil
}

func (u *updater) getZoneIds(ctx context.Context) (map[string]string, error) {
	res, err := u.getRequest(ctx, u.cfg.BaseURL+"/zones")
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

func (u *updater) getRecordIds(ctx context.Context, zoneID, recordType string) (map[string]string, error) {
	res, err := u.getRequest(ctx, u.cfg.BaseURL+"/records?zone_id="+zoneID)
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

func (u *updater) createRecord(ctx context.Context, record *hetzner.Record) error {
	body, err := json.Marshal(record)
	if err != nil {
		return err
	}

	return u.jsonRequest(ctx, http.MethodPost, u.cfg.BaseURL+"/records", body)
}

func (u *updater) updateRecord(ctx context.Context, record *hetzner.Record) error {
	body, err := json.Marshal(record)
	if err != nil {
		return err
	}

	return u.jsonRequest(ctx, http.MethodPut, u.cfg.BaseURL+"/records/"+record.ID, body)
}

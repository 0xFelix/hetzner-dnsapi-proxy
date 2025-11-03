package cloud

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/internal/model"
)

type updater struct {
	cfg    *config.Config
	client *hcloud.Client
	m      sync.Mutex
}

func New(cfg *config.Config) func(http.Handler) http.Handler {
	u := &updater{
		cfg:    cfg,
		client: hcloud.NewClient(hcloud.WithToken(cfg.Token)),
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			data, err := model.ReqDataFromContext(r.Context())
			if err != nil {
				log.Printf("%v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			log.Printf("received request to update '%s' data of '%s' to '%s'", data.Type, data.FullName, data.Value)
			if err := u.update(r.Context(), data); err != nil {
				log.Printf("failed to update record: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (u *updater) update(ctx context.Context, data *model.ReqData) error {
	// Ensure only one simultaneous update sequence
	u.m.Lock()
	defer u.m.Unlock()

	t := recordTypeFromString(data.Type)

	z, err := u.getZone(ctx, data.Zone)
	if err != nil {
		return fmt.Errorf("could not find zone id for record %s", data.FullName)
	}

	r, err := u.getRecord(ctx, z, data.Name, t)
	if err != nil {
		return err
	}

	if r != nil {
		r.TTL = &u.cfg.RecordTTL
		r.Records = []hcloud.ZoneRRSetRecord{{
			Value: strconv.Quote(data.Value),
		}}
		return u.updateRecord(ctx, r)
	}

	return u.createRecord(ctx, z, hcloud.ZoneRRSetCreateOpts{
		Name: data.Name,
		Type: t,
		TTL:  &u.cfg.RecordTTL,
		Records: []hcloud.ZoneRRSetRecord{{
			Value: strconv.Quote(data.Value),
		}},
	})
}

func (u *updater) getZone(ctx context.Context, zoneName string) (*hcloud.Zone, error) {
	zone, _, err := u.client.Zone.Get(ctx, zoneName)
	if err != nil {
		return nil, err
	}

	return zone, nil
}

func (u *updater) getRecord(
	ctx context.Context,
	zone *hcloud.Zone,
	recordName string,
	recordType hcloud.ZoneRRSetType,
) (*hcloud.ZoneRRSet, error) {
	record, _, err := u.client.Zone.GetRRSetByNameAndType(ctx, zone, recordName, recordType)
	if err != nil {
		return nil, err
	}

	return record, nil
}

//nolint:gocyclo // reason: recordTypeFromString is simple enough
func recordTypeFromString(recordType string) hcloud.ZoneRRSetType {
	switch recordType {
	case "A":
		return hcloud.ZoneRRSetTypeA
	case "AAAA":
		return hcloud.ZoneRRSetTypeAAAA
	case "CAA":
		return hcloud.ZoneRRSetTypeCAA
	case "CNAME":
		return hcloud.ZoneRRSetTypeCNAME
	case "DS":
		return hcloud.ZoneRRSetTypeDS
	case "HINFO":
		return hcloud.ZoneRRSetTypeHINFO
	case "HTTPS":
		return hcloud.ZoneRRSetTypeHTTPS
	case "MX":
		return hcloud.ZoneRRSetTypeMX
	case "NS":
		return hcloud.ZoneRRSetTypeNS
	case "PTR":
		return hcloud.ZoneRRSetTypePTR
	case "RP":
		return hcloud.ZoneRRSetTypeRP
	case "SOA":
		return hcloud.ZoneRRSetTypeSOA
	case "SRV":
		return hcloud.ZoneRRSetTypeSRV
	case "SVCB":
		return hcloud.ZoneRRSetTypeSVCB
	case "TLSA":
		return hcloud.ZoneRRSetTypeTLSA
	case "TXT":
		return hcloud.ZoneRRSetTypeTXT
	default:
		return ""
	}
}

func (u *updater) createRecord(ctx context.Context, zone *hcloud.Zone, record hcloud.ZoneRRSetCreateOpts) error {
	_, _, err := u.client.Zone.CreateRRSet(ctx, zone, record)
	return err
}

func (u *updater) updateRecord(ctx context.Context, record *hcloud.ZoneRRSet) error {
	_, _, err := u.client.Zone.UpdateRRSet(ctx, record, hcloud.ZoneRRSetUpdateOpts{})
	return err
}

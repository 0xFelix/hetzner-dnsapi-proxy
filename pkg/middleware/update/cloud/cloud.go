package cloud

import (
	"context"
	"strconv"
	"sync"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/data"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/hetzner"
)

type updater struct {
	cfg    *config.Config
	client *hcloud.Client
	m      *sync.Mutex
}

func New(cfg *config.Config, m *sync.Mutex) *updater {
	return &updater{
		cfg:    cfg,
		client: hcloud.NewClient(hcloud.WithToken(cfg.Token)),
		m:      m,
	}
}

func (u *updater) Update(ctx context.Context, reqData *data.ReqData) error {
	// Ensure only one simultaneous update sequence
	u.m.Lock()
	defer u.m.Unlock()

	rrSetType, err := hetzner.RRSetTypeFromString(reqData.Type)
	if err != nil {
		return err
	}

	zone, _, err := u.client.Zone.Get(ctx, reqData.Zone)
	if err != nil {
		return err
	}

	rrSet, _, err := u.client.Zone.GetRRSetByNameAndType(ctx, zone, reqData.Name, rrSetType)
	if err != nil {
		return err
	}

	if rrSet != nil {
		return u.updateRRSet(ctx, rrSet, reqData.Value)
	}

	return u.createRRSet(ctx, zone, rrSetType, reqData.Name, reqData.Value)
}

func (u *updater) updateRRSet(ctx context.Context, rrSet *hcloud.ZoneRRSet, val string) error {
	rrSet.TTL = &u.cfg.RecordTTL
	rrSet.Records = []hcloud.ZoneRRSetRecord{{
		Value: strconv.Quote(val),
	}}
	_, _, err := u.client.Zone.UpdateRRSet(ctx, rrSet, hcloud.ZoneRRSetUpdateOpts{})
	return err
}

func (u *updater) createRRSet(ctx context.Context, zone *hcloud.Zone, rrSetType hcloud.ZoneRRSetType, name, val string) error {
	opts := hcloud.ZoneRRSetCreateOpts{
		Name: name,
		Type: rrSetType,
		TTL:  &u.cfg.RecordTTL,
		Records: []hcloud.ZoneRRSetRecord{{
			Value: strconv.Quote(val),
		}},
	}
	_, _, err := u.client.Zone.CreateRRSet(ctx, zone, opts)
	return err
}

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
		client: hetzner.NewHCloudClient(cfg),
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
	if rrSet.TTL == nil || *rrSet.TTL != u.cfg.RecordTTL {
		opts := hcloud.ZoneRRSetChangeTTLOpts{TTL: &u.cfg.RecordTTL}
		action, _, err := u.client.Zone.ChangeRRSetTTL(ctx, rrSet, opts)
		if err != nil {
			return err
		}
		if action != nil {
			if err := u.client.Action.WaitFor(ctx, action); err != nil {
				return err
			}
		}
	}

	opts := hcloud.ZoneRRSetSetRecordsOpts{
		Records: []hcloud.ZoneRRSetRecord{{
			Value: quoteIfRequired(val, rrSet.Type),
		}},
	}
	action, _, err := u.client.Zone.SetRRSetRecords(ctx, rrSet, opts)
	if err != nil {
		return err
	}
	if action != nil {
		return u.client.Action.WaitFor(ctx, action)
	}

	return nil
}

func (u *updater) createRRSet(ctx context.Context, zone *hcloud.Zone, rrSetType hcloud.ZoneRRSetType, name, val string) error {
	opts := hcloud.ZoneRRSetCreateOpts{
		Name: name,
		Type: rrSetType,
		TTL:  &u.cfg.RecordTTL,
		Records: []hcloud.ZoneRRSetRecord{{
			Value: quoteIfRequired(val, rrSetType),
		}},
	}
	result, _, err := u.client.Zone.CreateRRSet(ctx, zone, opts)
	if err != nil {
		return err
	}
	if result.Action != nil {
		return u.client.Action.WaitFor(ctx, result.Action)
	}

	return nil
}

func quoteIfRequired(val string, rrSetType hcloud.ZoneRRSetType) string {
	if rrSetType == hcloud.ZoneRRSetTypeTXT {
		return strconv.Quote(val)
	}
	return val
}

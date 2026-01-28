package cloud

import (
	"context"
	"sync"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/data"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/hetzner"
)

type cleaner struct {
	cfg    *config.Config
	client *hcloud.Client
	m      *sync.Mutex
}

func New(cfg *config.Config, m *sync.Mutex) *cleaner {
	return &cleaner{
		cfg:    cfg,
		client: hetzner.NewHCloudClient(cfg),
		m:      m,
	}
}

func (u *cleaner) Clean(ctx context.Context, reqData *data.ReqData) error {
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

	action, _, err := u.client.Zone.RemoveRRSetRecords(ctx, rrSet, hcloud.ZoneRRSetRemoveRecordsOpts{
		Records: rrSet.Records,
	})
	if err != nil {
		return err
	}
	if action != nil {
		return u.client.Action.WaitFor(ctx, action)
	}

	return nil
}

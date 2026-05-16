package cloud

import (
	"context"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/data"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/hetzner"
)

type cleaner struct {
	cfg    *config.Config
	client *hcloud.Client
}

func New(cfg *config.Config) *cleaner {
	return &cleaner{
		cfg:    cfg,
		client: hetzner.NewHCloudClient(cfg),
	}
}

func (u *cleaner) Clean(ctx context.Context, reqData *data.ReqData) error {
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
		Records: []hcloud.ZoneRRSetRecord{{Value: hetzner.QuoteIfRequired(reqData.Value, rrSetType)}},
	})
	if err != nil {
		return err
	}
	if action != nil {
		return u.client.Action.WaitFor(ctx, action)
	}

	return nil
}

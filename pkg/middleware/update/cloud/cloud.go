package cloud

import (
	"context"
	"errors"
	"fmt"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/data"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/hetzner"
)

type updater struct {
	cfg    *config.Config
	client *hcloud.Client
}

func New(cfg *config.Config) *updater {
	return &updater{
		cfg:    cfg,
		client: hetzner.NewHCloudClient(cfg),
	}
}

func (u *updater) Update(ctx context.Context, reqData *data.ReqData) error {
	rrSetType, err := hetzner.RRSetTypeFromString(reqData.Type)
	if err != nil {
		return err
	}

	zone, _, err := u.client.Zone.Get(ctx, reqData.Zone)
	if err != nil {
		return err
	}

	name := hetzner.RRSetName(reqData.Name)
	rrSet, _, err := u.client.Zone.GetRRSetByNameAndType(ctx, zone, name, rrSetType)
	if err != nil {
		return err
	}

	if rrSet != nil {
		return u.applyToRRSet(ctx, rrSet, reqData)
	}

	if reqData.CheckConflicts {
		if conflictErr := u.checkConflicts(ctx, zone, rrSetType, name); conflictErr != nil {
			return conflictErr
		}
	}

	err = u.createRRSet(ctx, zone, rrSetType, name, reqData.Value)
	if err == nil || ctx.Err() != nil {
		return err
	}

	// A concurrent request may have created the RRSet, or a conflicting one,
	// in the meantime. The API error for this is not relied upon, so check
	// for both after any failed create.
	rrSet, _, getErr := u.client.Zone.GetRRSetByNameAndType(ctx, zone, name, rrSetType)
	if getErr != nil {
		return err
	}
	if rrSet != nil {
		return u.applyToRRSet(ctx, rrSet, reqData)
	}
	if reqData.CheckConflicts {
		if conflictErr := u.checkConflicts(ctx, zone, rrSetType, name); errors.Is(conflictErr, data.ErrConflict) {
			return conflictErr
		}
	}

	return err
}

// applyToRRSet applies the requested value to an existing RRSet according to reqData.Mode.
func (u *updater) applyToRRSet(ctx context.Context, rrSet *hcloud.ZoneRRSet, reqData *data.ReqData) error {
	// A '.' SRV target (service not available) must be the sole record, so don't append to it.
	if reqData.Mode == data.ModeAppend && !hetzner.HasSRVUnavailable(rrSet) {
		return u.addRRSetRecord(ctx, rrSet, reqData.Value)
	}
	return u.setRRSetRecords(ctx, rrSet, reqData.Value)
}

// checkConflicts fails with data.ErrConflict if creating an RRSet of the given
// type would clash with an existing one: a CNAME cannot coexist with any other
// type on the same name.
func (u *updater) checkConflicts(ctx context.Context, zone *hcloud.Zone, rrSetType hcloud.ZoneRRSetType, name string) error {
	rrSets, err := u.client.Zone.AllRRSetsWithOpts(ctx, zone, hcloud.ZoneRRSetListOpts{Name: name})
	if err != nil {
		return err
	}

	for _, rrSet := range rrSets {
		if rrSet.Name != name || rrSet.Type == rrSetType {
			continue
		}
		if rrSetType == hcloud.ZoneRRSetTypeCNAME || rrSet.Type == hcloud.ZoneRRSetTypeCNAME {
			return fmt.Errorf("%w: cannot create %s record set, %s record set exists for %q",
				data.ErrConflict, rrSetType, rrSet.Type, name)
		}
	}

	return nil
}

func (u *updater) changeTTL(ctx context.Context, rrSet *hcloud.ZoneRRSet) error {
	if rrSet.TTL != nil && *rrSet.TTL == u.cfg.RecordTTL {
		return nil
	}

	opts := hcloud.ZoneRRSetChangeTTLOpts{TTL: &u.cfg.RecordTTL}
	action, _, err := u.client.Zone.ChangeRRSetTTL(ctx, rrSet, opts)
	return hetzner.WaitForAction(ctx, u.client, action, err)
}

func (u *updater) setRRSetRecords(ctx context.Context, rrSet *hcloud.ZoneRRSet, val string) error {
	if err := u.changeTTL(ctx, rrSet); err != nil {
		return err
	}

	opts := hcloud.ZoneRRSetSetRecordsOpts{
		Records: []hcloud.ZoneRRSetRecord{{
			Value: hetzner.QuoteIfRequired(val, rrSet.Type),
		}},
	}
	action, _, err := u.client.Zone.SetRRSetRecords(ctx, rrSet, opts)
	return hetzner.WaitForAction(ctx, u.client, action, err)
}

func (u *updater) addRRSetRecord(ctx context.Context, rrSet *hcloud.ZoneRRSet, val string) error {
	if err := u.changeTTL(ctx, rrSet); err != nil {
		return err
	}

	val = hetzner.QuoteIfRequired(val, rrSet.Type)
	if _, found := hetzner.FindRecord(rrSet, val); found {
		return nil
	}

	opts := hcloud.ZoneRRSetAddRecordsOpts{
		Records: []hcloud.ZoneRRSetRecord{{Value: val}},
	}
	action, _, err := u.client.Zone.AddRRSetRecords(ctx, rrSet, opts)
	return hetzner.WaitForAction(ctx, u.client, action, err)
}

func (u *updater) createRRSet(ctx context.Context, zone *hcloud.Zone, rrSetType hcloud.ZoneRRSetType, name, val string) error {
	opts := hcloud.ZoneRRSetCreateOpts{
		Name: name,
		Type: rrSetType,
		TTL:  &u.cfg.RecordTTL,
		Records: []hcloud.ZoneRRSetRecord{{
			Value: hetzner.QuoteIfRequired(val, rrSetType),
		}},
	}
	result, _, err := u.client.Zone.CreateRRSet(ctx, zone, opts)
	return hetzner.WaitForAction(ctx, u.client, result.Action, err)
}

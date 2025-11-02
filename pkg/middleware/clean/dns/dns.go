package dns

import (
	"context"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/data"
)

type cleaner struct{}

func New() *cleaner {
	return &cleaner{}
}

func (u *cleaner) Clean(_ context.Context, _ *data.ReqData) error {
	return nil
}

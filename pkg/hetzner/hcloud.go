package hetzner

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
)

func NewHCloudClient(cfg *config.Config) *hcloud.Client {
	version := "dev"
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				version = setting.Value
				break
			}
		}
	}

	opts := []hcloud.ClientOption{
		hcloud.WithToken(cfg.Token),
		hcloud.WithHTTPClient(&http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		}),
		hcloud.WithApplication("hetzner-dnsapi-proxy", version),
		hcloud.WithEndpoint(cfg.BaseURL),
	}

	return hcloud.NewClient(opts...)
}

func RRSetTypeFromString(rType string) (hcloud.ZoneRRSetType, error) {
	switch rrType := hcloud.ZoneRRSetType(rType); rrType {
	case hcloud.ZoneRRSetTypeA,
		hcloud.ZoneRRSetTypeAAAA,
		hcloud.ZoneRRSetTypeTXT:
		return rrType, nil
	case hcloud.ZoneRRSetTypeCAA,
		hcloud.ZoneRRSetTypeCNAME,
		hcloud.ZoneRRSetTypeDS,
		hcloud.ZoneRRSetTypeHINFO,
		hcloud.ZoneRRSetTypeHTTPS,
		hcloud.ZoneRRSetTypeMX,
		hcloud.ZoneRRSetTypeNS,
		hcloud.ZoneRRSetTypePTR,
		hcloud.ZoneRRSetTypeRP,
		hcloud.ZoneRRSetTypeSOA,
		hcloud.ZoneRRSetTypeSRV,
		hcloud.ZoneRRSetTypeSVCB,
		hcloud.ZoneRRSetTypeTLSA:
		return "", fmt.Errorf("unsupported resource record set type %s", rrType)
	default:
		return "", fmt.Errorf("unrecognized resource record set type %s", rType)
	}
}

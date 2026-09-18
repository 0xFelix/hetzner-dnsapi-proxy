package hetzner

import (
	"fmt"
	"runtime/debug"
	"strconv"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
)

// version is set at build time via -ldflags -X.
var version string

func NewHCloudClient(cfg *config.Config) *hcloud.Client {
	opts := []hcloud.ClientOption{
		hcloud.WithToken(cfg.Token),
		hcloud.WithApplication("hetzner-dnsapi-proxy", appVersion()),
		hcloud.WithEndpoint(cfg.BaseURL),
	}

	return hcloud.NewClient(opts...)
}

func appVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				return setting.Value
			}
		}
	}
	return "dev"
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

func QuoteIfRequired(val string, rrSetType hcloud.ZoneRRSetType) string {
	if rrSetType == hcloud.ZoneRRSetTypeTXT {
		return strconv.Quote(val)
	}
	return val
}

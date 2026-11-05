package hetzner

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
)

const asciiDEL = 0x7f

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

// RRSetName returns the name the API expects for a record name that is
// relative to its zone. An empty name refers to the zone apex, which the
// API represents as '@'.
func RRSetName(name string) string {
	if name == "" {
		return "@"
	}
	return name
}

func RRSetTypeFromString(rType string) (hcloud.ZoneRRSetType, error) {
	switch rrType := hcloud.ZoneRRSetType(rType); rrType {
	case hcloud.ZoneRRSetTypeA,
		hcloud.ZoneRRSetTypeAAAA,
		hcloud.ZoneRRSetTypeCNAME,
		hcloud.ZoneRRSetTypeSRV,
		hcloud.ZoneRRSetTypeTXT:
		return rrType, nil
	case hcloud.ZoneRRSetTypeCAA,
		hcloud.ZoneRRSetTypeDS,
		hcloud.ZoneRRSetTypeHINFO,
		hcloud.ZoneRRSetTypeHTTPS,
		hcloud.ZoneRRSetTypeMX,
		hcloud.ZoneRRSetTypeNS,
		hcloud.ZoneRRSetTypePTR,
		hcloud.ZoneRRSetTypeRP,
		hcloud.ZoneRRSetTypeSOA,
		hcloud.ZoneRRSetTypeSVCB,
		hcloud.ZoneRRSetTypeTLSA:
		return "", fmt.Errorf("unsupported resource record set type %s", rrType)
	default:
		return "", fmt.Errorf("unrecognized resource record set type %s", rType)
	}
}

// QuoteIfRequired returns TXT values as a quoted character string in zone
// file format (RFC 1035): '"' and '\' are escaped with a backslash and ASCII
// control characters as '\DDD'. Values of other types are returned unchanged.
func QuoteIfRequired(val string, rrSetType hcloud.ZoneRRSetType) string {
	if rrSetType != hcloud.ZoneRRSetTypeTXT {
		return val
	}

	var b strings.Builder
	b.WriteByte('"')
	for i := range len(val) {
		switch c := val[i]; {
		case c == '"' || c == '\\':
			b.WriteByte('\\')
			b.WriteByte(c)
		case c < ' ' || c == asciiDEL:
			fmt.Fprintf(&b, "\\%03d", c)
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// WaitForAction returns err if it is set and otherwise waits for action to
// complete, if there is one.
func WaitForAction(ctx context.Context, client *hcloud.Client, action *hcloud.Action, err error) error {
	if err != nil {
		return err
	}
	if action == nil {
		return nil
	}
	return client.Action.WaitFor(ctx, action)
}

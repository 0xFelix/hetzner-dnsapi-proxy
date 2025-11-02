package hetzner

import (
	"fmt"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

const RequestTimeout = 60 * time.Second

type Zone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Zones struct {
	Zones []Zone `json:"zones"`
}

type Record struct {
	ID     string `json:"id,omitempty"`
	Name   string `json:"name"`
	TTL    int    `json:"ttl"`
	Type   string `json:"type"`
	Value  string `json:"value"`
	ZoneID string `json:"zone_id"`
}

type Records struct {
	Records []Record `json:"records"`
}

func RRSetTypeFromString(rType string) (hcloud.ZoneRRSetType, error) {
	switch rrType := hcloud.ZoneRRSetType(rType); rrType {
	case hcloud.ZoneRRSetTypeA,
		hcloud.ZoneRRSetTypeTXT:
		return rrType, nil
	case hcloud.ZoneRRSetTypeAAAA,
		hcloud.ZoneRRSetTypeCAA,
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

package libdnsapi

import (
	"net/http"

	"github.com/onsi/gomega/ghttp"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/hetzner"
	"github.com/0xfelix/hetzner-dnsapi-proxy/tests/libserver"
)

const (
	headerAuthAPIToken = "Auth-API-Token" //#nosec G101
)

func Zones() []hetzner.Zone {
	return []hetzner.Zone{
		{
			ID:   libserver.ZoneID,
			Name: libserver.ZoneName,
		},
	}
}

func Records() []hetzner.Record {
	return []hetzner.Record{
		{
			ID:     libserver.ARecordID,
			Name:   libserver.ARecordName,
			TTL:    libserver.DefaultTTL,
			Type:   libserver.RecordTypeA,
			Value:  libserver.AExisting,
			ZoneID: libserver.ZoneID,
		},
		{
			ID:     libserver.AAAARecordID,
			Name:   libserver.AAAARecordName,
			TTL:    libserver.DefaultTTL,
			Type:   libserver.RecordTypeAAAA,
			Value:  libserver.AAAAExisting,
			ZoneID: libserver.ZoneID,
		},
		{
			ID:     libserver.TXTRecordID,
			Name:   libserver.TXTRecordName,
			TTL:    libserver.DefaultTTL,
			Type:   libserver.RecordTypeTXT,
			Value:  libserver.TXTExisting,
			ZoneID: libserver.ZoneID,
		},
	}
}

func NewARecord() hetzner.Record {
	return hetzner.Record{
		Name:   libserver.ARecordName,
		TTL:    libserver.DefaultTTL,
		Type:   libserver.RecordTypeA,
		Value:  libserver.AUpdated,
		ZoneID: libserver.ZoneID,
	}
}

func UpdatedARecord() hetzner.Record {
	return hetzner.Record{
		ID:     libserver.ARecordID,
		Name:   libserver.ARecordName,
		TTL:    libserver.DefaultTTL,
		Type:   libserver.RecordTypeA,
		Value:  libserver.AUpdated,
		ZoneID: libserver.ZoneID,
	}
}

func NewAAAARecord() hetzner.Record {
	return hetzner.Record{
		Name:   libserver.AAAARecordName,
		TTL:    libserver.DefaultTTL,
		Type:   libserver.RecordTypeAAAA,
		Value:  libserver.AAAAUpdated,
		ZoneID: libserver.ZoneID,
	}
}

func UpdatedAAAARecord() hetzner.Record {
	return hetzner.Record{
		ID:     libserver.AAAARecordID,
		Name:   libserver.AAAARecordName,
		TTL:    libserver.DefaultTTL,
		Type:   libserver.RecordTypeAAAA,
		Value:  libserver.AAAAUpdated,
		ZoneID: libserver.ZoneID,
	}
}

func NewTXTRecord() hetzner.Record {
	return hetzner.Record{
		Name:   libserver.TXTRecordName,
		TTL:    libserver.DefaultTTL,
		Type:   libserver.RecordTypeTXT,
		Value:  libserver.TXTUpdated,
		ZoneID: libserver.ZoneID,
	}
}

func UpdatedTXTRecord() hetzner.Record {
	return hetzner.Record{
		ID:     libserver.TXTRecordID,
		Name:   libserver.TXTRecordName,
		TTL:    libserver.DefaultTTL,
		Type:   libserver.RecordTypeTXT,
		Value:  libserver.TXTUpdated,
		ZoneID: libserver.ZoneID,
	}
}

func GetZones(token string, zones []hetzner.Zone) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest(http.MethodGet, "/v1/zones"),
		ghttp.VerifyHeader(http.Header{
			headerAuthAPIToken: []string{token},
		}),
		ghttp.RespondWithJSONEncoded(http.StatusOK, hetzner.Zones{
			Zones: zones,
		}),
	)
}

func GetRecords(token, zoneID string, records []hetzner.Record) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest(http.MethodGet, "/v1/records", "zone_id="+zoneID),
		ghttp.VerifyHeader(http.Header{
			headerAuthAPIToken: []string{token},
		}),
		ghttp.RespondWithJSONEncoded(http.StatusOK, hetzner.Records{
			Records: records,
		}),
	)
}

func PostRecord(token string, record hetzner.Record) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest(http.MethodPost, "/v1/records"),
		ghttp.VerifyHeader(http.Header{
			headerAuthAPIToken: []string{token},
		}),
		ghttp.VerifyJSONRepresenting(record),
	)
}

func PutRecord(token string, record hetzner.Record) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest(http.MethodPut, "/v1/records/"+record.ID),
		ghttp.VerifyHeader(http.Header{
			headerAuthAPIToken: []string{token},
		}),
		ghttp.VerifyJSONRepresenting(record),
	)
}

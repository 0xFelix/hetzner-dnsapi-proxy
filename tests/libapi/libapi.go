package libapi

import (
	"net/http"

	"github.com/onsi/gomega/ghttp"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/hetzner"
)

const (
	TLD                   = "tld"
	ZoneName              = "test.tld"
	ARecordName           = "asub"
	ARecordNameFull       = "asub.test.tld"
	TXTRecordNameNoPrefix = "txtsub.test.tld"
	TXTRecordName         = "_acme-challenge.txtsub"
	TXTRecordNameFull     = "_acme-challenge.txtsub.test.tld"
	DefaultTTL            = 60
	AExisting             = "127.0.0.1"
	AUpdated              = "1.2.3.4"
	TXTExisting           = "randomvalue"
	TXTUpdated            = "changedrandomvalue"
	RecordTypeA           = "A"
	RecordTypeTXT         = "TXT"

	ZoneID      = "1"
	aRecordID   = "1"
	txtRecordID = "2"

	headerAuthAPIToken = "Auth-API-Token" //#nosec G101
)

func Zones() []hetzner.Zone {
	return []hetzner.Zone{
		{
			ID:   ZoneID,
			Name: ZoneName,
		},
	}
}

func Records() []hetzner.Record {
	return []hetzner.Record{
		{
			ID:     aRecordID,
			Name:   ARecordName,
			TTL:    DefaultTTL,
			Type:   RecordTypeA,
			Value:  AExisting,
			ZoneID: ZoneID,
		},
		{
			ID:     txtRecordID,
			Name:   TXTRecordName,
			TTL:    DefaultTTL,
			Type:   RecordTypeTXT,
			Value:  TXTExisting,
			ZoneID: ZoneID,
		},
	}
}

func NewARecord() hetzner.Record {
	return hetzner.Record{
		Name:   ARecordName,
		TTL:    DefaultTTL,
		Type:   RecordTypeA,
		Value:  AUpdated,
		ZoneID: ZoneID,
	}
}

func UpdatedARecord() hetzner.Record {
	return hetzner.Record{
		ID:     aRecordID,
		Name:   ARecordName,
		TTL:    DefaultTTL,
		Type:   RecordTypeA,
		Value:  AUpdated,
		ZoneID: ZoneID,
	}
}

func NewTXTRecord() hetzner.Record {
	return hetzner.Record{
		Name:   TXTRecordName,
		TTL:    DefaultTTL,
		Type:   RecordTypeTXT,
		Value:  TXTUpdated,
		ZoneID: ZoneID,
	}
}

func UpdatedTXTRecord() hetzner.Record {
	return hetzner.Record{
		ID:     txtRecordID,
		Name:   TXTRecordName,
		TTL:    DefaultTTL,
		Type:   RecordTypeTXT,
		Value:  TXTUpdated,
		ZoneID: ZoneID,
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

package libcloudapi

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"
	"github.com/onsi/gomega/ghttp"

	"github.com/0xfelix/hetzner-dnsapi-proxy/tests/libserver"
)

func Ptr[T any](v T) *T {
	return &v
}

func Zone() schema.Zone {
	return schema.Zone{
		ID:   MustParseInt(libserver.ZoneID),
		Name: libserver.ZoneName,
	}
}

func Records() []schema.ZoneRRSet {
	const ttl = 300
	return []schema.ZoneRRSet{
		{
			ID:   libserver.ARecordName + "/" + libserver.RecordTypeA,
			Name: libserver.ARecordName,
			Type: libserver.RecordTypeA,
			TTL:  Ptr(ttl),
			Records: []schema.ZoneRRSetRecord{
				{Value: strconv.Quote(libserver.AExisting)},
			},
			Zone: MustParseInt(libserver.ZoneID),
		},
		{
			ID:   libserver.TXTRecordName + "/" + libserver.RecordTypeTXT,
			Name: libserver.TXTRecordName,
			Type: libserver.RecordTypeTXT,
			TTL:  Ptr(ttl),
			Records: []schema.ZoneRRSetRecord{
				{Value: strconv.Quote(libserver.TXTExisting)},
			},
			Zone: MustParseInt(libserver.ZoneID),
		},
	}
}

func NewARecord() schema.ZoneRRSet {
	return schema.ZoneRRSet{
		Name: libserver.ARecordName,
		Type: libserver.RecordTypeA,
		TTL:  Ptr(libserver.DefaultTTL),
		Records: []schema.ZoneRRSetRecord{
			{Value: strconv.Quote(libserver.AUpdated)},
		},
		Zone: MustParseInt(libserver.ZoneID),
	}
}

func ExistingARecord() schema.ZoneRRSet {
	r := NewARecord()
	r.ID = libserver.ARecordName + "/" + libserver.RecordTypeA
	return r
}

func NewTXTRecord() schema.ZoneRRSet {
	return schema.ZoneRRSet{
		Name: libserver.TXTRecordName,
		Type: libserver.RecordTypeTXT,
		TTL:  Ptr(libserver.DefaultTTL),
		Records: []schema.ZoneRRSetRecord{
			{Value: strconv.Quote(libserver.TXTUpdated)},
		},
		Zone: MustParseInt(libserver.ZoneID),
	}
}

func ExistingTXTRecord() schema.ZoneRRSet {
	r := NewTXTRecord()
	r.ID = libserver.TXTRecordName + "/" + libserver.RecordTypeTXT
	return r
}

func GetZone(token string, zone schema.Zone) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest(http.MethodGet, "/v1/zones/"+zone.Name),
		ghttp.VerifyHeader(http.Header{
			"Authorization": []string{"Bearer " + token},
		}),
		ghttp.RespondWithJSONEncoded(http.StatusOK, schema.ZoneGetResponse{
			Zone: zone,
		}),
	)
}

func GetRRSet(token string, zone schema.Zone, rrSet schema.ZoneRRSet) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest(http.MethodGet, fmt.Sprintf("/v1/zones/%d/rrsets/%s/%s", zone.ID, rrSet.Name, rrSet.Type)),
		ghttp.VerifyHeader(http.Header{
			"Authorization": []string{"Bearer " + token},
		}),
		ghttp.RespondWithJSONEncoded(http.StatusOK, schema.ZoneRRSetGetResponse{
			RRSet: rrSet,
		}),
	)
}

func GetRRSetNotFound(token string, zone schema.Zone, name, rType string) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest(http.MethodGet, fmt.Sprintf("/v1/zones/%d/rrsets/%s/%s", zone.ID, name, rType)),
		ghttp.VerifyHeader(http.Header{
			"Authorization": []string{"Bearer " + token},
		}),
		ghttp.RespondWithJSONEncoded(http.StatusNotFound, schema.ErrorResponse{
			Error: schema.Error{
				Code:    "not_found",
				Message: "rrset not found",
			},
		}),
	)
}

func CreateRRSet(token string, zone schema.Zone, rrSet schema.ZoneRRSet) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest(http.MethodPost, fmt.Sprintf("/v1/zones/%d/rrsets", zone.ID)),
		ghttp.VerifyHeader(http.Header{
			"Authorization": []string{"Bearer " + token},
		}),
		ghttp.VerifyJSONRepresenting(schema.ZoneRRSetCreateRequest{
			Name:    rrSet.Name,
			Type:    string(hcloud.ZoneRRSetType(rrSet.Type)),
			TTL:     rrSet.TTL,
			Records: rrSet.Records,
		}),
		ghttp.RespondWithJSONEncoded(http.StatusCreated, schema.ZoneRRSetCreateResponse{
			RRSet: rrSet,
		}),
	)
}

func ChangeRRSetTTL(token string, zone schema.Zone, rrSet schema.ZoneRRSet) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest(http.MethodPost, fmt.Sprintf("/v1/zones/%d/rrsets/%s/%s/actions/change_ttl", zone.ID, rrSet.Name, rrSet.Type)),
		ghttp.VerifyHeader(http.Header{
			"Authorization": []string{"Bearer " + token},
		}),
		ghttp.VerifyJSONRepresenting(schema.ZoneRRSetChangeTTLRequest{
			TTL: rrSet.TTL,
		}),
		ghttp.RespondWithJSONEncoded(http.StatusOK, schema.ActionGetResponse{
			Action: schema.Action{
				ID:     1,
				Status: "success",
			},
		}),
	)
}

func SetRRSetRecords(token string, zone schema.Zone, rrSet schema.ZoneRRSet) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest(http.MethodPost, fmt.Sprintf("/v1/zones/%d/rrsets/%s/%s/actions/set_records", zone.ID, rrSet.Name, rrSet.Type)),
		ghttp.VerifyHeader(http.Header{
			"Authorization": []string{"Bearer " + token},
		}),
		ghttp.VerifyJSONRepresenting(schema.ZoneRRSetSetRecordsRequest{
			Records: rrSet.Records,
		}),
		ghttp.RespondWithJSONEncoded(http.StatusOK, schema.ActionGetResponse{
			Action: schema.Action{
				ID:     1,
				Status: "success",
			},
		}),
	)
}

func RemoveRRSetRecords(token string, zone schema.Zone, rrSet schema.ZoneRRSet) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest(http.MethodPost, fmt.Sprintf("/v1/zones/%d/rrsets/%s/%s/actions/remove_records", zone.ID, rrSet.Name, rrSet.Type)),
		ghttp.VerifyHeader(http.Header{
			"Authorization": []string{"Bearer " + token},
		}),
		ghttp.VerifyJSONRepresenting(schema.ZoneRRSetRemoveRecordsRequest{
			Records: rrSet.Records, // Simplified: assume we remove what we expect
		}),
		ghttp.RespondWithJSONEncoded(http.StatusOK, schema.ActionGetResponse{
			Action: schema.Action{
				ID:     1,
				Status: "success",
			},
		}),
	)
}

func MustParseInt(s string) int64 {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		panic(err)
	}
	return i
}

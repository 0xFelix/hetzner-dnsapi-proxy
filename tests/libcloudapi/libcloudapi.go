package libcloudapi

import (
	"fmt"
	"net/http"
	"strconv"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"

	"github.com/0xfelix/hetzner-dnsapi-proxy/tests/libserver"
)

const (
	headerAuthorization = "Authorization"
	authBearerPrefix    = "Bearer "
	existingTTL         = 300
)

func Zone() schema.Zone {
	return schema.Zone{
		ID:   mustParseInt(libserver.ZoneID),
		Name: libserver.ZoneName,
	}
}

func ExistingRRSetA() schema.ZoneRRSet {
	return schema.ZoneRRSet{
		ID:   libserver.ARecordName + "/" + libserver.RecordTypeA,
		Name: libserver.ARecordName,
		Type: libserver.RecordTypeA,
		TTL:  ptr(existingTTL),
		Records: []schema.ZoneRRSetRecord{
			{Value: libserver.AExisting},
		},
		Zone: mustParseInt(libserver.ZoneID),
	}
}

func ExistingRRSetAAAA() schema.ZoneRRSet {
	return schema.ZoneRRSet{
		ID:   libserver.AAAARecordName + "/" + libserver.RecordTypeAAAA,
		Name: libserver.AAAARecordName,
		Type: libserver.RecordTypeAAAA,
		TTL:  ptr(existingTTL),
		Records: []schema.ZoneRRSetRecord{
			{Value: libserver.AAAAExisting},
		},
		Zone: mustParseInt(libserver.ZoneID),
	}
}

func ExistingRRSetTXT() schema.ZoneRRSet {
	return schema.ZoneRRSet{
		ID:   libserver.TXTRecordName + "/" + libserver.RecordTypeTXT,
		Name: libserver.TXTRecordName,
		Type: libserver.RecordTypeTXT,
		TTL:  ptr(existingTTL),
		Records: []schema.ZoneRRSetRecord{
			{Value: strconv.Quote(libserver.TXTExisting)},
		},
		Zone: mustParseInt(libserver.ZoneID),
	}
}

func NewRRSetA() schema.ZoneRRSet {
	return schema.ZoneRRSet{
		Name: libserver.ARecordName,
		Type: libserver.RecordTypeA,
		TTL:  ptr(libserver.DefaultTTL),
		Records: []schema.ZoneRRSetRecord{
			{Value: libserver.AUpdated},
		},
		Zone: mustParseInt(libserver.ZoneID),
	}
}

func UpdatedRRSetA() schema.ZoneRRSet {
	r := NewRRSetA()
	r.ID = libserver.ARecordName + "/" + libserver.RecordTypeA
	return r
}

func NewRRSetAAAA() schema.ZoneRRSet {
	return schema.ZoneRRSet{
		Name: libserver.AAAARecordName,
		Type: libserver.RecordTypeAAAA,
		TTL:  ptr(libserver.DefaultTTL),
		Records: []schema.ZoneRRSetRecord{
			{Value: libserver.AAAAUpdated},
		},
		Zone: mustParseInt(libserver.ZoneID),
	}
}

func UpdatedRRSetAAAA() schema.ZoneRRSet {
	r := NewRRSetAAAA()
	r.ID = libserver.AAAARecordName + "/" + libserver.RecordTypeAAAA
	return r
}

func NewRRSetTXT() schema.ZoneRRSet {
	return schema.ZoneRRSet{
		Name: libserver.TXTRecordName,
		Type: libserver.RecordTypeTXT,
		TTL:  ptr(libserver.DefaultTTL),
		Records: []schema.ZoneRRSetRecord{
			{Value: strconv.Quote(libserver.TXTUpdated)},
		},
		Zone: mustParseInt(libserver.ZoneID),
	}
}

func UpdatedRRSetTXT() schema.ZoneRRSet {
	r := NewRRSetTXT()
	r.ID = libserver.TXTRecordName + "/" + libserver.RecordTypeTXT
	return r
}

func GetZone(token string, zone schema.Zone) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest(http.MethodGet, "/v1/zones/"+zone.Name),
		ghttp.VerifyHeader(http.Header{
			headerAuthorization: []string{authBearerPrefix + token},
		}),
		ghttp.RespondWithJSONEncoded(http.StatusOK, schema.ZoneGetResponse{
			Zone: zone,
		}),
	)
}

func GetRRSet(token string, zone schema.Zone, rrSet schema.ZoneRRSet, found bool) http.HandlerFunc {
	handlers := []http.HandlerFunc{
		ghttp.VerifyRequest(http.MethodGet, fmt.Sprintf("/v1/zones/%d/rrsets/%s/%s", zone.ID, rrSet.Name, rrSet.Type)),
		ghttp.VerifyHeader(http.Header{
			headerAuthorization: []string{authBearerPrefix + token},
		}),
	}

	if found {
		handlers = append(handlers, ghttp.RespondWithJSONEncoded(http.StatusOK, schema.ZoneRRSetGetResponse{
			RRSet: rrSet,
		}))
	} else {
		handlers = append(handlers, ghttp.RespondWithJSONEncoded(http.StatusNotFound, schema.ErrorResponse{
			Error: schema.Error{
				Code:    "not_found",
				Message: "rrset not found",
			},
		}))
	}

	return ghttp.CombineHandlers(handlers...)
}

func CreateRRSet(token string, zone schema.Zone, rrSet schema.ZoneRRSet) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest(http.MethodPost, fmt.Sprintf("/v1/zones/%d/rrsets", zone.ID)),
		ghttp.VerifyHeader(http.Header{
			headerAuthorization: []string{authBearerPrefix + token},
		}),
		ghttp.VerifyJSONRepresenting(schema.ZoneRRSetCreateRequest{
			Name:    rrSet.Name,
			Type:    rrSet.Type,
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
			headerAuthorization: []string{authBearerPrefix + token},
		}),
		ghttp.VerifyJSONRepresenting(schema.ZoneRRSetChangeTTLRequest{
			TTL: rrSet.TTL,
		}),
		getResponseSuccess(),
	)
}

func SetRRSetRecords(token string, zone schema.Zone, rrSet schema.ZoneRRSet) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest(http.MethodPost, fmt.Sprintf("/v1/zones/%d/rrsets/%s/%s/actions/set_records", zone.ID, rrSet.Name, rrSet.Type)),
		ghttp.VerifyHeader(http.Header{
			headerAuthorization: []string{authBearerPrefix + token},
		}),
		ghttp.VerifyJSONRepresenting(schema.ZoneRRSetSetRecordsRequest{
			Records: rrSet.Records,
		}),
		getResponseSuccess(),
	)
}

func RemoveRRSetRecords(token string, zone schema.Zone, rrSet schema.ZoneRRSet) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest(http.MethodPost, fmt.Sprintf("/v1/zones/%d/rrsets/%s/%s/actions/remove_records", zone.ID, rrSet.Name, rrSet.Type)),
		ghttp.VerifyHeader(http.Header{
			headerAuthorization: []string{authBearerPrefix + token},
		}),
		ghttp.VerifyJSONRepresenting(schema.ZoneRRSetRemoveRecordsRequest{
			Records: rrSet.Records,
		}),
		getResponseSuccess(),
	)
}

func getResponseSuccess() http.HandlerFunc {
	return ghttp.RespondWithJSONEncoded(http.StatusOK, schema.ActionGetResponse{
		Action: schema.Action{
			ID:     1,
			Status: "success",
		},
	})
}

func mustParseInt(s string) int64 {
	i, err := strconv.ParseInt(s, 10, 64)
	Expect(err).ToNot(HaveOccurred())
	return i
}

func ptr[T any](v T) *T {
	return &v
}

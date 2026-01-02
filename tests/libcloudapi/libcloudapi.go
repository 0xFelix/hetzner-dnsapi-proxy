package libcloudapi

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"github.com/0xfelix/hetzner-dnsapi-proxy/tests/libserver"
)

const (
	headerAuthorization = "Authorization"
	authBearerPrefix    = "Bearer "
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

func NewRRSetA() schema.ZoneRRSet {
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

func ExistingRRSetA() schema.ZoneRRSet {
	r := NewRRSetA()
	r.ID = libserver.ARecordName + "/" + libserver.RecordTypeA
	return r
}

func NewRRSetTXT() schema.ZoneRRSet {
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

func ExistingRRSetTXT() schema.ZoneRRSet {
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
	if found {
		return ghttp.CombineHandlers(
			ghttp.VerifyRequest(http.MethodGet, fmt.Sprintf("/v1/zones/%d/rrsets/%s/%s", zone.ID, rrSet.Name, rrSet.Type)),
			ghttp.VerifyHeader(http.Header{
				headerAuthorization: []string{authBearerPrefix + token},
			}),
			ghttp.RespondWithJSONEncoded(http.StatusOK, schema.ZoneRRSetGetResponse{
				RRSet: rrSet,
			}),
		)
	}
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest(http.MethodGet, fmt.Sprintf("/v1/zones/%d/rrsets/%s/%s", zone.ID, rrSet.Name, rrSet.Type)),
		ghttp.VerifyHeader(http.Header{
			headerAuthorization: []string{authBearerPrefix + token},
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
		respondSuccessAction(),
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
		respondSuccessAction(),
	)
}

func RemoveRRSetRecords(token string, zone schema.Zone, rrSet schema.ZoneRRSet) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest(http.MethodPost, fmt.Sprintf("/v1/zones/%d/rrsets/%s/%s/actions/remove_records", zone.ID, rrSet.Name, rrSet.Type)),
		ghttp.VerifyHeader(http.Header{
			headerAuthorization: []string{authBearerPrefix + token},
		}),
		ghttp.VerifyJSONRepresenting(schema.ZoneRRSetRemoveRecordsRequest{
			Records: rrSet.Records, // Simplified: assume we remove what we expect
		}),
		respondSuccessAction(),
	)
}

func respondSuccessAction() http.HandlerFunc {
	return ghttp.RespondWithJSONEncoded(http.StatusOK, schema.ActionGetResponse{
		Action: schema.Action{
			ID:     1,
			Status: "success",
		},
	})
}

func MustParseInt(s string) int64 {
	i, err := strconv.ParseInt(s, 10, 64)
	Expect(err).ToNot(HaveOccurred())
	return i
}

package libcloudapi

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"
	"github.com/onsi/gomega/ghttp"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/hetzner"
	"github.com/0xfelix/hetzner-dnsapi-proxy/tests/libdnsapi"
)

func GetZone(token string, zone hetzner.Zone) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest(http.MethodGet, "/v1/zones/"+zone.Name),
		ghttp.VerifyHeader(http.Header{
			"Authorization": []string{"Bearer " + token},
		}),
		ghttp.RespondWithJSONEncoded(http.StatusOK, schema.ZoneGetResponse{
			Zone: schema.Zone{
				ID:   MustParseInt(zone.ID),
				Name: zone.Name,
			},
		}),
	)
}

func GetRRSet(token string, zone hetzner.Zone, record hetzner.Record) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest(http.MethodGet, fmt.Sprintf("/v1/zones/%s/rrsets/%s/%s", zone.ID, record.Name, record.Type)),
		ghttp.VerifyHeader(http.Header{
			"Authorization": []string{"Bearer " + token},
		}),
		ghttp.RespondWithJSONEncoded(http.StatusOK, schema.ZoneRRSetGetResponse{
			RRSet: schema.ZoneRRSet{
				ID:   record.Name + "/" + record.Type,
				Name: record.Name,
				Type: record.Type,
				Zone: MustParseInt(zone.ID),
				Records: []schema.ZoneRRSetRecord{
					{Value: strconv.Quote(record.Value)},
				},
			},
		}),
	)
}

func GetRRSetNotFound(token string, zone hetzner.Zone, name, rType string) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest(http.MethodGet, fmt.Sprintf("/v1/zones/%s/rrsets/%s/%s", zone.ID, name, rType)),
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

func CreateRRSet(token string, zone hetzner.Zone, record hetzner.Record) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest(http.MethodPost, fmt.Sprintf("/v1/zones/%s/rrsets", zone.ID)),
		ghttp.VerifyHeader(http.Header{
			"Authorization": []string{"Bearer " + token},
		}),
		ghttp.VerifyJSONRepresenting(schema.ZoneRRSetCreateRequest{
			Name: record.Name,
			Type: string(hcloud.ZoneRRSetType(record.Type)),
			TTL:  &record.TTL,
			Records: []schema.ZoneRRSetRecord{
				{Value: strconv.Quote(record.Value)},
			},
		}),
		ghttp.RespondWithJSONEncoded(http.StatusCreated, schema.ZoneRRSetCreateResponse{
			RRSet: schema.ZoneRRSet{
				ID:   record.Name + "/" + record.Type,
				Name: record.Name,
				Type: record.Type,
				Records: []schema.ZoneRRSetRecord{
					{Value: strconv.Quote(record.Value)},
				},
			},
		}),
	)
}

func ChangeRRSetTTL(token string, zone hetzner.Zone, record hetzner.Record) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest(http.MethodPost, fmt.Sprintf("/v1/zones/%s/rrsets/%s/%s/actions/change_ttl", zone.ID, record.Name, record.Type)),
		ghttp.VerifyHeader(http.Header{
			"Authorization": []string{"Bearer " + token},
		}),
		ghttp.VerifyJSONRepresenting(schema.ZoneRRSetChangeTTLRequest{
			TTL: &record.TTL,
		}),
		ghttp.RespondWithJSONEncoded(http.StatusOK, schema.ActionGetResponse{
			Action: schema.Action{
				ID:     1,
				Status: "success",
			},
		}),
	)
}

func SetRRSetRecords(token string, zone hetzner.Zone, record hetzner.Record) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest(http.MethodPost, fmt.Sprintf("/v1/zones/%s/rrsets/%s/%s/actions/set_records", zone.ID, record.Name, record.Type)),
		ghttp.VerifyHeader(http.Header{
			"Authorization": []string{"Bearer " + token},
		}),
		ghttp.VerifyJSONRepresenting(schema.ZoneRRSetSetRecordsRequest{
			Records: []schema.ZoneRRSetRecord{
				{Value: strconv.Quote(record.Value)},
			},
		}),
		ghttp.RespondWithJSONEncoded(http.StatusOK, schema.ActionGetResponse{
			Action: schema.Action{
				ID:     1,
				Status: "success",
			},
		}),
	)
}

func RemoveRRSetRecords(token string, zone hetzner.Zone, record hetzner.Record) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest(http.MethodPost, fmt.Sprintf("/v1/zones/%s/rrsets/%s/%s/actions/remove_records", zone.ID, record.Name, record.Type)),
		ghttp.VerifyHeader(http.Header{
			"Authorization": []string{"Bearer " + token},
		}),
		ghttp.VerifyJSONRepresenting(schema.ZoneRRSetRemoveRecordsRequest{
			Records: []schema.ZoneRRSetRecord{
				{Value: strconv.Quote(libdnsapi.TXTExisting)}, // Mock assumes we fetch existing and remove it
			},
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

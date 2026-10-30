package tests

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
	"github.com/0xfelix/hetzner-dnsapi-proxy/tests/libcloudapi"
	"github.com/0xfelix/hetzner-dnsapi-proxy/tests/libserver"
)

const (
	typeCNAME = "CNAME"
	typeSRV   = "SRV"
	otherHost = "a.test.tld"
	srvInput  = "0 5 25567 mc.test.tld"
)

func rec(fqdn, typ, value string) map[string]string {
	return map[string]string{keyFQDN: fqdn, "type": typ, keyValue: value}
}

var _ = Describe("Records", func() {
	var (
		api      *ghttp.Server
		server   *httptest.Server
		token    string
		username string
		password string
	)

	BeforeEach(func() {
		api = ghttp.NewServer()
		server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL)
	})

	AfterEach(func() {
		server.Close()
		api.Close()
	})

	Context("should succeed", func() {
		DescribeTable(
			"creating a new record", func(ctx context.Context, rrSet schema.ZoneRRSet, fqdn, value string) {
				api.AppendHandlers(
					libcloudapi.GetZone(token, libcloudapi.Zone()),
					libcloudapi.GetRRSet(token, libcloudapi.Zone(), rrSet, false),
					libcloudapi.CreateRRSet(token, libcloudapi.Zone(), rrSet),
				)

				Expect(doRecordsRequest(
					ctx, server.URL+"/records/present", username, password,
					rec(fqdn, rrSet.Type, value),
				)).To(Equal(http.StatusOK))
				Expect(api.ReceivedRequests()).To(HaveLen(3))
			},
			Entry("CNAME", libcloudapi.NewRRSetCNAME(), libserver.CNAMERecordNameFull, "target.test.tld"),
			Entry("CNAME with dot suffix", libcloudapi.NewRRSetCNAME(), libserver.CNAMERecordNameFull+".", "target.test.tld."),
			Entry("SRV", libcloudapi.NewRRSetSRV(), libserver.SRVRecordNameFull, srvInput),
			Entry("SRV with dot suffix", libcloudapi.NewRRSetSRV(), libserver.SRVRecordNameFull+".", "0 5 25567 MC.test.tld."),
		)

		It("replacing an existing CNAME record", func(ctx context.Context) {
			api.AppendHandlers(
				libcloudapi.GetZone(token, libcloudapi.Zone()),
				libcloudapi.GetRRSet(token, libcloudapi.Zone(), libcloudapi.ExistingRRSetCNAME(), true),
				libcloudapi.SetRRSetRecords(token, libcloudapi.Zone(), libcloudapi.NewRRSetCNAME()),
			)

			Expect(doRecordsRequest(
				ctx, server.URL+"/records/present", username, password,
				rec(libserver.CNAMERecordNameFull, typeCNAME, "target.test.tld"),
			)).To(Equal(http.StatusOK))
			Expect(api.ReceivedRequests()).To(HaveLen(3))
		})

		DescribeTable(
			"adding a SRV record to an existing RRSet", func(ctx context.Context, ttl, expectedRequests int) {
				rrSet := libcloudapi.ExistingRRSetSRV()
				rrSet.TTL = &ttl
				api.AppendHandlers(
					libcloudapi.GetZone(token, libcloudapi.Zone()),
					libcloudapi.GetRRSet(token, libcloudapi.Zone(), rrSet, true),
				)
				if ttl != libserver.DefaultTTL {
					api.AppendHandlers(libcloudapi.ChangeRRSetTTL(token, libcloudapi.Zone(), libcloudapi.ExistingRRSetSRV()))
				}
				api.AppendHandlers(
					libcloudapi.AddRRSetRecords(token, libcloudapi.Zone(), rrSet, []schema.ZoneRRSetRecord{
						{Value: libserver.SRVValue},
					}),
				)

				Expect(doRecordsRequest(
					ctx, server.URL+"/records/present", username, password,
					rec(libserver.SRVRecordNameFull, typeSRV, srvInput),
				)).To(Equal(http.StatusOK))
				Expect(api.ReceivedRequests()).To(HaveLen(expectedRequests))
			},
			Entry("with matching TTL", libserver.DefaultTTL, 3),
			Entry("with different TTL", libserver.DefaultTTL+1, 4),
		)

		It("not adding a SRV record that already exists", func(ctx context.Context) {
			rrSet := libcloudapi.ExistingRRSetSRV()
			rrSet.Records = append(rrSet.Records, schema.ZoneRRSetRecord{Value: libserver.SRVValue})
			api.AppendHandlers(
				libcloudapi.GetZone(token, libcloudapi.Zone()),
				libcloudapi.GetRRSet(token, libcloudapi.Zone(), rrSet, true),
			)

			Expect(doRecordsRequest(
				ctx, server.URL+"/records/present", username, password,
				rec(libserver.SRVRecordNameFull, typeSRV, srvInput),
			)).To(Equal(http.StatusOK))
			Expect(api.ReceivedRequests()).To(HaveLen(2))
		})

		DescribeTable(
			"cleaning up a record", func(ctx context.Context, rrSet schema.ZoneRRSet, fqdn, value string) {
				api.AppendHandlers(
					libcloudapi.GetZone(token, libcloudapi.Zone()),
					libcloudapi.GetRRSet(token, libcloudapi.Zone(), rrSet, true),
					libcloudapi.RemoveRRSetRecords(token, libcloudapi.Zone(), rrSet, rrSet.Records),
				)

				Expect(doRecordsRequest(
					ctx, server.URL+"/records/cleanup", username, password,
					rec(fqdn, rrSet.Type, value),
				)).To(Equal(http.StatusOK))
				Expect(api.ReceivedRequests()).To(HaveLen(3))
			},
			Entry("CNAME", libcloudapi.NewRRSetCNAME(), libserver.CNAMERecordNameFull, "target.test.tld"),
			Entry("SRV", libcloudapi.NewRRSetSRV(), libserver.SRVRecordNameFull, srvInput),
		)

		It("cleaning up a record that does not exist", func(ctx context.Context) {
			api.AppendHandlers(
				libcloudapi.GetZone(token, libcloudapi.Zone()),
				libcloudapi.GetRRSet(token, libcloudapi.Zone(), libcloudapi.NewRRSetCNAME(), false),
			)

			Expect(doRecordsRequest(
				ctx, server.URL+"/records/cleanup", username, password,
				rec(libserver.CNAMERecordNameFull, typeCNAME, "target.test.tld"),
			)).To(Equal(http.StatusOK))
			Expect(api.ReceivedRequests()).To(HaveLen(2))
		})
	})

	Context("should make no api calls and should fail", func() {
		AfterEach(func() {
			Expect(api.ReceivedRequests()).To(BeEmpty())
		})

		DescribeTable(
			"on invalid input", func(ctx context.Context, data map[string]string) {
				Expect(doRecordsRequest(
					ctx, server.URL+"/records/present", username, password, data,
				)).To(Equal(http.StatusBadRequest))
			},
			Entry("missing type", map[string]string{"fqdn": libserver.CNAMERecordNameFull, "value": otherHost}),
			Entry("unsupported type", rec(libserver.CNAMERecordNameFull, "MX", otherHost)),
			Entry("CNAME at the zone apex", rec(libserver.ZoneName, typeCNAME, otherHost)),
			Entry("CNAME with invalid target", rec(libserver.CNAMERecordNameFull, typeCNAME, "a b.test.tld")),
			Entry("SRV with missing field", rec(libserver.SRVRecordNameFull, typeSRV, "0 5 mc.test.tld")),
			Entry("SRV with port out of range", rec(libserver.SRVRecordNameFull, typeSRV, "0 5 70000 mc.test.tld")),
			Entry("A with invalid address", rec(libserver.ARecordNameFull, "A", "nope")),
		)

		DescribeTable(
			"when access is denied", func(ctx context.Context, path string) {
				server.Close()
				server = libserver.NewNoAllowedDomains(api.URL())
				Expect(doRecordsRequest(
					ctx, server.URL+path, username, password,
					rec(libserver.CNAMERecordNameFull, typeCNAME, otherHost),
				)).To(Equal(http.StatusUnauthorized))
			},
			Entry("present", "/records/present"),
			Entry("cleanup", "/records/cleanup"),
		)

		DescribeTable(
			"when the endpoint is not enabled", func(ctx context.Context, path string) {
				server.Close()
				server, _, username, password = libserver.NewWithEndpoints(
					api.URL(), libserver.DefaultTTL, config.NewConfig().Endpoints,
				)
				Expect(doRecordsRequest(
					ctx, server.URL+path, username, password,
					rec(libserver.CNAMERecordNameFull, typeCNAME, otherHost),
				)).To(Equal(http.StatusNotFound))
			},
			Entry("present", "/records/present"),
			Entry("cleanup", "/records/cleanup"),
		)
	})
})

func doRecordsRequest(ctx context.Context, serverURL, username, password string, data map[string]string) int {
	body, err := json.Marshal(data)
	Expect(err).ToNot(HaveOccurred())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, serverURL, bytes.NewReader(body))
	Expect(err).ToNot(HaveOccurred())
	req.Header.Add("Content-Type", "application/json")
	req.SetBasicAuth(username, password)

	res, err := (&http.Client{}).Do(req)
	Expect(err).ToNot(HaveOccurred())
	Expect(res.Body.Close()).To(Succeed())

	return res.StatusCode
}

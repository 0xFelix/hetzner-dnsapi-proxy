package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"github.com/0xfelix/hetzner-dnsapi-proxy/tests/libcloudapi"
	"github.com/0xfelix/hetzner-dnsapi-proxy/tests/libdnsapi"
	"github.com/0xfelix/hetzner-dnsapi-proxy/tests/libserver"
)

var _ = Describe("HTTPReq", func() {
	var (
		api      *ghttp.Server
		server   *httptest.Server
		token    string
		username string
		password string
	)

	BeforeEach(func() {
		api = ghttp.NewServer()
	})

	AfterEach(func() {
		server.Close()
		api.Close()
	})

	Context("should succeed", func() {

		DescribeTable("creating a new record", func(ctx context.Context, cloudAPI bool, fqdn string, appendHandlers func()) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, cloudAPI)
			appendHandlers()

			Expect(doHTTPReqRequest(ctx, server.URL+"/httpreq/present", username, password,
				map[string]string{
					"fqdn":  fqdn,
					"value": libserver.TXTUpdated,
				},
			)).To(Equal(http.StatusOK))
			Expect(api.ReceivedRequests()).To(HaveLen(3))
		},
			Entry("DNS API: with dot suffix", false, libserver.TXTRecordNameFull+".", func() {
				api.AppendHandlers(
					libdnsapi.GetZones(token, libdnsapi.Zones()),
					libdnsapi.GetRecords(token, libserver.ZoneID, nil),
					libdnsapi.PostRecord(token, libdnsapi.NewTXTRecord()),
				)
			}),
			Entry("DNS API: without dot suffix", false, libserver.TXTRecordNameFull, func() {
				api.AppendHandlers(
					libdnsapi.GetZones(token, libdnsapi.Zones()),
					libdnsapi.GetRecords(token, libserver.ZoneID, nil),
					libdnsapi.PostRecord(token, libdnsapi.NewTXTRecord()),
				)
			}),
			Entry("Cloud API: with dot suffix", true, libserver.TXTRecordNameFull+".", func() {
				api.AppendHandlers(
					libcloudapi.GetZone(token, libcloudapi.Zones()[0]),
					libcloudapi.GetRRSetNotFound(token, libcloudapi.Zones()[0], libserver.TXTRecordName, "TXT"),
					libcloudapi.CreateRRSet(token, libcloudapi.Zones()[0], libcloudapi.NewTXTRecord()),
				)
			}),
			Entry("Cloud API: without dot suffix", true, libserver.TXTRecordNameFull, func() {
				api.AppendHandlers(
					libcloudapi.GetZone(token, libcloudapi.Zones()[0]),
					libcloudapi.GetRRSetNotFound(token, libcloudapi.Zones()[0], libserver.TXTRecordName, "TXT"),
					libcloudapi.CreateRRSet(token, libcloudapi.Zones()[0], libcloudapi.NewTXTRecord()),
				)
			}),
		)

		DescribeTable("updating an existing record", func(ctx context.Context, cloudAPI bool, fqdn string, appendHandlers func()) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, cloudAPI)
			appendHandlers()

			Expect(doHTTPReqRequest(ctx, server.URL+"/httpreq/present", username, password,
				map[string]string{
					"fqdn":  fqdn,
					"value": libserver.TXTUpdated,
				},
			)).To(Equal(http.StatusOK))
			if cloudAPI {
				Expect(api.ReceivedRequests()).To(HaveLen(4))
			} else {
				Expect(api.ReceivedRequests()).To(HaveLen(3))
			}
		},
			Entry("DNS API: with dot suffix", false, libserver.TXTRecordNameFull+".", func() {
				api.AppendHandlers(
					libdnsapi.GetZones(token, libdnsapi.Zones()),
					libdnsapi.GetRecords(token, libserver.ZoneID, libdnsapi.Records()),
					libdnsapi.PutRecord(token, libdnsapi.UpdatedTXTRecord()),
				)
			}),
			Entry("DNS API: without dot suffix", false, libserver.TXTRecordNameFull, func() {
				api.AppendHandlers(
					libdnsapi.GetZones(token, libdnsapi.Zones()),
					libdnsapi.GetRecords(token, libserver.ZoneID, libdnsapi.Records()),
					libdnsapi.PutRecord(token, libdnsapi.UpdatedTXTRecord()),
				)
			}),
			Entry("Cloud API: with dot suffix", true, libserver.TXTRecordNameFull+".", func() {
				api.AppendHandlers(
					libcloudapi.GetZone(token, libcloudapi.Zones()[0]),
					libcloudapi.GetRRSet(token, libcloudapi.Zones()[0], libcloudapi.Records()[1]),
					libcloudapi.ChangeRRSetTTL(token, libcloudapi.Zones()[0], libcloudapi.UpdatedTXTRecord()),
					libcloudapi.SetRRSetRecords(token, libcloudapi.Zones()[0], libcloudapi.UpdatedTXTRecord()),
				)
			}),
			Entry("Cloud API: without dot suffix", true, libserver.TXTRecordNameFull, func() {
				api.AppendHandlers(
					libcloudapi.GetZone(token, libcloudapi.Zones()[0]),
					libcloudapi.GetRRSet(token, libcloudapi.Zones()[0], libcloudapi.Records()[1]),
					libcloudapi.ChangeRRSetTTL(token, libcloudapi.Zones()[0], libcloudapi.UpdatedTXTRecord()),
					libcloudapi.SetRRSetRecords(token, libcloudapi.Zones()[0], libcloudapi.UpdatedTXTRecord()),
				)
			}),
		)

		DescribeTable("should succeed cleaning up via Cloud API", func(ctx context.Context, fqdn string, appendHandlers func()) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, true)
			appendHandlers()

			Expect(doHTTPReqRequest(ctx, server.URL+"/httpreq/cleanup", username, password,
				map[string]string{
					"fqdn": fqdn,
				},
			)).To(Equal(http.StatusOK))
			Expect(api.ReceivedRequests()).To(HaveLen(3))
		},
			Entry("with dot suffix", libserver.TXTRecordNameFull+".", func() {
				api.AppendHandlers(
					libcloudapi.GetZone(token, libcloudapi.Zones()[0]),
					libcloudapi.GetRRSet(token, libcloudapi.Zones()[0], libcloudapi.NewTXTRecord()),
					libcloudapi.RemoveRRSetRecords(token, libcloudapi.Zones()[0], libcloudapi.NewTXTRecord()),
				)
			}),
			Entry("without dot suffix", libserver.TXTRecordNameFull, func() {
				api.AppendHandlers(
					libcloudapi.GetZone(token, libcloudapi.Zones()[0]),
					libcloudapi.GetRRSet(token, libcloudapi.Zones()[0], libcloudapi.NewTXTRecord()),
					libcloudapi.RemoveRRSetRecords(token, libcloudapi.Zones()[0], libcloudapi.NewTXTRecord()),
				)
			}),
		)
	})

	Context("should make no api calls and", func() {
		AfterEach(func() {
			Expect(api.ReceivedRequests()).To(BeEmpty())
		})

		DescribeTable("should succeed cleaning up", func(ctx context.Context, fqdn string) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, false)

			Expect(doHTTPReqRequest(ctx, server.URL+"/httpreq/cleanup", username, password,
				map[string]string{
					"fqdn": fqdn,
				},
			)).To(Equal(http.StatusOK))
		},
			Entry("with dot suffix", libserver.TXTRecordNameFull+"."),
			Entry("without dot suffix", libserver.TXTRecordNameFull),
		)

		Context("should fail", func() {
			DescribeTable("when fqdn is missing", func(ctx context.Context, cloudAPI bool) {
				server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, cloudAPI)
				Expect(doHTTPReqRequest(ctx, server.URL+"/httpreq/present", username, password,
					map[string]string{
						"value": libserver.TXTUpdated,
					},
				)).To(Equal(http.StatusBadRequest))
			},
				Entry("DNS API", false),
				Entry("Cloud API", true),
			)

			DescribeTable("when value is missing", func(ctx context.Context, cloudAPI bool) {
				server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, cloudAPI)
				Expect(doHTTPReqRequest(ctx, server.URL+"/httpreq/present", username, password,
					map[string]string{
						"fqdn": libserver.TXTRecordNameFull,
					},
				)).To(Equal(http.StatusBadRequest))
			},
				Entry("DNS API", false),
				Entry("Cloud API", true),
			)

			DescribeTable("when fqdn is malformed", func(ctx context.Context, cloudAPI bool) {
				server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, cloudAPI)
				Expect(doHTTPReqRequest(ctx, server.URL+"/httpreq/present", username, password,
					map[string]string{
						"fqdn":  libserver.TLD,
						"value": libserver.TXTUpdated,
					},
				)).To(Equal(http.StatusBadRequest))
			},
				Entry("DNS API", false),
				Entry("Cloud API", true),
			)

			DescribeTable("when access is denied", func(ctx context.Context, fqdn string, cloudAPI bool) {
				server = libserver.NewNoAllowedDomains(api.URL(), cloudAPI)
				Expect(doHTTPReqRequest(ctx, server.URL+"/httpreq/present", username, password,
					map[string]string{
						"fqdn":  fqdn,
						"value": libserver.TXTUpdated,
					},
				)).To(Equal(http.StatusUnauthorized))
			},
				Entry("DNS API: with dot suffix", libserver.TXTRecordNameFull+".", false),
				Entry("DNS API: without dot suffix", libserver.TXTRecordNameFull, false),
				Entry("Cloud API: with dot suffix", libserver.TXTRecordNameFull+".", true),
				Entry("Cloud API: without dot suffix", libserver.TXTRecordNameFull, true),
			)
		})
	})
})

func doHTTPReqRequest(ctx context.Context, serverURL, username, password string, data map[string]string) int {
	body, err := json.Marshal(data)
	Expect(err).ToNot(HaveOccurred())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, serverURL, bytes.NewReader(body))
	Expect(err).ToNot(HaveOccurred())
	req.Header.Add("Content-Type", "application/json")
	req.SetBasicAuth(username, password)

	c := &http.Client{}
	res, err := c.Do(req)
	Expect(err).ToNot(HaveOccurred())
	Expect(res.Body.Close()).To(Succeed())

	return res.StatusCode
}

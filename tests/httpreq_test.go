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
		DescribeTable("creating a new record", func(ctx context.Context, cloudAPI bool, fqdn string) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, cloudAPI)

			if cloudAPI {
				api.AppendHandlers(
					libcloudapi.GetZone(token, libcloudapi.Zone()),
					libcloudapi.GetRRSet(token, libcloudapi.Zone(), libcloudapi.NewRRSetTXT(), false),
					libcloudapi.CreateRRSet(token, libcloudapi.Zone(), libcloudapi.NewRRSetTXT()),
				)
			} else {
				api.AppendHandlers(
					libdnsapi.GetZones(token, libdnsapi.Zones()),
					libdnsapi.GetRecords(token, libserver.ZoneID, nil),
					libdnsapi.PostRecord(token, libdnsapi.NewTXTRecord()),
				)
			}

			Expect(doHTTPReqRequest(ctx, server.URL+"/httpreq/present", username, password,
				map[string]string{
					"fqdn":  fqdn,
					"value": libserver.TXTUpdated,
				},
			)).To(Equal(http.StatusOK))
			Expect(api.ReceivedRequests()).To(HaveLen(3))
		},
			Entry("DNS API: with dot suffix", false, libserver.TXTRecordNameFull+"."),
			Entry("DNS API: without dot suffix", false, libserver.TXTRecordNameFull),
			Entry("Cloud API: with dot suffix", true, libserver.TXTRecordNameFull+"."),
			Entry("Cloud API: without dot suffix", true, libserver.TXTRecordNameFull),
		)

		DescribeTable("updating an existing record", func(ctx context.Context, cloudAPI bool, fqdn string) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, cloudAPI)

			if cloudAPI {
				api.AppendHandlers(
					libcloudapi.GetZone(token, libcloudapi.Zone()),
					libcloudapi.GetRRSet(token, libcloudapi.Zone(), libcloudapi.ExistingRRSetTXT(), true),
					libcloudapi.ChangeRRSetTTL(token, libcloudapi.Zone(), libcloudapi.UpdatedRRSetTXT()),
					libcloudapi.SetRRSetRecords(token, libcloudapi.Zone(), libcloudapi.UpdatedRRSetTXT()),
				)
			} else {
				api.AppendHandlers(
					libdnsapi.GetZones(token, libdnsapi.Zones()),
					libdnsapi.GetRecords(token, libserver.ZoneID, libdnsapi.Records()),
					libdnsapi.PutRecord(token, libdnsapi.UpdatedTXTRecord()),
				)
			}

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
			Entry("DNS API: with dot suffix", false, libserver.TXTRecordNameFull+"."),
			Entry("DNS API: without dot suffix", false, libserver.TXTRecordNameFull),
			Entry("Cloud API: with dot suffix", true, libserver.TXTRecordNameFull+"."),
			Entry("Cloud API: without dot suffix", true, libserver.TXTRecordNameFull),
		)

		DescribeTable("cleaning up", func(ctx context.Context, cloudAPI bool, fqdn string) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, cloudAPI)

			if cloudAPI {
				api.AppendHandlers(
					libcloudapi.GetZone(token, libcloudapi.Zone()),
					libcloudapi.GetRRSet(token, libcloudapi.Zone(), libcloudapi.ExistingRRSetTXT(), true),
					libcloudapi.RemoveRRSetRecords(token, libcloudapi.Zone(), libcloudapi.ExistingRRSetTXT()),
				)
			}
			// DNS API entries have nil appendHandlers, so no handlers are appended for DNS API.

			Expect(doHTTPReqRequest(ctx, server.URL+"/httpreq/cleanup", username, password,
				map[string]string{
					"fqdn": fqdn,
				},
			)).To(Equal(http.StatusOK))
			if cloudAPI {
				Expect(api.ReceivedRequests()).To(HaveLen(3))
			} else {
				Expect(api.ReceivedRequests()).To(BeEmpty())
			}
		},
			Entry("DNS API: with dot suffix", false, libserver.TXTRecordNameFull+"."),
			Entry("DNS API: without dot suffix", false, libserver.TXTRecordNameFull),
			Entry("Cloud API: with dot suffix", true, libserver.TXTRecordNameFull+"."),
			Entry("Cloud API: without dot suffix", true, libserver.TXTRecordNameFull),
		)
	})

	Context("should make no api calls and should fail", func() {
		AfterEach(func() {
			Expect(api.ReceivedRequests()).To(BeEmpty())
		})

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

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

	"github.com/0xfelix/hetzner-dnsapi-proxy/tests/libapi"
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
		server, token, username, password = libserver.New(api.URL(), libapi.DefaultTTL)
	})

	AfterEach(func() {
		server.Close()
		api.Close()
	})

	Context("should succeed", func() {
		DescribeTable("creating a new record", func(ctx context.Context, fqdn string) {
			api.AppendHandlers(
				libapi.GetZones(token, libapi.Zones()),
				libapi.GetRecords(token, libapi.ZoneID, nil),
				libapi.PostRecord(token, libapi.NewTXTRecord()),
			)

			Expect(doHTTPReqRequest(ctx, server.URL+"/httpreq/present", username, password,
				map[string]string{
					"fqdn":  fqdn,
					"value": libapi.TXTUpdated,
				},
			)).To(Equal(http.StatusOK))

			Expect(api.ReceivedRequests()).To(HaveLen(3))
		},
			Entry("with dot suffix", libapi.TXTRecordNameFull+"."),
			Entry("without dot suffix", libapi.TXTRecordNameFull),
		)

		DescribeTable("updating an existing record", func(ctx context.Context, fqdn string) {
			api.AppendHandlers(
				libapi.GetZones(token, libapi.Zones()),
				libapi.GetRecords(token, libapi.ZoneID, libapi.Records()),
				libapi.PutRecord(token, libapi.UpdatedTXTRecord()),
			)

			Expect(doHTTPReqRequest(ctx, server.URL+"/httpreq/present", username, password,
				map[string]string{
					"fqdn":  fqdn,
					"value": libapi.TXTUpdated,
				},
			)).To(Equal(http.StatusOK))

			Expect(api.ReceivedRequests()).To(HaveLen(3))
		},
			Entry("with dot suffix", libapi.TXTRecordNameFull+"."),
			Entry("without dot suffix", libapi.TXTRecordNameFull),
		)

		DescribeTable("cleaning up a record", func(ctx context.Context, fqdn string) {
			Expect(doHTTPReqRequest(ctx, server.URL+"/httpreq/cleanup", username, password,
				map[string]string{
					"fqdn": fqdn,
				},
			)).To(Equal(http.StatusOK))
		},
			Entry("with dot suffix", libapi.TXTRecordNameFull+"."),
			Entry("without dot suffix", libapi.TXTRecordNameFull),
		)
	})

	Context("should make no api calls and", func() {
		AfterEach(func() {
			Expect(api.ReceivedRequests()).To(BeEmpty())
		})

		Context("should fail", func() {
			It("when fqdn is missing", func(ctx context.Context) {
				Expect(doHTTPReqRequest(ctx, server.URL+"/httpreq/present", username, password,
					map[string]string{
						"value": libapi.TXTUpdated,
					},
				)).To(Equal(http.StatusBadRequest))
			})

			It("when value is missing", func(ctx context.Context) {
				Expect(doHTTPReqRequest(ctx, server.URL+"/httpreq/present", username, password,
					map[string]string{
						"fqdn": libapi.TXTRecordNameFull,
					},
				)).To(Equal(http.StatusBadRequest))
			})

			It("when fqdn is malformed", func(ctx context.Context) {
				Expect(doHTTPReqRequest(ctx, server.URL+"/httpreq/present", username, password,
					map[string]string{
						"fqdn":  libapi.TLD,
						"value": libapi.TXTUpdated,
					},
				)).To(Equal(http.StatusBadRequest))
			})

			DescribeTable("when access is denied", func(ctx context.Context, fqdn string) {
				server = libserver.NewNoAllowedDomains(api.URL())
				Expect(doHTTPReqRequest(ctx, server.URL+"/httpreq/present", username, password,
					map[string]string{
						"fqdn":  fqdn,
						"value": libapi.TXTUpdated,
					},
				)).To(Equal(http.StatusUnauthorized))
			},
				Entry("with dot suffix", libapi.TXTRecordNameFull+"."),
				Entry("without dot suffix", libapi.TXTRecordNameFull),
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

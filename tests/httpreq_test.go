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
		api    *ghttp.Server
		server *httptest.Server
		token  string
	)

	BeforeEach(func() {
		api = ghttp.NewServer()
		server, token = libserver.New(api.URL(), libapi.DefaultTTL)
	})

	AfterEach(func() {
		server.Close()
		api.Close()
	})

	Context("should succeed", func() {
		AfterEach(func() {
			Expect(api.ReceivedRequests()).To(HaveLen(3))
		})

		DescribeTable("creating a new record", func(ctx context.Context, fqdn string) {
			api.AppendHandlers(
				libapi.GetZones(token, libapi.Zones()),
				libapi.GetRecords(token, libapi.ZoneID, nil),
				libapi.PostRecord(token, libapi.NewTXTRecord()),
			)

			Expect(doHTTPReqRequest(ctx, server.URL+"/httpreq/present", map[string]string{
				"fqdn":  fqdn,
				"value": libapi.TXTUpdated,
			})).To(Equal(http.StatusOK))
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

			Expect(doHTTPReqRequest(ctx, server.URL+"/httpreq/present", map[string]string{
				"fqdn":  fqdn,
				"value": libapi.TXTUpdated,
			})).To(Equal(http.StatusOK))
		},
			Entry("with dot suffix", libapi.TXTRecordNameFull+"."),
			Entry("without dot suffix", libapi.TXTRecordNameFull),
		)
	})

	Context("should make no api calls and", func() {
		AfterEach(func() {
			Expect(api.ReceivedRequests()).To(HaveLen(0))
		})

		It("should succeed cleaning up", func() {
			res, err := http.Post(server.URL+"/httpreq/cleanup", "application/json", nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.StatusCode).To(Equal(http.StatusOK))
		})

		Context("should fail", func() {
			It("when fqdn is missing", func(ctx context.Context) {
				Expect(doHTTPReqRequest(ctx, server.URL+"/httpreq/present", map[string]string{
					"value": libapi.TXTUpdated,
				})).To(Equal(http.StatusBadRequest))
			})

			It("when value is missing", func(ctx context.Context) {
				Expect(doHTTPReqRequest(ctx, server.URL+"/httpreq/present", map[string]string{
					"fqdn": libapi.TXTRecordNameFull,
				})).To(Equal(http.StatusBadRequest))
			})
		})
	})
})

func doHTTPReqRequest(ctx context.Context, url string, data map[string]string) int {
	body, err := json.Marshal(data)
	Expect(err).ToNot(HaveOccurred())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	Expect(err).ToNot(HaveOccurred())
	req.Header.Add("Content-Type", "application/json")

	c := &http.Client{}
	res, err := c.Do(req)
	Expect(err).ToNot(HaveOccurred())
	Expect(res.Body.Close()).To(Succeed())

	return res.StatusCode
}

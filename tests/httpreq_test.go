package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"

	"github.com/0xfelix/hetzner-dnsapi-proxy/tests/libcloudapi"
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
		DescribeTable("creating a new record", func(ctx context.Context, fqdn string) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL)

			api.AppendHandlers(
				libcloudapi.GetZone(token, libcloudapi.Zone()),
				libcloudapi.GetRRSet(token, libcloudapi.Zone(), libcloudapi.NewRRSetTXT(), false),
				libcloudapi.CreateRRSet(token, libcloudapi.Zone(), libcloudapi.NewRRSetTXT()),
			)

			Expect(doHTTPReqRequest(ctx, server.URL+"/httpreq/present", username, password,
				map[string]string{
					"fqdn":  fqdn,
					"value": libserver.TXTUpdated,
				},
			)).To(Equal(http.StatusOK))
			Expect(api.ReceivedRequests()).To(HaveLen(3))
		},
			Entry("with dot suffix", libserver.TXTRecordNameFull+"."),
			Entry("without dot suffix", libserver.TXTRecordNameFull),
		)

		DescribeTable("updating an existing record", func(ctx context.Context, fqdn string) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL)

			api.AppendHandlers(
				libcloudapi.GetZone(token, libcloudapi.Zone()),
				libcloudapi.GetRRSet(token, libcloudapi.Zone(), libcloudapi.ExistingRRSetTXT(), true),
				libcloudapi.ChangeRRSetTTL(token, libcloudapi.Zone(), libcloudapi.UpdatedRRSetTXT()),
				libcloudapi.SetRRSetRecords(token, libcloudapi.Zone(), libcloudapi.UpdatedRRSetTXT()),
			)

			Expect(doHTTPReqRequest(ctx, server.URL+"/httpreq/present", username, password,
				map[string]string{
					"fqdn":  fqdn,
					"value": libserver.TXTUpdated,
				},
			)).To(Equal(http.StatusOK))
			Expect(api.ReceivedRequests()).To(HaveLen(4))
		},
			Entry("with dot suffix", libserver.TXTRecordNameFull+"."),
			Entry("without dot suffix", libserver.TXTRecordNameFull),
		)

		DescribeTable("cleaning up", func(ctx context.Context, fqdn string) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL)

			rrSet := libcloudapi.ExistingRRSetTXT()
			api.AppendHandlers(
				libcloudapi.GetZone(token, libcloudapi.Zone()),
				libcloudapi.GetRRSet(token, libcloudapi.Zone(), rrSet, true),
				libcloudapi.RemoveRRSetRecords(token, libcloudapi.Zone(), rrSet, []schema.ZoneRRSetRecord{
					{Value: strconv.Quote(libserver.TXTExisting)},
				}),
			)

			Expect(doHTTPReqRequest(ctx, server.URL+"/httpreq/cleanup", username, password,
				map[string]string{
					"fqdn":  fqdn,
					"value": libserver.TXTExisting,
				},
			)).To(Equal(http.StatusOK))
			Expect(api.ReceivedRequests()).To(HaveLen(3))
		},
			Entry("with dot suffix", libserver.TXTRecordNameFull+"."),
			Entry("without dot suffix", libserver.TXTRecordNameFull),
		)
	})

	Context("should make no api calls and should fail", func() {
		AfterEach(func() {
			Expect(api.ReceivedRequests()).To(BeEmpty())
		})

		It("when fqdn is missing", func(ctx context.Context) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL)
			Expect(doHTTPReqRequest(ctx, server.URL+"/httpreq/present", username, password,
				map[string]string{
					"value": libserver.TXTUpdated,
				},
			)).To(Equal(http.StatusBadRequest))
		})

		It("when value is missing", func(ctx context.Context) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL)
			Expect(doHTTPReqRequest(ctx, server.URL+"/httpreq/present", username, password,
				map[string]string{
					"fqdn": libserver.TXTRecordNameFull,
				},
			)).To(Equal(http.StatusBadRequest))
		})

		It("when fqdn is malformed", func(ctx context.Context) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL)
			Expect(doHTTPReqRequest(ctx, server.URL+"/httpreq/present", username, password,
				map[string]string{
					"fqdn":  libserver.TLD,
					"value": libserver.TXTUpdated,
				},
			)).To(Equal(http.StatusBadRequest))
		})

		DescribeTable("when access is denied", func(ctx context.Context, fqdn string) {
			server = libserver.NewNoAllowedDomains(api.URL())
			Expect(doHTTPReqRequest(ctx, server.URL+"/httpreq/present", username, password,
				map[string]string{
					"fqdn":  fqdn,
					"value": libserver.TXTUpdated,
				},
			)).To(Equal(http.StatusUnauthorized))
		},
			Entry("with dot suffix", libserver.TXTRecordNameFull+"."),
			Entry("without dot suffix", libserver.TXTRecordNameFull),
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

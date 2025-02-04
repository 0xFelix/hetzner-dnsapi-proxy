package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"github.com/0xfelix/hetzner-dnsapi-proxy/tests/libapi"
	"github.com/0xfelix/hetzner-dnsapi-proxy/tests/libserver"
)

var _ = Describe("Plain", func() {
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

		It("creating a new record", func(ctx context.Context) {
			api.AppendHandlers(
				libapi.GetZones(token, libapi.Zones()),
				libapi.GetRecords(token, libapi.ZoneID, nil),
				libapi.PostRecord(token, libapi.NewARecord()),
			)

			Expect(doPlainRequest(ctx, server.URL+"/plain/update", url.Values{
				"hostname": []string{libapi.ARecordNameFull},
				"ip":       []string{libapi.AUpdated},
			})).To(Equal(http.StatusOK))
		})

		It("updating an existing record", func(ctx context.Context) {
			api.AppendHandlers(
				libapi.GetZones(token, libapi.Zones()),
				libapi.GetRecords(token, libapi.ZoneID, libapi.Records()),
				libapi.PutRecord(token, libapi.UpdatedARecord()),
			)

			Expect(doPlainRequest(ctx, server.URL+"/plain/update", url.Values{
				"hostname": []string{libapi.ARecordNameFull},
				"ip":       []string{libapi.AUpdated},
			})).To(Equal(http.StatusOK))
		})
	})

	Context("should make no api calls and should fail", func() {
		AfterEach(func() {
			Expect(api.ReceivedRequests()).To(HaveLen(0))
		})

		It("when hostname is missing", func(ctx context.Context) {
			Expect(doPlainRequest(ctx, server.URL+"/plain/update", url.Values{
				"ip": []string{libapi.AUpdated},
			})).To(Equal(http.StatusBadRequest))
		})

		It("when ip is missing", func(ctx context.Context) {
			Expect(doPlainRequest(ctx, server.URL+"/plain/update", url.Values{
				"hostname": []string{libapi.ARecordNameFull},
			})).To(Equal(http.StatusBadRequest))
		})

		It("when hostname is malformed", func(ctx context.Context) {
			Expect(doPlainRequest(ctx, server.URL+"/plain/update", url.Values{
				"hostname": []string{libapi.TLD},
				"ip":       []string{libapi.AUpdated},
			})).To(Equal(http.StatusBadRequest))
		})
	})
})

func doPlainRequest(ctx context.Context, serverURL string, data url.Values) int {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL, http.NoBody)
	Expect(err).ToNot(HaveOccurred())
	req.URL.RawQuery = data.Encode()

	c := &http.Client{}
	res, err := c.Do(req)
	Expect(err).ToNot(HaveOccurred())
	Expect(res.Body.Close()).To(Succeed())

	return res.StatusCode
}

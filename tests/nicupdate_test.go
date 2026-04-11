package tests

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"github.com/0xfelix/hetzner-dnsapi-proxy/tests/libcloudapi"
	"github.com/0xfelix/hetzner-dnsapi-proxy/tests/libserver"
)

var _ = Describe("NicUpdate", func() {
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
		//nolint:dupl
		It("creating a new record", func(ctx context.Context) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL)

			api.AppendHandlers(
				libcloudapi.GetZone(token, libcloudapi.Zone()),
				libcloudapi.GetRRSet(token, libcloudapi.Zone(), libcloudapi.NewRRSetA(), false),
				libcloudapi.CreateRRSet(token, libcloudapi.Zone(), libcloudapi.NewRRSetA()),
			)

			status, body := doNicRequest(ctx, server.URL+"/nic/update", username, password, url.Values{
				"hostname": []string{libserver.ARecordNameFull},
				"myip":     []string{libserver.AUpdated},
			})
			Expect(status).To(Equal(http.StatusOK))
			Expect(body).To(Equal("good " + libserver.AUpdated))
			Expect(api.ReceivedRequests()).To(HaveLen(3))
		})

		//nolint:dupl
		It("creating a new AAAA record", func(ctx context.Context) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL)

			api.AppendHandlers(
				libcloudapi.GetZone(token, libcloudapi.Zone()),
				libcloudapi.GetRRSet(token, libcloudapi.Zone(), libcloudapi.NewRRSetAAAA(), false),
				libcloudapi.CreateRRSet(token, libcloudapi.Zone(), libcloudapi.NewRRSetAAAA()),
			)

			status, body := doNicRequest(ctx, server.URL+"/nic/update", username, password, url.Values{
				"hostname": []string{libserver.AAAARecordNameFull},
				"myip":     []string{libserver.AAAAUpdated},
			})
			Expect(status).To(Equal(http.StatusOK))
			Expect(body).To(Equal("good " + libserver.AAAAUpdated))
			Expect(api.ReceivedRequests()).To(HaveLen(3))
		})

		//nolint:dupl
		It("updating an existing record", func(ctx context.Context) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL)

			api.AppendHandlers(
				libcloudapi.GetZone(token, libcloudapi.Zone()),
				libcloudapi.GetRRSet(token, libcloudapi.Zone(), libcloudapi.ExistingRRSetA(), true),
				libcloudapi.ChangeRRSetTTL(token, libcloudapi.Zone(), libcloudapi.UpdatedRRSetA()),
				libcloudapi.SetRRSetRecords(token, libcloudapi.Zone(), libcloudapi.UpdatedRRSetA()),
			)

			status, body := doNicRequest(ctx, server.URL+"/nic/update", username, password, url.Values{
				"hostname": []string{libserver.ARecordNameFull},
				"myip":     []string{libserver.AUpdated},
			})
			Expect(status).To(Equal(http.StatusOK))
			Expect(body).To(Equal("good " + libserver.AUpdated))
			Expect(api.ReceivedRequests()).To(HaveLen(4))
		})

		//nolint:dupl
		It("updating an existing AAAA record", func(ctx context.Context) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL)

			api.AppendHandlers(
				libcloudapi.GetZone(token, libcloudapi.Zone()),
				libcloudapi.GetRRSet(token, libcloudapi.Zone(), libcloudapi.ExistingRRSetAAAA(), true),
				libcloudapi.ChangeRRSetTTL(token, libcloudapi.Zone(), libcloudapi.UpdatedRRSetAAAA()),
				libcloudapi.SetRRSetRecords(token, libcloudapi.Zone(), libcloudapi.UpdatedRRSetAAAA()),
			)

			status, body := doNicRequest(ctx, server.URL+"/nic/update", username, password, url.Values{
				"hostname": []string{libserver.AAAARecordNameFull},
				"myip":     []string{libserver.AAAAUpdated},
			})
			Expect(status).To(Equal(http.StatusOK))
			Expect(body).To(Equal("good " + libserver.AAAAUpdated))
			Expect(api.ReceivedRequests()).To(HaveLen(4))
		})

		It("using client ip when myip is omitted", func(ctx context.Context) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL)

			api.AppendHandlers(
				libcloudapi.GetZone(token, libcloudapi.Zone()),
				libcloudapi.GetRRSet(token, libcloudapi.Zone(), libcloudapi.ExistingRRSetA(), true),
				libcloudapi.ChangeRRSetTTL(token, libcloudapi.Zone(), libcloudapi.ClientIPRRSetA()),
				libcloudapi.SetRRSetRecords(token, libcloudapi.Zone(), libcloudapi.ClientIPRRSetA()),
			)

			status, body := doNicRequest(ctx, server.URL+"/nic/update", username, password, url.Values{
				"hostname": []string{libserver.ARecordNameFull},
			})
			Expect(status).To(Equal(http.StatusOK))
			Expect(body).To(Equal("good " + libserver.AExisting))
			Expect(api.ReceivedRequests()).To(HaveLen(4))
		})
	})

	Context("should make no api calls and return a DynDNS2 error token", func() {
		AfterEach(func() {
			Expect(api.ReceivedRequests()).To(BeEmpty())
		})

		It("notfqdn when hostname is missing", func(ctx context.Context) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL)
			status, body := doNicRequest(ctx, server.URL+"/nic/update", username, password, url.Values{
				"myip": []string{libserver.AUpdated},
			})
			Expect(status).To(Equal(http.StatusOK))
			Expect(body).To(Equal("notfqdn"))
		})

		It("notfqdn when myip is invalid", func(ctx context.Context) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL)
			status, body := doNicRequest(ctx, server.URL+"/nic/update", username, password, url.Values{
				"hostname": []string{libserver.ARecordNameFull},
				"myip":     []string{"invalid"},
			})
			Expect(status).To(Equal(http.StatusOK))
			Expect(body).To(Equal("notfqdn"))
		})

		It("notfqdn when hostname is malformed", func(ctx context.Context) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL)
			status, body := doNicRequest(ctx, server.URL+"/nic/update", username, password, url.Values{
				"hostname": []string{libserver.TLD},
				"myip":     []string{libserver.AUpdated},
			})
			Expect(status).To(Equal(http.StatusOK))
			Expect(body).To(Equal("notfqdn"))
		})

		It("nohost when ip-only auth denies access", func(ctx context.Context) {
			server = libserver.NewNoAllowedDomains(api.URL())
			status, body := doNicRequest(ctx, server.URL+"/nic/update", username, password, url.Values{
				"hostname": []string{libserver.ARecordNameFull},
				"myip":     []string{libserver.AUpdated},
			})
			Expect(status).To(Equal(http.StatusOK))
			Expect(body).To(Equal("nohost"))
		})

		It("badauth when credentials are invalid", func(ctx context.Context) {
			server, _, username, password = libserver.New(api.URL(), libserver.DefaultTTL)
			status, body := doNicRequest(ctx, server.URL+"/nic/update", username+"x", password,
				url.Values{
					"hostname": []string{libserver.ARecordNameFull},
					"myip":     []string{libserver.AUpdated},
				})
			Expect(status).To(Equal(http.StatusUnauthorized))
			Expect(body).To(Equal("badauth"))
		})
	})
})

func doNicRequest(ctx context.Context, serverURL, username, password string, data url.Values) (status int, body string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL, http.NoBody)
	Expect(err).ToNot(HaveOccurred())
	req.URL.RawQuery = data.Encode()
	req.SetBasicAuth(username, password)

	c := &http.Client{}
	res, err := c.Do(req)
	Expect(err).ToNot(HaveOccurred())
	b, err := io.ReadAll(res.Body)
	Expect(err).ToNot(HaveOccurred())
	Expect(res.Body.Close()).To(Succeed())

	return res.StatusCode, string(b)
}

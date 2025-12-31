package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"github.com/0xfelix/hetzner-dnsapi-proxy/tests/libcloudapi"
	"github.com/0xfelix/hetzner-dnsapi-proxy/tests/libdnsapi"
	"github.com/0xfelix/hetzner-dnsapi-proxy/tests/libserver"
)

var _ = Describe("Plain", func() {
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
		DescribeTable("creating a new record", func(ctx context.Context, cloudAPI bool) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, cloudAPI)

			if cloudAPI {
				api.AppendHandlers(
					libcloudapi.GetZone(token, libcloudapi.Zone()),
					libcloudapi.GetRRSet(token, libcloudapi.Zone(), libcloudapi.NewRRSetA(), false),
					libcloudapi.CreateRRSet(token, libcloudapi.Zone(), libcloudapi.NewRRSetA()),
				)
			} else {
				api.AppendHandlers(
					libdnsapi.GetZones(token, libdnsapi.Zones()),
					libdnsapi.GetRecords(token, libserver.ZoneID, nil),
					libdnsapi.PostRecord(token, libdnsapi.NewARecord()),
				)
			}

			Expect(doPlainRequest(ctx, server.URL+"/plain/update", username, password, url.Values{
				"hostname": []string{libserver.ARecordNameFull},
				"ip":       []string{libserver.AUpdated},
			})).To(Equal(http.StatusOK))

			Expect(api.ReceivedRequests()).To(HaveLen(3))
		},
			Entry("DNS API", false),
			Entry("Cloud API", true),
		)

		DescribeTable("updating an existing record", func(ctx context.Context, cloudAPI bool) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, cloudAPI)

			if cloudAPI {
				api.AppendHandlers(
					libcloudapi.GetZone(token, libcloudapi.Zone()),
					libcloudapi.GetRRSet(token, libcloudapi.Zone(), libcloudapi.ExistingRRSetA(), true),
					libcloudapi.ChangeRRSetTTL(token, libcloudapi.Zone(), libcloudapi.UpdatedRRSetA()),
					libcloudapi.SetRRSetRecords(token, libcloudapi.Zone(), libcloudapi.UpdatedRRSetA()),
				)
			} else {
				api.AppendHandlers(
					libdnsapi.GetZones(token, libdnsapi.Zones()),
					libdnsapi.GetRecords(token, libserver.ZoneID, libdnsapi.Records()),
					libdnsapi.PutRecord(token, libdnsapi.UpdatedARecord()),
				)
			}

			Expect(doPlainRequest(ctx, server.URL+"/plain/update", username, password, url.Values{
				"hostname": []string{libserver.ARecordNameFull},
				"ip":       []string{libserver.AUpdated},
			})).To(Equal(http.StatusOK))

			if cloudAPI {
				Expect(api.ReceivedRequests()).To(HaveLen(4))
			} else {
				Expect(api.ReceivedRequests()).To(HaveLen(3))
			}
		},
			Entry("DNS API", false),
			Entry("Cloud API", true),
		)
	})

	Context("should make no api calls and should fail", func() {
		AfterEach(func() {
			Expect(api.ReceivedRequests()).To((BeEmpty()))
		})

		DescribeTable("when hostname is missing", func(ctx context.Context, cloudAPI bool) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, cloudAPI)
			Expect(doPlainRequest(ctx, server.URL+"/plain/update", username, password, url.Values{
				"ip": []string{libserver.AUpdated},
			})).To(Equal(http.StatusBadRequest))
		},
			Entry("DNS API", false),
			Entry("Cloud API", true),
		)

		DescribeTable("when ip is missing", func(ctx context.Context, cloudAPI bool) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, cloudAPI)
			Expect(doPlainRequest(ctx, server.URL+"/plain/update", username, password, url.Values{
				"hostname": []string{libserver.ARecordNameFull},
			})).To(Equal(http.StatusBadRequest))
		},
			Entry("DNS API", false),
			Entry("Cloud API", true),
		)

		DescribeTable("when hostname is malformed", func(ctx context.Context, cloudAPI bool) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, cloudAPI)
			Expect(doPlainRequest(ctx, server.URL+"/plain/update", username, password, url.Values{
				"hostname": []string{libserver.TLD},
				"ip":       []string{libserver.AUpdated},
			})).To(Equal(http.StatusBadRequest))
		},
			Entry("DNS API", false),
			Entry("Cloud API", true),
		)

		DescribeTable("when access is denied", func(ctx context.Context, cloudAPI bool) {
			server = libserver.NewNoAllowedDomains(api.URL(), cloudAPI)
			Expect(doPlainRequest(ctx, server.URL+"/plain/update", username, password, url.Values{
				"hostname": []string{libserver.ARecordNameFull},
				"ip":       []string{libserver.AUpdated},
			})).To(Equal(http.StatusUnauthorized))
		},
			Entry("DNS API", false),
			Entry("Cloud API", true),
		)
	})
})

func doPlainRequest(ctx context.Context, serverURL, username, password string, data url.Values) int {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL, http.NoBody)
	Expect(err).ToNot(HaveOccurred())
	req.URL.RawQuery = data.Encode()
	req.SetBasicAuth(username, password)

	c := &http.Client{}
	res, err := c.Do(req)
	Expect(err).ToNot(HaveOccurred())
	Expect(res.Body.Close()).To(Succeed())

	return res.StatusCode
}

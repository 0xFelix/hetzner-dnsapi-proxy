package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/onsi/gomega/gstruct"

	"github.com/0xfelix/hetzner-dnsapi-proxy/tests/libcloudapi"
	"github.com/0xfelix/hetzner-dnsapi-proxy/tests/libserver"
)

var _ = Describe("AcmeDNS", func() {
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
		DescribeTable("creating a new record", func(ctx context.Context, subdomain string) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL)

			api.AppendHandlers(
				libcloudapi.GetZone(token, libcloudapi.Zone()),
				libcloudapi.GetRRSet(token, libcloudapi.Zone(), libcloudapi.NewRRSetTXT(), false),
				libcloudapi.CreateRRSet(token, libcloudapi.Zone(), libcloudapi.NewRRSetTXT()),
			)

			statusCode, resBody := doAcmeDNSRequest(ctx, server.URL+"/acmedns/update", username, password,
				map[string]string{
					"subdomain": subdomain,
					"txt":       libserver.TXTUpdated,
				},
			)
			Expect(statusCode).To(Equal(http.StatusOK))
			var resData map[string]string
			Expect(json.Unmarshal(resBody, &resData)).To(Succeed())
			Expect(resData).To(gstruct.MatchAllKeys(gstruct.Keys{
				"txt": Equal(libserver.TXTUpdated),
			}))
			Expect(api.ReceivedRequests()).To(HaveLen(3))
		},
			Entry("with prefix", libserver.TXTRecordNameFull),
			Entry("without prefix", libserver.TXTRecordNameNoPrefix),
		)

		DescribeTable("updating an existing record", func(ctx context.Context, subdomain string) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL)

			api.AppendHandlers(
				libcloudapi.GetZone(token, libcloudapi.Zone()),
				libcloudapi.GetRRSet(token, libcloudapi.Zone(), libcloudapi.ExistingRRSetTXT(), true),
				libcloudapi.ChangeRRSetTTL(token, libcloudapi.Zone(), libcloudapi.UpdatedRRSetTXT()),
				libcloudapi.SetRRSetRecords(token, libcloudapi.Zone(), libcloudapi.UpdatedRRSetTXT()),
			)

			statusCode, resBody := doAcmeDNSRequest(ctx, server.URL+"/acmedns/update", username, password,
				map[string]string{
					"subdomain": subdomain,
					"txt":       libserver.TXTUpdated,
				},
			)
			Expect(statusCode).To(Equal(http.StatusOK))
			var resData map[string]string
			Expect(json.Unmarshal(resBody, &resData)).To(Succeed())
			Expect(resData).To(gstruct.MatchAllKeys(gstruct.Keys{
				"txt": Equal(libserver.TXTUpdated),
			}))
			Expect(api.ReceivedRequests()).To(HaveLen(4))
		},
			Entry("with prefix", libserver.TXTRecordNameFull),
			Entry("without prefix", libserver.TXTRecordNameNoPrefix),
		)
	})

	Context("should make no api calls and should fail", func() {
		const subdomainTXTMissing = "subdomain or txt is missing\n"

		AfterEach(func() {
			Expect(api.ReceivedRequests()).To(BeEmpty())
		})

		It("when subdomain is missing", func(ctx context.Context) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL)
			statusCode, resBody := doAcmeDNSRequest(ctx, server.URL+"/acmedns/update", username, password,
				map[string]string{
					"txt": libserver.TXTUpdated,
				},
			)
			Expect(statusCode).To(Equal(http.StatusBadRequest))
			Expect(string(resBody)).To(Equal(subdomainTXTMissing))
		})

		It("when txt is missing", func(ctx context.Context) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL)
			statusCode, resBody := doAcmeDNSRequest(ctx, server.URL+"/acmedns/update", username, password,
				map[string]string{
					"subdomain": libserver.TXTRecordNameFull,
				},
			)
			Expect(statusCode).To(Equal(http.StatusBadRequest))
			Expect(string(resBody)).To(Equal(subdomainTXTMissing))
		})

		It("when subdomain is malformed", func(ctx context.Context) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL)
			statusCode, resBody := doAcmeDNSRequest(ctx, server.URL+"/acmedns/update", username, password,
				map[string]string{
					"subdomain": libserver.TLD,
					"txt":       libserver.TXTUpdated,
				},
			)
			Expect(statusCode).To(Equal(http.StatusBadRequest))
			Expect(string(resBody)).To(Equal("invalid fqdn: tld\n"))
		})

		DescribeTable("when access is denied", func(ctx context.Context, subdomain string) {
			server = libserver.NewNoAllowedDomains(api.URL())
			statusCode, resBody := doAcmeDNSRequest(ctx, server.URL+"/acmedns/update", username, password,
				map[string]string{
					"subdomain": subdomain,
					"txt":       libserver.TXTUpdated,
				},
			)
			Expect(statusCode).To(Equal(http.StatusUnauthorized))
			Expect(resBody).To(BeEmpty())
		},
			Entry("with prefix", libserver.TXTRecordNameFull),
			Entry("without prefix", libserver.TXTRecordNameNoPrefix),
		)
	})
})

func doAcmeDNSRequest(ctx context.Context, serverURL, username, password string, data map[string]string) (statusCode int, resBody []byte) {
	body, err := json.Marshal(data)
	Expect(err).ToNot(HaveOccurred())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, serverURL, bytes.NewReader(body))
	Expect(err).ToNot(HaveOccurred())
	req.Header.Add("X-Api-User", username)
	req.Header.Add("X-Api-Key", password)

	// Explicitly set Content-Type to empty instead of application/json.
	// Some AcmeDNS clients do not provide this header.
	req.Header.Add("Content-Type", "")

	c := &http.Client{}
	res, err := c.Do(req)
	Expect(err).ToNot(HaveOccurred())

	resBody, err = io.ReadAll(res.Body)
	Expect(err).ToNot(HaveOccurred())
	Expect(res.Body.Close()).To(Succeed())

	return res.StatusCode, resBody
}

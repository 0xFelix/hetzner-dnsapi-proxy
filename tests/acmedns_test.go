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
	"github.com/0xfelix/hetzner-dnsapi-proxy/tests/libdnsapi"
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
		DescribeTable("creating a new record", func(ctx context.Context, cloudAPI bool, subdomain string) {
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
			Entry("DNS API: with prefix", false, libserver.TXTRecordNameFull),
			Entry("DNS API: without prefix", false, libserver.TXTRecordNameNoPrefix),
			Entry("Cloud API: with prefix", true, libserver.TXTRecordNameFull),
			Entry("Cloud API: without prefix", true, libserver.TXTRecordNameNoPrefix),
		)

		DescribeTable("updating an existing record", func(ctx context.Context, cloudAPI bool, subdomain string) {
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
			if cloudAPI {
				Expect(api.ReceivedRequests()).To(HaveLen(4))
			} else {
				Expect(api.ReceivedRequests()).To(HaveLen(3))
			}
		},
			Entry("DNS API: with prefix", false, libserver.TXTRecordNameFull),
			Entry("DNS API: without prefix", false, libserver.TXTRecordNameNoPrefix),
			Entry("Cloud API: with prefix", true, libserver.TXTRecordNameFull),
			Entry("Cloud API: without prefix", true, libserver.TXTRecordNameNoPrefix),
		)
	})

	Context("should make no api calls and should fail", func() {
		const subdomainTXTMissing = "subdomain or txt is missing\n"

		AfterEach(func() {
			Expect(api.ReceivedRequests()).To(BeEmpty())
		})

		DescribeTable("when subdomain is missing", func(ctx context.Context, cloudAPI bool) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, cloudAPI)
			statusCode, resBody := doAcmeDNSRequest(ctx, server.URL+"/acmedns/update", username, password,
				map[string]string{
					"txt": libserver.TXTUpdated,
				},
			)
			Expect(statusCode).To(Equal(http.StatusBadRequest))
			Expect(string(resBody)).To(Equal(subdomainTXTMissing))
		},
			Entry("DNS API", false),
			Entry("Cloud API", true),
		)

		DescribeTable("when txt is missing", func(ctx context.Context, cloudAPI bool) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, cloudAPI)
			statusCode, resBody := doAcmeDNSRequest(ctx, server.URL+"/acmedns/update", username, password,
				map[string]string{
					"subdomain": libserver.TXTRecordNameFull,
				},
			)
			Expect(statusCode).To(Equal(http.StatusBadRequest))
			Expect(string(resBody)).To(Equal(subdomainTXTMissing))
		},
			Entry("DNS API", false),
			Entry("Cloud API", true),
		)

		DescribeTable("when subdomain is malformed", func(ctx context.Context, cloudAPI bool) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, cloudAPI)
			statusCode, resBody := doAcmeDNSRequest(ctx, server.URL+"/acmedns/update", username, password,
				map[string]string{
					"subdomain": libserver.TLD,
					"txt":       libserver.TXTUpdated,
				},
			)
			Expect(statusCode).To(Equal(http.StatusBadRequest))
			Expect(string(resBody)).To(Equal("invalid fqdn: tld\n"))
		},
			Entry("DNS API", false),
			Entry("Cloud API", true),
		)

		DescribeTable("when access is denied", func(ctx context.Context, subdomain string, cloudAPI bool) {
			server = libserver.NewNoAllowedDomains(api.URL(), cloudAPI)
			statusCode, resBody := doAcmeDNSRequest(ctx, server.URL+"/acmedns/update", username, password,
				map[string]string{
					"subdomain": subdomain,
					"txt":       libserver.TXTUpdated,
				},
			)
			Expect(statusCode).To(Equal(http.StatusUnauthorized))
			Expect(resBody).To(BeEmpty())
		},
			Entry("DNS API: with prefix", libserver.TXTRecordNameFull, false),
			Entry("DNS API: without prefix", libserver.TXTRecordNameNoPrefix, false),
			Entry("Cloud API: with prefix", libserver.TXTRecordNameFull, true),
			Entry("Cloud API: without prefix", libserver.TXTRecordNameNoPrefix, true),
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

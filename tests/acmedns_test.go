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

		DescribeTable("creating a new record", func(ctx context.Context, cloudAPI bool, subdomain string, appendHandlers func()) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, libserver.WithCloudAPI(cloudAPI))
			appendHandlers()

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
			Entry("DNS API: with prefix", false, libserver.TXTRecordNameFull, func() {
				api.AppendHandlers(
					libdnsapi.GetZones(token, libdnsapi.Zones()),
					libdnsapi.GetRecords(token, libserver.ZoneID, nil),
					libdnsapi.PostRecord(token, libdnsapi.NewTXTRecord()),
				)
			}),
			Entry("DNS API: without prefix", false, libserver.TXTRecordNameNoPrefix, func() {
				api.AppendHandlers(
					libdnsapi.GetZones(token, libdnsapi.Zones()),
					libdnsapi.GetRecords(token, libserver.ZoneID, nil),
					libdnsapi.PostRecord(token, libdnsapi.NewTXTRecord()),
				)
			}),
			Entry("Cloud API: with prefix", true, libserver.TXTRecordNameFull, func() {
				api.AppendHandlers(
					libcloudapi.GetZone(token, libdnsapi.Zones()[0]),
					libcloudapi.GetRRSetNotFound(token, libdnsapi.Zones()[0], libserver.TXTRecordName, "TXT"),
					libcloudapi.CreateRRSet(token, libdnsapi.Zones()[0], libdnsapi.NewTXTRecord()),
				)
			}),
			Entry("Cloud API: without prefix", true, libserver.TXTRecordNameNoPrefix, func() {
				api.AppendHandlers(
					libcloudapi.GetZone(token, libdnsapi.Zones()[0]),
					libcloudapi.GetRRSetNotFound(token, libdnsapi.Zones()[0], libserver.TXTRecordName, "TXT"),
					libcloudapi.CreateRRSet(token, libdnsapi.Zones()[0], libdnsapi.NewTXTRecord()),
				)
			}),
		)

		DescribeTable("updating an existing record", func(ctx context.Context, cloudAPI bool, subdomain string, appendHandlers func()) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, libserver.WithCloudAPI(cloudAPI))
			appendHandlers()

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
			Entry("DNS API: with prefix", false, libserver.TXTRecordNameFull, func() {
				api.AppendHandlers(
					libdnsapi.GetZones(token, libdnsapi.Zones()),
					libdnsapi.GetRecords(token, libserver.ZoneID, libdnsapi.Records()),
					libdnsapi.PutRecord(token, libdnsapi.UpdatedTXTRecord()),
				)
			}),
			Entry("DNS API: without prefix", false, libserver.TXTRecordNameNoPrefix, func() {
				api.AppendHandlers(
					libdnsapi.GetZones(token, libdnsapi.Zones()),
					libdnsapi.GetRecords(token, libserver.ZoneID, libdnsapi.Records()),
					libdnsapi.PutRecord(token, libdnsapi.UpdatedTXTRecord()),
				)
			}),
			Entry("Cloud API: with prefix", true, libserver.TXTRecordNameFull, func() {
				api.AppendHandlers(
					libcloudapi.GetZone(token, libdnsapi.Zones()[0]),
					libcloudapi.GetRRSet(token, libdnsapi.Zones()[0], libdnsapi.NewTXTRecord()),
					libcloudapi.ChangeRRSetTTL(token, libdnsapi.Zones()[0], libdnsapi.UpdatedTXTRecord()),
					libcloudapi.SetRRSetRecords(token, libdnsapi.Zones()[0], libdnsapi.UpdatedTXTRecord()),
				)
			}),
			Entry("Cloud API: without prefix", true, libserver.TXTRecordNameNoPrefix, func() {
				api.AppendHandlers(
					libcloudapi.GetZone(token, libdnsapi.Zones()[0]),
					libcloudapi.GetRRSet(token, libdnsapi.Zones()[0], libdnsapi.NewTXTRecord()),
					libcloudapi.ChangeRRSetTTL(token, libdnsapi.Zones()[0], libdnsapi.UpdatedTXTRecord()),
					libcloudapi.SetRRSetRecords(token, libdnsapi.Zones()[0], libdnsapi.UpdatedTXTRecord()),
				)
			}),
		)
	})

	Context("should make no api calls and should fail", func() {
		const subdomainTXTMissing = "subdomain or txt is missing\n"

		AfterEach(func() {
			Expect(api.ReceivedRequests()).To(BeEmpty())
		})

		DescribeTable("for both APIs", func(ctx context.Context, cloudAPI bool) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, libserver.WithCloudAPI(cloudAPI))

			statusCode, resBody := doAcmeDNSRequest(ctx, server.URL+"/acmedns/update", username, password,
				map[string]string{
					"txt": libserver.TXTUpdated,
				},
			)
			Expect(statusCode).To(Equal(http.StatusBadRequest))
			Expect(string(resBody)).To(Equal(subdomainTXTMissing))

			statusCode, resBody = doAcmeDNSRequest(ctx, server.URL+"/acmedns/update", username, password,
				map[string]string{
					"subdomain": libserver.TXTRecordNameFull,
				},
			)
			Expect(statusCode).To(Equal(http.StatusBadRequest))
			Expect(string(resBody)).To(Equal(subdomainTXTMissing))

			statusCode, resBody = doAcmeDNSRequest(ctx, server.URL+"/acmedns/update", username, password,
				map[string]string{
					"subdomain": libserver.TLD,
					"txt":       libserver.TXTUpdated,
				},
			)
			Expect(statusCode).To(Equal(http.StatusBadRequest))
			Expect(string(resBody)).To(Equal("invalid fqdn: tld\n"))

			server.Close()
			server = libserver.NewNoAllowedDomains(api.URL(), libserver.WithCloudAPI(cloudAPI))
			statusCode, resBody = doAcmeDNSRequest(ctx, server.URL+"/acmedns/update", username, password,
				map[string]string{
					"subdomain": libserver.TXTRecordNameFull,
					"txt":       libserver.TXTUpdated,
				},
			)
			Expect(statusCode).To(Equal(http.StatusUnauthorized))
			Expect(resBody).To(BeEmpty())
		},
			Entry("DNS API", false),
			Entry("Cloud API", true),
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

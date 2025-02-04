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

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/hetzner"
	"github.com/0xfelix/hetzner-dnsapi-proxy/tests/libapi"
	"github.com/0xfelix/hetzner-dnsapi-proxy/tests/libserver"
)

var _ = Describe("DirectAdmin", func() {
	var (
		api    *ghttp.Server
		server *httptest.Server
		token  string

		statusOK = url.Values{
			"error": []string{"0"},
			"text":  []string{"OK"},
		}
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

		DescribeTable("creating a new", func(ctx context.Context, domain, name, recordType, value string, record func() hetzner.Record) {
			api.AppendHandlers(
				libapi.GetZones(token, libapi.Zones()),
				libapi.GetRecords(token, libapi.ZoneID, nil),
				libapi.PostRecord(token, record()),
			)

			statusCode, resData := doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_DNS_CONTROL", url.Values{
				"domain": []string{domain},
				"action": []string{"add"},
				"type":   []string{recordType},
				"name":   []string{name},
				"value":  []string{value},
			})
			Expect(statusCode).To(Equal(http.StatusOK))
			Expect(resData).To(Equal(statusOK))
		},
			Entry("A record with fqdn in domain",
				libapi.ARecordNameFull, "", libapi.RecordTypeA, libapi.AUpdated, libapi.NewARecord),
			Entry("A record with fqdn from name and domain",
				libapi.ZoneName, libapi.ARecordName, libapi.RecordTypeA, libapi.AUpdated, libapi.NewARecord),
			Entry("TXT record with fqdn in domain",
				libapi.TXTRecordNameFull, "", libapi.RecordTypeTXT, libapi.TXTUpdated, libapi.NewTXTRecord),
			Entry("TXT recordd with fqdn from name and domain",
				libapi.ZoneName, libapi.TXTRecordName, libapi.RecordTypeTXT, libapi.TXTUpdated, libapi.NewTXTRecord),
		)

		DescribeTable("updating an existing", func(ctx context.Context, domain, name, recordType, value string, record func() hetzner.Record) {
			api.AppendHandlers(
				libapi.GetZones(token, libapi.Zones()),
				libapi.GetRecords(token, libapi.ZoneID, libapi.Records()),
				libapi.PutRecord(token, record()),
			)

			statusCode, resData := doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_DNS_CONTROL", url.Values{
				"domain": []string{domain},
				"action": []string{"add"},
				"type":   []string{recordType},
				"name":   []string{name},
				"value":  []string{value},
			})
			Expect(statusCode).To(Equal(http.StatusOK))
			Expect(resData).To(Equal(statusOK))
		},
			Entry("A record with fqdn in domain",
				libapi.ARecordNameFull, "", libapi.RecordTypeA, libapi.AUpdated, libapi.UpdatedARecord),
			Entry("A record with fqdn from name and domain",
				libapi.ZoneName, libapi.ARecordName, libapi.RecordTypeA, libapi.AUpdated, libapi.UpdatedARecord),
			Entry("TXT record with fqdn in domain",
				libapi.TXTRecordNameFull, "", libapi.RecordTypeTXT, libapi.TXTUpdated, libapi.UpdatedTXTRecord),
			Entry("TXT recordd with fqdn from name and domain",
				libapi.ZoneName, libapi.TXTRecordName, libapi.RecordTypeTXT, libapi.TXTUpdated, libapi.UpdatedTXTRecord),
		)
	})

	Context("should make no api calls and", func() {
		AfterEach(func() {
			Expect(api.ReceivedRequests()).To(HaveLen(0))
		})

		DescribeTable("should succeed on action action than add with", func(ctx context.Context, action string) {
			statusCode, resData := doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_DNS_CONTROL", url.Values{
				"domain": []string{libapi.ARecordNameFull},
				"action": []string{action},
			})
			Expect(statusCode).To(Equal(http.StatusOK))
			Expect(resData).To(Equal(statusOK))
		},
			Entry("delete", "delete"),
			Entry("update", "update"),
			Entry("something", "something"),
		)

		It("should return allowed domains", func(ctx context.Context) {
			statusCode, resData := doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_SHOW_DOMAINS", nil)
			Expect(statusCode).To(Equal(http.StatusOK))
			Expect(resData).To(Equal(url.Values{
				"list": []string{"*"},
			}))
		})

		It("should succeed on calls to CMD_API_DOMAIN_POINTER", func(ctx context.Context) {
			statusCode, resData := doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_DOMAIN_POINTER", url.Values{
				"domain": []string{libapi.ZoneName},
			})
			Expect(statusCode).To(Equal(http.StatusOK))
			Expect(resData).To(BeEmpty())
		})

		Context("should fail", func() {
			It("when domain is missing", func(ctx context.Context) {
				statusCode, resData := doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_DNS_CONTROL", url.Values{
					"action": []string{"add"},
					"type":   []string{libapi.RecordTypeTXT},
					"name":   []string{libapi.TXTRecordName},
					"value":  []string{libapi.TXTUpdated},
				})
				Expect(statusCode).To(Equal(http.StatusBadRequest))
				Expect(resData).To(BeEmpty())
			})

			It("when action is missing", func(ctx context.Context) {
				statusCode, resData := doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_DNS_CONTROL", url.Values{
					"domain": []string{libapi.ZoneName},
					"type":   []string{libapi.RecordTypeTXT},
					"name":   []string{libapi.TXTRecordName},
					"value":  []string{libapi.TXTUpdated},
				})
				Expect(statusCode).To(Equal(http.StatusBadRequest))
				Expect(resData).To(BeEmpty())
			})

			It("when type is not A or TXT", func(ctx context.Context) {
				statusCode, resData := doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_DNS_CONTROL", url.Values{
					"action": []string{"add"},
					"domain": []string{libapi.ZoneName},
					"type":   []string{"madeup"},
					"name":   []string{libapi.TXTRecordName},
					"value":  []string{libapi.TXTUpdated},
				})
				Expect(statusCode).To(Equal(http.StatusBadRequest))
				Expect(resData).To(BeEmpty())
			})

			It("when domain is malformed and name is empty", func(ctx context.Context) {
				statusCode, resData := doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_DNS_CONTROL", url.Values{
					"action": []string{"add"},
					"domain": []string{libapi.TLD},
					"type":   []string{libapi.RecordTypeTXT},
					"name":   []string{""},
					"value":  []string{libapi.TXTUpdated},
				})
				Expect(statusCode).To(Equal(http.StatusBadRequest))
				Expect(resData).To(BeEmpty())
			})

			DescribeTable("when access is denied", func(ctx context.Context, domain, name, recordType string) {
				server = libserver.NewNoAllowedDomains(api.URL())
				statusCode, resData := doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_DNS_CONTROL", url.Values{
					"action": []string{"add"},
					"domain": []string{domain},
					"type":   []string{recordType},
					"name":   []string{name},
					"value":  []string{"something"},
				})
				Expect(statusCode).To(Equal(http.StatusForbidden))
				Expect(resData).To(BeEmpty())
			},
				Entry("A record with fqdn in domain", libapi.ARecordNameFull, "", libapi.RecordTypeA),
				Entry("A record with fqdn from name and domain", libapi.ZoneName, libapi.ARecordName, libapi.RecordTypeA),
				Entry("TXT record with fqdn in domain", libapi.TXTRecordNameFull, "", libapi.RecordTypeTXT),
				Entry("TXT recordd with fqdn from name and domain", libapi.ZoneName, libapi.TXTRecordName, libapi.RecordTypeTXT),
			)
		})
	})
})

func doDirectAdminRequest(ctx context.Context, serverURL string, data url.Values) (int, url.Values) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL, http.NoBody)
	Expect(err).ToNot(HaveOccurred())
	req.URL.RawQuery = data.Encode()

	c := &http.Client{}
	res, err := c.Do(req)
	Expect(err).ToNot(HaveOccurred())

	resBody, err := io.ReadAll(res.Body)
	Expect(err).ToNot(HaveOccurred())
	Expect(res.Body.Close()).To(Succeed())

	resData, err := url.ParseQuery(string(resBody))
	Expect(err).ToNot(HaveOccurred())

	return res.StatusCode, resData
}

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
	"github.com/0xfelix/hetzner-dnsapi-proxy/tests/libcloudapi"
	"github.com/0xfelix/hetzner-dnsapi-proxy/tests/libdnsapi"
	"github.com/0xfelix/hetzner-dnsapi-proxy/tests/libserver"
)

var _ = Describe("DirectAdmin", func() {
	var (
		api      *ghttp.Server
		server   *httptest.Server
		token    string
		username string
		password string

		statusOK = url.Values{
			"error": []string{"0"},
			"text":  []string{"OK"},
		}
	)

	BeforeEach(func() {
		api = ghttp.NewServer()
	})

	AfterEach(func() {
		server.Close()
		api.Close()
	})

	Context("should succeed", func() {
		DescribeTable("creating a new", func(ctx context.Context, cloudAPI bool, expectedRequests int, domain, name, recordType, value string, record func() hetzner.Record, prepareHandlers func()) {
			server, token, username, password = libserver.New(api.URL(), libdnsapi.DefaultTTL, libserver.WithCloudAPI(cloudAPI))
			prepareHandlers()

			statusCode, resData := doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_DNS_CONTROL", username, password,
				url.Values{
					"domain": []string{domain},
					"action": []string{"add"},
					"type":   []string{recordType},
					"name":   []string{name},
					"value":  []string{value},
				},
			)
			Expect(statusCode).To(Equal(http.StatusOK))
			values, err := url.ParseQuery(resData)
			Expect(err).ToNot(HaveOccurred())
			Expect(values).To(Equal(statusOK))
			Expect(api.ReceivedRequests()).To(HaveLen(expectedRequests))
		},
			Entry("DNS API: A record with fqdn in domain", false, 3,
				libdnsapi.ARecordNameFull, "", libdnsapi.RecordTypeA, libdnsapi.AUpdated, libdnsapi.NewARecord, func() {
					api.AppendHandlers(
						libdnsapi.GetZones(token, libdnsapi.Zones()),
						libdnsapi.GetRecords(token, libdnsapi.ZoneID, nil),
						libdnsapi.PostRecord(token, libdnsapi.NewARecord()),
					)
				}),
			Entry("DNS API: A record with fqdn from name and domain", false, 3,
				libdnsapi.ZoneName, libdnsapi.ARecordName, libdnsapi.RecordTypeA, libdnsapi.AUpdated, libdnsapi.NewARecord, func() {
					api.AppendHandlers(
						libdnsapi.GetZones(token, libdnsapi.Zones()),
						libdnsapi.GetRecords(token, libdnsapi.ZoneID, nil),
						libdnsapi.PostRecord(token, libdnsapi.NewARecord()),
					)
				}),
			Entry("DNS API: TXT record with fqdn in domain", false, 3,
				libdnsapi.TXTRecordNameFull, "", libdnsapi.RecordTypeTXT, libdnsapi.TXTUpdated, libdnsapi.NewTXTRecord, func() {
					api.AppendHandlers(
						libdnsapi.GetZones(token, libdnsapi.Zones()),
						libdnsapi.GetRecords(token, libdnsapi.ZoneID, nil),
						libdnsapi.PostRecord(token, libdnsapi.NewTXTRecord()),
					)
				}),
			Entry("DNS API: TXT record with fqdn from name and domain", false, 3,
				libdnsapi.ZoneName, libdnsapi.TXTRecordName, libdnsapi.RecordTypeTXT, libdnsapi.TXTUpdated, libdnsapi.NewTXTRecord, func() {
					api.AppendHandlers(
						libdnsapi.GetZones(token, libdnsapi.Zones()),
						libdnsapi.GetRecords(token, libdnsapi.ZoneID, nil),
						libdnsapi.PostRecord(token, libdnsapi.NewTXTRecord()),
					)
				}),
			Entry("Cloud API: A record with fqdn in domain", true, 3,
				libdnsapi.ARecordNameFull, "", libdnsapi.RecordTypeA, libdnsapi.AUpdated, libdnsapi.NewARecord, func() {
					api.AppendHandlers(
						libcloudapi.GetZone(token, libdnsapi.Zones()[0]),
						libcloudapi.GetRRSetNotFound(token, libdnsapi.Zones()[0], libdnsapi.ARecordName, "A"),
						libcloudapi.CreateRRSet(token, libdnsapi.Zones()[0], libdnsapi.NewARecord()),
					)
				}),
			Entry("Cloud API: A record with fqdn from name and domain", true, 3,
				libdnsapi.ZoneName, libdnsapi.ARecordName, libdnsapi.RecordTypeA, libdnsapi.AUpdated, libdnsapi.NewARecord, func() {
					api.AppendHandlers(
						libcloudapi.GetZone(token, libdnsapi.Zones()[0]),
						libcloudapi.GetRRSetNotFound(token, libdnsapi.Zones()[0], libdnsapi.ARecordName, "A"),
						libcloudapi.CreateRRSet(token, libdnsapi.Zones()[0], libdnsapi.NewARecord()),
					)
				}),
			Entry("Cloud API: TXT record with fqdn in domain", true, 3,
				libdnsapi.TXTRecordNameFull, "", libdnsapi.RecordTypeTXT, libdnsapi.TXTUpdated, libdnsapi.NewTXTRecord, func() {
					api.AppendHandlers(
						libcloudapi.GetZone(token, libdnsapi.Zones()[0]),
						libcloudapi.GetRRSetNotFound(token, libdnsapi.Zones()[0], libdnsapi.TXTRecordName, "TXT"),
						libcloudapi.CreateRRSet(token, libdnsapi.Zones()[0], libdnsapi.NewTXTRecord()),
					)
				}),
			Entry("Cloud API: TXT record with fqdn from name and domain", true, 3,
				libdnsapi.ZoneName, libdnsapi.TXTRecordName, libdnsapi.RecordTypeTXT, libdnsapi.TXTUpdated, libdnsapi.NewTXTRecord, func() {
					api.AppendHandlers(
						libcloudapi.GetZone(token, libdnsapi.Zones()[0]),
						libcloudapi.GetRRSetNotFound(token, libdnsapi.Zones()[0], libdnsapi.TXTRecordName, "TXT"),
						libcloudapi.CreateRRSet(token, libdnsapi.Zones()[0], libdnsapi.NewTXTRecord()),
					)
				}),
		)

		DescribeTable("updating an existing", func(ctx context.Context, cloudAPI bool, expectedRequests int, domain, name, recordType, value string, record func() hetzner.Record, prepareHandlers func()) {
			server, token, username, password = libserver.New(api.URL(), libdnsapi.DefaultTTL, libserver.WithCloudAPI(cloudAPI))
			prepareHandlers()

			statusCode, resData := doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_DNS_CONTROL", username, password,
				url.Values{
					"domain": []string{domain},
					"action": []string{"add"},
					"type":   []string{recordType},
					"name":   []string{name},
					"value":  []string{value},
				},
			)
			Expect(statusCode).To(Equal(http.StatusOK))
			values, err := url.ParseQuery(resData)
			Expect(err).ToNot(HaveOccurred())
			Expect(values).To(Equal(statusOK))
			Expect(api.ReceivedRequests()).To(HaveLen(expectedRequests))
		},
			Entry("DNS API: A record with fqdn in domain", false, 3,
				libdnsapi.ARecordNameFull, "", libdnsapi.RecordTypeA, libdnsapi.AUpdated, libdnsapi.UpdatedARecord, func() {
					api.AppendHandlers(
						libdnsapi.GetZones(token, libdnsapi.Zones()),
						libdnsapi.GetRecords(token, libdnsapi.ZoneID, libdnsapi.Records()),
						libdnsapi.PutRecord(token, libdnsapi.UpdatedARecord()),
					)
				}),
			Entry("DNS API: A record with fqdn from name and domain", false, 3,
				libdnsapi.ZoneName, libdnsapi.ARecordName, libdnsapi.RecordTypeA, libdnsapi.AUpdated, libdnsapi.UpdatedARecord, func() {
					api.AppendHandlers(
						libdnsapi.GetZones(token, libdnsapi.Zones()),
						libdnsapi.GetRecords(token, libdnsapi.ZoneID, libdnsapi.Records()),
						libdnsapi.PutRecord(token, libdnsapi.UpdatedARecord()),
					)
				}),
			Entry("DNS API: TXT record with fqdn in domain", false, 3,
				libdnsapi.TXTRecordNameFull, "", libdnsapi.RecordTypeTXT, libdnsapi.TXTUpdated, libdnsapi.UpdatedTXTRecord, func() {
					api.AppendHandlers(
						libdnsapi.GetZones(token, libdnsapi.Zones()),
						libdnsapi.GetRecords(token, libdnsapi.ZoneID, libdnsapi.Records()),
						libdnsapi.PutRecord(token, libdnsapi.UpdatedTXTRecord()),
					)
				}),
			Entry("DNS API: TXT record with fqdn from name and domain", false, 3,
				libdnsapi.ZoneName, libdnsapi.TXTRecordName, libdnsapi.RecordTypeTXT, libdnsapi.TXTUpdated, libdnsapi.UpdatedTXTRecord, func() {
					api.AppendHandlers(
						libdnsapi.GetZones(token, libdnsapi.Zones()),
						libdnsapi.GetRecords(token, libdnsapi.ZoneID, libdnsapi.Records()),
						libdnsapi.PutRecord(token, libdnsapi.UpdatedTXTRecord()),
					)
				}),
			Entry("Cloud API: A record with fqdn in domain", true, 4,
				libdnsapi.ARecordNameFull, "", libdnsapi.RecordTypeA, libdnsapi.AUpdated, libdnsapi.UpdatedARecord, func() {
					api.AppendHandlers(
						libcloudapi.GetZone(token, libdnsapi.Zones()[0]),
						libcloudapi.GetRRSet(token, libdnsapi.Zones()[0], libdnsapi.Records()[0]),
						libcloudapi.ChangeRRSetTTL(token, libdnsapi.Zones()[0], libdnsapi.UpdatedARecord()),
						libcloudapi.SetRRSetRecords(token, libdnsapi.Zones()[0], libdnsapi.UpdatedARecord()),
					)
				}),
			Entry("Cloud API: A record with fqdn from name and domain", true, 4,
				libdnsapi.ZoneName, libdnsapi.ARecordName, libdnsapi.RecordTypeA, libdnsapi.AUpdated, libdnsapi.UpdatedARecord, func() {
					api.AppendHandlers(
						libcloudapi.GetZone(token, libdnsapi.Zones()[0]),
						libcloudapi.GetRRSet(token, libdnsapi.Zones()[0], libdnsapi.Records()[0]),
						libcloudapi.ChangeRRSetTTL(token, libdnsapi.Zones()[0], libdnsapi.UpdatedARecord()),
						libcloudapi.SetRRSetRecords(token, libdnsapi.Zones()[0], libdnsapi.UpdatedARecord()),
					)
				}),
			Entry("Cloud API: TXT record with fqdn in domain", true, 4,
				libdnsapi.TXTRecordNameFull, "", libdnsapi.RecordTypeTXT, libdnsapi.TXTUpdated, libdnsapi.UpdatedTXTRecord, func() {
					api.AppendHandlers(
						libcloudapi.GetZone(token, libdnsapi.Zones()[0]),
						libcloudapi.GetRRSet(token, libdnsapi.Zones()[0], libdnsapi.Records()[1]),
						libcloudapi.ChangeRRSetTTL(token, libdnsapi.Zones()[0], libdnsapi.UpdatedTXTRecord()),
						libcloudapi.SetRRSetRecords(token, libdnsapi.Zones()[0], libdnsapi.UpdatedTXTRecord()),
					)
				}),
			Entry("Cloud API: TXT record with fqdn from name and domain", true, 4,
				libdnsapi.ZoneName, libdnsapi.TXTRecordName, libdnsapi.RecordTypeTXT, libdnsapi.TXTUpdated, libdnsapi.UpdatedTXTRecord, func() {
					api.AppendHandlers(
						libcloudapi.GetZone(token, libdnsapi.Zones()[0]),
						libcloudapi.GetRRSet(token, libdnsapi.Zones()[0], libdnsapi.Records()[1]),
						libcloudapi.ChangeRRSetTTL(token, libdnsapi.Zones()[0], libdnsapi.UpdatedTXTRecord()),
						libcloudapi.SetRRSetRecords(token, libdnsapi.Zones()[0], libdnsapi.UpdatedTXTRecord()),
					)
				}),
		)
	})

	Context("should make no api calls and", func() {
		BeforeEach(func() {
			server, token, username, password = libserver.New(api.URL(), libdnsapi.DefaultTTL)
		})

		AfterEach(func() {
			Expect(api.ReceivedRequests()).To(BeEmpty())
		})

		DescribeTable("should succeed on action action than add with", func(ctx context.Context, action string) {
			statusCode, resData := doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_DNS_CONTROL", username, password,
				url.Values{
					"domain": []string{libdnsapi.ARecordNameFull},
					"action": []string{action},
				},
			)
			Expect(statusCode).To(Equal(http.StatusOK))
			values, err := url.ParseQuery(resData)
			Expect(err).ToNot(HaveOccurred())
			Expect(values).To(Equal(statusOK))
		},
			Entry("delete", "delete"),
			Entry("update", "update"),
			Entry("something", "something"),
		)

		It("should return allowed domains", func(ctx context.Context) {
			statusCode, resData := doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_SHOW_DOMAINS", username, password, nil)
			Expect(statusCode).To(Equal(http.StatusOK))
			values, err := url.ParseQuery(resData)
			Expect(err).ToNot(HaveOccurred())
			Expect(values).To(Equal(url.Values{
				"list": []string{"*"},
			}))
		})

		It("should succeed on calls to CMD_API_DOMAIN_POINTER", func(ctx context.Context) {
			statusCode, resData := doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_DOMAIN_POINTER", username, password,
				url.Values{
					"domain": []string{libdnsapi.ZoneName},
				},
			)
			Expect(statusCode).To(Equal(http.StatusOK))
			Expect(resData).To(BeEmpty())
		})

		Context("should fail", func() {
			const domainActionMissing = "domain or action is missing\n"

			It("when domain is missing", func(ctx context.Context) {
				statusCode, resData := doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_DNS_CONTROL", username, password,
					url.Values{
						"action": []string{"add"},
						"type":   []string{libdnsapi.RecordTypeTXT},
						"name":   []string{libdnsapi.TXTRecordName},
						"value":  []string{libdnsapi.TXTUpdated},
					},
				)
				Expect(statusCode).To(Equal(http.StatusBadRequest))
				Expect(resData).To(Equal(domainActionMissing))
			})

			It("when action is missing", func(ctx context.Context) {
				statusCode, resData := doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_DNS_CONTROL", username, password,
					url.Values{
						"domain": []string{libdnsapi.ZoneName},
						"type":   []string{libdnsapi.RecordTypeTXT},
						"name":   []string{libdnsapi.TXTRecordName},
						"value":  []string{libdnsapi.TXTUpdated},
					},
				)
				Expect(statusCode).To(Equal(http.StatusBadRequest))
				Expect(resData).To(Equal(domainActionMissing))
			})

			It("when type is not A or TXT", func(ctx context.Context) {
				statusCode, resData := doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_DNS_CONTROL", username, password,
					url.Values{
						"action": []string{"add"},
						"domain": []string{libdnsapi.ZoneName},
						"type":   []string{"madeup"},
						"name":   []string{libdnsapi.TXTRecordName},
						"value":  []string{libdnsapi.TXTUpdated},
					},
				)
				Expect(statusCode).To(Equal(http.StatusBadRequest))
				Expect(resData).To(Equal("type can only be A or TXT\n"))
			})

			It("when domain is malformed and name is empty", func(ctx context.Context) {
				statusCode, resData := doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_DNS_CONTROL", username, password,
					url.Values{
						"action": []string{"add"},
						"domain": []string{libdnsapi.TLD},
						"type":   []string{libdnsapi.RecordTypeTXT},
						"name":   []string{""},
						"value":  []string{libdnsapi.TXTUpdated},
					},
				)
				Expect(statusCode).To(Equal(http.StatusBadRequest))
				Expect(resData).To(Equal("invalid fqdn: tld\n"))
			})

			DescribeTable("when access is denied", func(ctx context.Context, domain, name, recordType string) {
				server.Close()
				server = libserver.NewNoAllowedDomains(api.URL())
				statusCode, resData := doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_DNS_CONTROL", username, password,
					url.Values{
						"action": []string{"add"},
						"domain": []string{domain},
						"type":   []string{recordType},
						"name":   []string{name},
						"value":  []string{"something"},
					},
				)
				Expect(statusCode).To(Equal(http.StatusUnauthorized))
				Expect(resData).To(BeEmpty())
			},
				Entry("A record with fqdn in domain", libdnsapi.ARecordNameFull, "", libdnsapi.RecordTypeA),
				Entry("A record with fqdn from name and domain", libdnsapi.ZoneName, libdnsapi.ARecordName, libdnsapi.RecordTypeA),
				Entry("TXT record with fqdn in domain", libdnsapi.TXTRecordNameFull, "", libdnsapi.RecordTypeTXT),
				Entry("TXT record with fqdn from name and domain", libdnsapi.ZoneName, libdnsapi.TXTRecordName, libdnsapi.RecordTypeTXT),
			)
		})
	})
})

func doDirectAdminRequest(ctx context.Context, serverURL, username, password string, data url.Values) (statusCode int, resData string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL, http.NoBody)
	Expect(err).ToNot(HaveOccurred())
	req.SetBasicAuth(username, password)
	req.URL.RawQuery = data.Encode()

	c := &http.Client{}
	res, err := c.Do(req)
	Expect(err).ToNot(HaveOccurred())

	resBody, err := io.ReadAll(res.Body)
	Expect(err).ToNot(HaveOccurred())
	Expect(res.Body.Close()).To(Succeed())

	return res.StatusCode, string(resBody)
}

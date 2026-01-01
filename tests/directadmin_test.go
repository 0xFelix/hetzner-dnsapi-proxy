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
		DescribeTable("creating a new", func(ctx context.Context, cloudAPI bool, domain, name, recordType, value string, record func() hetzner.Record, appendHandlers func()) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, libserver.WithCloudAPI(cloudAPI))
			appendHandlers()

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
			Expect(api.ReceivedRequests()).To(HaveLen(3))
		},
			Entry("DNS API: A record with fqdn in domain", false,
				libserver.ARecordNameFull, "", libserver.RecordTypeA, libserver.AUpdated, libdnsapi.NewARecord, func() {
					api.AppendHandlers(
						libdnsapi.GetZones(token, libdnsapi.Zones()),
						libdnsapi.GetRecords(token, libserver.ZoneID, nil),
						libdnsapi.PostRecord(token, libdnsapi.NewARecord()),
					)
				}),
			Entry("DNS API: A record with fqdn from name and domain", false,
				libserver.ZoneName, libserver.ARecordName, libserver.RecordTypeA, libserver.AUpdated, libdnsapi.NewARecord, func() {
					api.AppendHandlers(
						libdnsapi.GetZones(token, libdnsapi.Zones()),
						libdnsapi.GetRecords(token, libserver.ZoneID, nil),
						libdnsapi.PostRecord(token, libdnsapi.NewARecord()),
					)
				}),
			Entry("DNS API: TXT record with fqdn in domain", false,
				libserver.TXTRecordNameFull, "", libserver.RecordTypeTXT, libserver.TXTUpdated, libdnsapi.NewTXTRecord, func() {
					api.AppendHandlers(
						libdnsapi.GetZones(token, libdnsapi.Zones()),
						libdnsapi.GetRecords(token, libserver.ZoneID, nil),
						libdnsapi.PostRecord(token, libdnsapi.NewTXTRecord()),
					)
				}),
			Entry("DNS API: TXT record with fqdn from name and domain", false,
				libserver.ZoneName, libserver.TXTRecordName, libserver.RecordTypeTXT, libserver.TXTUpdated, libdnsapi.NewTXTRecord, func() {
					api.AppendHandlers(
						libdnsapi.GetZones(token, libdnsapi.Zones()),
						libdnsapi.GetRecords(token, libserver.ZoneID, nil),
						libdnsapi.PostRecord(token, libdnsapi.NewTXTRecord()),
					)
				}),
			Entry("Cloud API: A record with fqdn in domain", true,
				libserver.ARecordNameFull, "", libserver.RecordTypeA, libserver.AUpdated, libdnsapi.NewARecord, func() {
					api.AppendHandlers(
						libcloudapi.GetZone(token, libdnsapi.Zones()[0]),
						libcloudapi.GetRRSetNotFound(token, libdnsapi.Zones()[0], libserver.ARecordName, "A"),
						libcloudapi.CreateRRSet(token, libdnsapi.Zones()[0], libdnsapi.NewARecord()),
					)
				}),
			Entry("Cloud API: A record with fqdn from name and domain", true,
				libserver.ZoneName, libserver.ARecordName, libserver.RecordTypeA, libserver.AUpdated, libdnsapi.NewARecord, func() {
					api.AppendHandlers(
						libcloudapi.GetZone(token, libdnsapi.Zones()[0]),
						libcloudapi.GetRRSetNotFound(token, libdnsapi.Zones()[0], libserver.ARecordName, "A"),
						libcloudapi.CreateRRSet(token, libdnsapi.Zones()[0], libdnsapi.NewARecord()),
					)
				}),
			Entry("Cloud API: TXT record with fqdn in domain", true,
				libserver.TXTRecordNameFull, "", libserver.RecordTypeTXT, libserver.TXTUpdated, libdnsapi.NewTXTRecord, func() {
					api.AppendHandlers(
						libcloudapi.GetZone(token, libdnsapi.Zones()[0]),
						libcloudapi.GetRRSetNotFound(token, libdnsapi.Zones()[0], libserver.TXTRecordName, "TXT"),
						libcloudapi.CreateRRSet(token, libdnsapi.Zones()[0], libdnsapi.NewTXTRecord()),
					)
				}),
			Entry("Cloud API: TXT record with fqdn from name and domain", true,
				libserver.ZoneName, libserver.TXTRecordName, libserver.RecordTypeTXT, libserver.TXTUpdated, libdnsapi.NewTXTRecord, func() {
					api.AppendHandlers(
						libcloudapi.GetZone(token, libdnsapi.Zones()[0]),
						libcloudapi.GetRRSetNotFound(token, libdnsapi.Zones()[0], libserver.TXTRecordName, "TXT"),
						libcloudapi.CreateRRSet(token, libdnsapi.Zones()[0], libdnsapi.NewTXTRecord()),
					)
				}),
		)

		DescribeTable("updating an existing", func(ctx context.Context, cloudAPI bool, domain, name, recordType, value string, record func() hetzner.Record, appendHandlers func()) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, libserver.WithCloudAPI(cloudAPI))
			appendHandlers()

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
			if cloudAPI {
				Expect(api.ReceivedRequests()).To(HaveLen(4))
			} else {
				Expect(api.ReceivedRequests()).To(HaveLen(3))
			}
		},
			Entry("DNS API: A record with fqdn in domain", false,
				libserver.ARecordNameFull, "", libserver.RecordTypeA, libserver.AUpdated, libdnsapi.UpdatedARecord, func() {
					api.AppendHandlers(
						libdnsapi.GetZones(token, libdnsapi.Zones()),
						libdnsapi.GetRecords(token, libserver.ZoneID, libdnsapi.Records()),
						libdnsapi.PutRecord(token, libdnsapi.UpdatedARecord()),
					)
				}),
			Entry("DNS API: A record with fqdn from name and domain", false,
				libserver.ZoneName, libserver.ARecordName, libserver.RecordTypeA, libserver.AUpdated, libdnsapi.UpdatedARecord, func() {
					api.AppendHandlers(
						libdnsapi.GetZones(token, libdnsapi.Zones()),
						libdnsapi.GetRecords(token, libserver.ZoneID, libdnsapi.Records()),
						libdnsapi.PutRecord(token, libdnsapi.UpdatedARecord()),
					)
				}),
			Entry("DNS API: TXT record with fqdn in domain", false,
				libserver.TXTRecordNameFull, "", libserver.RecordTypeTXT, libserver.TXTUpdated, libdnsapi.UpdatedTXTRecord, func() {
					api.AppendHandlers(
						libdnsapi.GetZones(token, libdnsapi.Zones()),
						libdnsapi.GetRecords(token, libserver.ZoneID, libdnsapi.Records()),
						libdnsapi.PutRecord(token, libdnsapi.UpdatedTXTRecord()),
					)
				}),
			Entry("DNS API: TXT record with fqdn from name and domain", false,
				libserver.ZoneName, libserver.TXTRecordName, libserver.RecordTypeTXT, libserver.TXTUpdated, libdnsapi.UpdatedTXTRecord, func() {
					api.AppendHandlers(
						libdnsapi.GetZones(token, libdnsapi.Zones()),
						libdnsapi.GetRecords(token, libserver.ZoneID, libdnsapi.Records()),
						libdnsapi.PutRecord(token, libdnsapi.UpdatedTXTRecord()),
					)
				}),
			Entry("Cloud API: A record with fqdn in domain", true,
				libserver.ARecordNameFull, "", libserver.RecordTypeA, libserver.AUpdated, libdnsapi.UpdatedARecord, func() {
					api.AppendHandlers(
						libcloudapi.GetZone(token, libdnsapi.Zones()[0]),
						libcloudapi.GetRRSet(token, libdnsapi.Zones()[0], libdnsapi.Records()[0]),
						libcloudapi.ChangeRRSetTTL(token, libdnsapi.Zones()[0], libdnsapi.UpdatedARecord()),
						libcloudapi.SetRRSetRecords(token, libdnsapi.Zones()[0], libdnsapi.UpdatedARecord()),
					)
				}),
			Entry("Cloud API: A record with fqdn from name and domain", true,
				libserver.ZoneName, libserver.ARecordName, libserver.RecordTypeA, libserver.AUpdated, libdnsapi.UpdatedARecord, func() {
					api.AppendHandlers(
						libcloudapi.GetZone(token, libdnsapi.Zones()[0]),
						libcloudapi.GetRRSet(token, libdnsapi.Zones()[0], libdnsapi.Records()[0]),
						libcloudapi.ChangeRRSetTTL(token, libdnsapi.Zones()[0], libdnsapi.UpdatedARecord()),
						libcloudapi.SetRRSetRecords(token, libdnsapi.Zones()[0], libdnsapi.UpdatedARecord()),
					)
				}),
			Entry("Cloud API: TXT record with fqdn in domain", true,
				libserver.TXTRecordNameFull, "", libserver.RecordTypeTXT, libserver.TXTUpdated, libdnsapi.UpdatedTXTRecord, func() {
					api.AppendHandlers(
						libcloudapi.GetZone(token, libdnsapi.Zones()[0]),
						libcloudapi.GetRRSet(token, libdnsapi.Zones()[0], libdnsapi.Records()[1]),
						libcloudapi.ChangeRRSetTTL(token, libdnsapi.Zones()[0], libdnsapi.UpdatedTXTRecord()),
						libcloudapi.SetRRSetRecords(token, libdnsapi.Zones()[0], libdnsapi.UpdatedTXTRecord()),
					)
				}),
			Entry("Cloud API: TXT record with fqdn from name and domain", true,
				libserver.ZoneName, libserver.TXTRecordName, libserver.RecordTypeTXT, libserver.TXTUpdated, libdnsapi.UpdatedTXTRecord, func() {
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
		AfterEach(func() {
			Expect(api.ReceivedRequests()).To(BeEmpty())
		})

		DescribeTable("should succeed on action action than add with", func(ctx context.Context, action string, cloudAPI bool) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, libserver.WithCloudAPI(cloudAPI))

			statusCode, resData := doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_DNS_CONTROL", username, password,
				url.Values{
					"domain": []string{libserver.ARecordNameFull},
					"action": []string{action},
				},
			)
			Expect(statusCode).To(Equal(http.StatusOK))
			values, err := url.ParseQuery(resData)
			Expect(err).ToNot(HaveOccurred())
			Expect(values).To(Equal(statusOK))
		},
			Entry("delete (DNS)", "delete", false),
			Entry("update (DNS)", "update", false),
			Entry("something (DNS)", "something", false),
			Entry("delete (Cloud)", "delete", true),
			Entry("update (Cloud)", "update", true),
			Entry("something (Cloud)", "something", true),
		)

		DescribeTable("should return allowed domains", func(ctx context.Context, cloudAPI bool) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, libserver.WithCloudAPI(cloudAPI))

			statusCode, resData := doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_SHOW_DOMAINS", username, password, nil)
			Expect(statusCode).To(Equal(http.StatusOK))
			values, err := url.ParseQuery(resData)
			Expect(err).ToNot(HaveOccurred())
			Expect(values).To(Equal(url.Values{
				"list": []string{"*"},
			}))
		},
			Entry("DNS API", false),
			Entry("Cloud API", true),
		)

		DescribeTable("should succeed on calls to CMD_API_DOMAIN_POINTER", func(ctx context.Context, cloudAPI bool) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, libserver.WithCloudAPI(cloudAPI))

			statusCode, resData := doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_DOMAIN_POINTER", username, password,
				url.Values{
					"domain": []string{libserver.ZoneName},
				},
			)
			Expect(statusCode).To(Equal(http.StatusOK))
			Expect(resData).To(BeEmpty())
		},
			Entry("DNS API", false),
			Entry("Cloud API", true),
		)

		Context("should fail", func() {
			const domainActionMissing = "domain or action is missing\n"

			DescribeTable("for both APIs", func(ctx context.Context, cloudAPI bool) {
				server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, libserver.WithCloudAPI(cloudAPI))

				statusCode, resData := doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_DNS_CONTROL", username, password,
					url.Values{
						"action": []string{"add"},
						"type":   []string{libserver.RecordTypeTXT},
						"name":   []string{libserver.TXTRecordName},
						"value":  []string{libserver.TXTUpdated},
					},
				)
				Expect(statusCode).To(Equal(http.StatusBadRequest))
				Expect(resData).To(Equal(domainActionMissing))

				statusCode, resData = doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_DNS_CONTROL", username, password,
					url.Values{
						"domain": []string{libserver.ZoneName},
						"type":   []string{libserver.RecordTypeTXT},
						"name":   []string{libserver.TXTRecordName},
						"value":  []string{libserver.TXTUpdated},
					},
				)
				Expect(statusCode).To(Equal(http.StatusBadRequest))
				Expect(resData).To(Equal(domainActionMissing))

				statusCode, resData = doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_DNS_CONTROL", username, password,
					url.Values{
						"action": []string{"add"},
						"domain": []string{libserver.ZoneName},
						"type":   []string{"madeup"},
						"name":   []string{libserver.TXTRecordName},
						"value":  []string{libserver.TXTUpdated},
					},
				)
				Expect(statusCode).To(Equal(http.StatusBadRequest))
				Expect(resData).To(Equal("type can only be A or TXT\n"))

				statusCode, resData = doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_DNS_CONTROL", username, password,
					url.Values{
						"action": []string{"add"},
						"domain": []string{libserver.TLD},
						"type":   []string{libserver.RecordTypeTXT},
						"name":   []string{""},
						"value":  []string{libserver.TXTUpdated},
					},
				)
				Expect(statusCode).To(Equal(http.StatusBadRequest))
				Expect(resData).To(Equal("invalid fqdn: tld\n"))

				server.Close()
				server = libserver.NewNoAllowedDomains(api.URL(), libserver.WithCloudAPI(cloudAPI))
				statusCode, resData = doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_DNS_CONTROL", username, password,
					url.Values{
						"action": []string{"add"},
						"domain": []string{libserver.ARecordNameFull},
						"type":   []string{libserver.RecordTypeA},
						"name":   []string{""},
						"value":  []string{"something"},
					},
				)
				Expect(statusCode).To(Equal(http.StatusUnauthorized))
				Expect(resData).To(BeEmpty())
			},
				Entry("DNS API", false),
				Entry("Cloud API", true),
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

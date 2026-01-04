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

	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"

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
		DescribeTable("creating a new", func(ctx context.Context, cloudAPI bool, domain, name, recordType, value string) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, cloudAPI)

			if cloudAPI {
				var newRRSet func() schema.ZoneRRSet
				switch recordType {
				case libserver.RecordTypeA:
					newRRSet = libcloudapi.NewRRSetA
				case libserver.RecordTypeAAAA:
					newRRSet = libcloudapi.NewRRSetAAAA
				case libserver.RecordTypeTXT:
					newRRSet = libcloudapi.NewRRSetTXT
				}
				api.AppendHandlers(
					libcloudapi.GetZone(token, libcloudapi.Zone()),
					libcloudapi.GetRRSet(token, libcloudapi.Zone(), newRRSet(), false),
					libcloudapi.CreateRRSet(token, libcloudapi.Zone(), newRRSet()),
				)
			} else {
				var newRecord hetzner.Record
				switch recordType {
				case libserver.RecordTypeA:
					newRecord = libdnsapi.NewARecord()
				case libserver.RecordTypeAAAA:
					newRecord = libdnsapi.NewAAAARecord()
				case libserver.RecordTypeTXT:
					newRecord = libdnsapi.NewTXTRecord()
				}
				api.AppendHandlers(
					libdnsapi.GetZones(token, libdnsapi.Zones()),
					libdnsapi.GetRecords(token, libserver.ZoneID, nil),
					libdnsapi.PostRecord(token, newRecord),
				)
			}

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
				libserver.ARecordNameFull, "", libserver.RecordTypeA, libserver.AUpdated),
			Entry("DNS API: A record with fqdn from name and domain", false,
				libserver.ZoneName, libserver.ARecordName, libserver.RecordTypeA, libserver.AUpdated),
			Entry("DNS API: AAAA record with fqdn in domain", false,
				libserver.AAAARecordNameFull, "", libserver.RecordTypeAAAA, libserver.AAAAUpdated),
			Entry("DNS API: AAAA record with fqdn from name and domain", false,
				libserver.ZoneName, libserver.AAAARecordName, libserver.RecordTypeAAAA, libserver.AAAAUpdated),
			Entry("DNS API: TXT record with fqdn in domain", false,
				libserver.TXTRecordNameFull, "", libserver.RecordTypeTXT, libserver.TXTUpdated),
			Entry("DNS API: TXT record with fqdn from name and domain", false,
				libserver.ZoneName, libserver.TXTRecordName, libserver.RecordTypeTXT, libserver.TXTUpdated),
			Entry("Cloud API: A record with fqdn in domain", true,
				libserver.ARecordNameFull, "", libserver.RecordTypeA, libserver.AUpdated),
			Entry("Cloud API: A record with fqdn from name and domain", true,
				libserver.ZoneName, libserver.ARecordName, libserver.RecordTypeA, libserver.AUpdated),
			Entry("Cloud API: AAAA record with fqdn in domain", true,
				libserver.AAAARecordNameFull, "", libserver.RecordTypeAAAA, libserver.AAAAUpdated),
			Entry("Cloud API: AAAA record with fqdn from name and domain", true,
				libserver.ZoneName, libserver.AAAARecordName, libserver.RecordTypeAAAA, libserver.AAAAUpdated),
			Entry("Cloud API: TXT record with fqdn in domain", true,
				libserver.TXTRecordNameFull, "", libserver.RecordTypeTXT, libserver.TXTUpdated),
			Entry("Cloud API: TXT record with fqdn from name and domain", true,
				libserver.ZoneName, libserver.TXTRecordName, libserver.RecordTypeTXT, libserver.TXTUpdated),
		)

		DescribeTable("updating an existing", func(
			ctx context.Context, cloudAPI bool, domain, name, recordType, value string,
		) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, cloudAPI)

			if cloudAPI {
				var existingRRSet func() schema.ZoneRRSet
				var updatedRRSet func() schema.ZoneRRSet
				switch recordType {
				case libserver.RecordTypeA:
					existingRRSet = libcloudapi.ExistingRRSetA
					updatedRRSet = libcloudapi.UpdatedRRSetA
				case libserver.RecordTypeAAAA:
					existingRRSet = libcloudapi.ExistingRRSetAAAA
					updatedRRSet = libcloudapi.UpdatedRRSetAAAA
				case libserver.RecordTypeTXT:
					existingRRSet = libcloudapi.ExistingRRSetTXT
					updatedRRSet = libcloudapi.UpdatedRRSetTXT
				}
				api.AppendHandlers(
					libcloudapi.GetZone(token, libcloudapi.Zone()),
					libcloudapi.GetRRSet(token, libcloudapi.Zone(), existingRRSet(), true),
					libcloudapi.ChangeRRSetTTL(token, libcloudapi.Zone(), updatedRRSet()),
					libcloudapi.SetRRSetRecords(token, libcloudapi.Zone(), updatedRRSet()),
				)
			} else {
				var updatedRecord hetzner.Record
				switch recordType {
				case libserver.RecordTypeA:
					updatedRecord = libdnsapi.UpdatedARecord()
				case libserver.RecordTypeAAAA:
					updatedRecord = libdnsapi.UpdatedAAAARecord()
				case libserver.RecordTypeTXT:
					updatedRecord = libdnsapi.UpdatedTXTRecord()
				}
				api.AppendHandlers(
					libdnsapi.GetZones(token, libdnsapi.Zones()),
					libdnsapi.GetRecords(token, libserver.ZoneID, libdnsapi.Records()),
					libdnsapi.PutRecord(token, updatedRecord),
				)
			}

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
				libserver.ARecordNameFull, "", libserver.RecordTypeA, libserver.AUpdated),
			Entry("DNS API: A record with fqdn from name and domain", false,
				libserver.ZoneName, libserver.ARecordName, libserver.RecordTypeA, libserver.AUpdated),
			Entry("DNS API: AAAA record with fqdn in domain", false,
				libserver.AAAARecordNameFull, "", libserver.RecordTypeAAAA, libserver.AAAAUpdated),
			Entry("DNS API: AAAA record with fqdn from name and domain", false,
				libserver.ZoneName, libserver.AAAARecordName, libserver.RecordTypeAAAA, libserver.AAAAUpdated),
			Entry("DNS API: TXT record with fqdn in domain", false,
				libserver.TXTRecordNameFull, "", libserver.RecordTypeTXT, libserver.TXTUpdated),
			Entry("DNS API: TXT record with fqdn from name and domain", false,
				libserver.ZoneName, libserver.TXTRecordName, libserver.RecordTypeTXT, libserver.TXTUpdated),
			Entry("Cloud API: A record with fqdn in domain", true,
				libserver.ARecordNameFull, "", libserver.RecordTypeA, libserver.AUpdated),
			Entry("Cloud API: A record with fqdn from name and domain", true,
				libserver.ZoneName, libserver.ARecordName, libserver.RecordTypeA, libserver.AUpdated),
			Entry("Cloud API: AAAA record with fqdn in domain", true,
				libserver.AAAARecordNameFull, "", libserver.RecordTypeAAAA, libserver.AAAAUpdated),
			Entry("Cloud API: AAAA record with fqdn from name and domain", true,
				libserver.ZoneName, libserver.AAAARecordName, libserver.RecordTypeAAAA, libserver.AAAAUpdated),
			Entry("Cloud API: TXT record with fqdn in domain", true,
				libserver.TXTRecordNameFull, "", libserver.RecordTypeTXT, libserver.TXTUpdated),
			Entry("Cloud API: TXT record with fqdn from name and domain", true,
				libserver.ZoneName, libserver.TXTRecordName, libserver.RecordTypeTXT, libserver.TXTUpdated),
		)
	})

	Context("should make no api calls and", func() {
		AfterEach(func() {
			Expect(api.ReceivedRequests()).To(BeEmpty())
		})

		DescribeTable("should succeed on action other than add with", func(ctx context.Context, action string, cloudAPI bool) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, cloudAPI)

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
			Entry("DNS API: delete", "delete", false),
			Entry("DNS API: update", "update", false),
			Entry("DNS API: something", "something", false),
			Entry("Cloud API: delete", "delete", true),
			Entry("Cloud API: update", "update", true),
			Entry("Cloud API: something", "something", true),
		)

		DescribeTable("should return allowed domains", func(ctx context.Context, cloudAPI bool) {
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, cloudAPI)

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
			server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, cloudAPI)

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

			DescribeTable("when domain is missing", func(ctx context.Context, cloudAPI bool) {
				server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, cloudAPI)
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
			},
				Entry("DNS API", false),
				Entry("Cloud API", true),
			)

			DescribeTable("when action is missing", func(ctx context.Context, cloudAPI bool) {
				server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, cloudAPI)
				statusCode, resData := doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_DNS_CONTROL", username, password,
					url.Values{
						"domain": []string{libserver.ZoneName},
						"type":   []string{libserver.RecordTypeTXT},
						"name":   []string{libserver.TXTRecordName},
						"value":  []string{libserver.TXTUpdated},
					},
				)
				Expect(statusCode).To(Equal(http.StatusBadRequest))
				Expect(resData).To(Equal(domainActionMissing))
			},
				Entry("DNS API", false),
				Entry("Cloud API", true),
			)

			DescribeTable("when type is not A, AAAA or TXT", func(ctx context.Context, cloudAPI bool) {
				server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, cloudAPI)
				statusCode, resData := doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_DNS_CONTROL", username, password,
					url.Values{
						"action": []string{"add"},
						"domain": []string{libserver.ZoneName},
						"type":   []string{"madeup"},
						"name":   []string{libserver.TXTRecordName},
						"value":  []string{libserver.TXTUpdated},
					},
				)
				Expect(statusCode).To(Equal(http.StatusBadRequest))
				Expect(resData).To(Equal("type can only be A, AAAA or TXT\n"))
			},
				Entry("DNS API", false),
				Entry("Cloud API", true),
			)

			DescribeTable("when ip is invalid", func(ctx context.Context, recordType, value, expectedError string, cloudAPI bool) {
				server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, cloudAPI)
				statusCode, resData := doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_DNS_CONTROL", username, password,
					url.Values{
						"action": []string{"add"},
						"domain": []string{libserver.ZoneName},
						"type":   []string{recordType},
						"name":   []string{libserver.ARecordName},
						"value":  []string{value},
					},
				)
				Expect(statusCode).To(Equal(http.StatusBadRequest))
				Expect(resData).To(Equal(expectedError + "\n"))
			},
				Entry("DNS API: A with invalid IP", libserver.RecordTypeA, "invalid", "invalid ip address", false),
				Entry("DNS API: A with IPv6", libserver.RecordTypeA, libserver.AAAAUpdated, "invalid ipv4 address", false),
				Entry("DNS API: AAAA with invalid IP", libserver.RecordTypeAAAA, "invalid", "invalid ip address", false),
				Entry("DNS API: AAAA with IPv4", libserver.RecordTypeAAAA, libserver.AUpdated, "invalid ipv6 address", false),
				Entry("Cloud API: A with invalid IP", libserver.RecordTypeA, "invalid", "invalid ip address", true),
				Entry("Cloud API: A with IPv6", libserver.RecordTypeA, libserver.AAAAUpdated, "invalid ipv4 address", true),
				Entry("Cloud API: AAAA with invalid IP", libserver.RecordTypeAAAA, "invalid", "invalid ip address", true),
				Entry("Cloud API: AAAA with IPv4", libserver.RecordTypeAAAA, libserver.AUpdated, "invalid ipv6 address", true),
			)

			DescribeTable("when domain is malformed and name is empty", func(ctx context.Context, cloudAPI bool) {
				server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL, cloudAPI)
				statusCode, resData := doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_DNS_CONTROL", username, password,
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
			},
				Entry("DNS API", false),
				Entry("Cloud API", true),
			)

			DescribeTable("when access is denied", func(ctx context.Context, domain, name, recordType string, cloudAPI bool) {
				server = libserver.NewNoAllowedDomains(api.URL(), cloudAPI)
				value := "something"
				switch recordType {
				case libserver.RecordTypeA:
					value = libserver.AUpdated
				case libserver.RecordTypeAAAA:
					value = libserver.AAAAUpdated
				}
				statusCode, resData := doDirectAdminRequest(ctx, server.URL+"/directadmin/CMD_API_DNS_CONTROL", username, password,
					url.Values{
						"action": []string{"add"},
						"domain": []string{domain},
						"type":   []string{recordType},
						"name":   []string{name},
						"value":  []string{value},
					},
				)
				Expect(statusCode).To(Equal(http.StatusUnauthorized))
				Expect(resData).To(BeEmpty())
			},
				Entry("DNS API: A record with fqdn in domain",
					libserver.ARecordNameFull, "", libserver.RecordTypeA, false),
				Entry("DNS API: A record with fqdn from name and domain",
					libserver.ZoneName, libserver.ARecordName, libserver.RecordTypeA, false),
				Entry("DNS API: TXT record with fqdn in domain",
					libserver.TXTRecordNameFull, "", libserver.RecordTypeTXT, false),
				Entry("DNS API: TXT record with fqdn from name and domain",
					libserver.ZoneName, libserver.TXTRecordName, libserver.RecordTypeTXT, false),
				Entry("Cloud API: A record with fqdn in domain",
					libserver.ARecordNameFull, "", libserver.RecordTypeA, true),
				Entry("Cloud API: A record with fqdn from name and domain",
					libserver.ZoneName, libserver.ARecordName, libserver.RecordTypeA, true),
				Entry("Cloud API: TXT record with fqdn in domain",
					libserver.TXTRecordNameFull, "", libserver.RecordTypeTXT, true),
				Entry("Cloud API: TXT record with fqdn from name and domain",
					libserver.ZoneName, libserver.TXTRecordName, libserver.RecordTypeTXT, true),
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

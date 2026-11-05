package tests

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"github.com/hetznercloud/hcloud-go/v2/hcloud/schema"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
	"github.com/0xfelix/hetzner-dnsapi-proxy/tests/libcloudapi"
	"github.com/0xfelix/hetzner-dnsapi-proxy/tests/libserver"
)

const (
	otherHost = "a.test.tld"
	srvInput  = "0 5 25567 mc.test.tld"
)

func rec(fqdn, typ, value string) map[string]any {
	return map[string]any{keyFQDN: fqdn, keyType: typ, keyValue: value}
}

func recReplace(fqdn, typ, value string) map[string]any {
	r := rec(fqdn, typ, value)
	r["replace"] = true
	return r
}

var _ = Describe("Records", func() {
	var (
		api      *ghttp.Server
		server   *httptest.Server
		token    string
		username string
		password string
	)

	BeforeEach(func() {
		api = ghttp.NewServer()
		server, token, username, password = libserver.New(api.URL(), libserver.DefaultTTL)
	})

	AfterEach(func() {
		server.Close()
		api.Close()
	})

	Context("should succeed", func() {
		DescribeTable(
			"creating a new record", func(ctx context.Context, rrSet schema.ZoneRRSet, fqdn, value string) {
				api.AppendHandlers(
					libcloudapi.GetZone(token, libcloudapi.Zone()),
					libcloudapi.GetRRSet(token, libcloudapi.Zone(), rrSet, false),
					libcloudapi.ListRRSets(token, libcloudapi.Zone(), rrSet.Name, nil),
					libcloudapi.CreateRRSet(token, libcloudapi.Zone(), rrSet),
				)

				Expect(doRecordsRequest(
					ctx, server.URL+"/records/present", username, password,
					rec(fqdn, rrSet.Type, value),
				)).To(Equal(http.StatusOK))
				Expect(api.ReceivedRequests()).To(HaveLen(4))
			},
			Entry("A", libcloudapi.NewRRSetA(), libserver.ARecordNameFull, libserver.AUpdated),
			Entry("AAAA", libcloudapi.NewRRSetAAAA(), libserver.AAAARecordNameFull, libserver.AAAAUpdated),
			Entry("TXT", libcloudapi.NewRRSetTXT(), libserver.TXTRecordNameFull, libserver.TXTUpdated),
			Entry("CNAME", libcloudapi.NewRRSetCNAME(), libserver.CNAMERecordNameFull, "target.test.tld"),
			Entry("CNAME with dot suffix", libcloudapi.NewRRSetCNAME(), libserver.CNAMERecordNameFull+".", "target.test.tld."),
			Entry("SRV", libcloudapi.NewRRSetSRV(), libserver.SRVRecordNameFull, srvInput),
			Entry("SRV with dot suffix", libcloudapi.NewRRSetSRV(), libserver.SRVRecordNameFull+".", "0 5 25567 MC.test.tld."),
		)

		It("replacing an existing CNAME record", func(ctx context.Context) {
			api.AppendHandlers(
				libcloudapi.GetZone(token, libcloudapi.Zone()),
				libcloudapi.GetRRSet(token, libcloudapi.Zone(), libcloudapi.ExistingRRSetCNAME(), true),
				libcloudapi.SetRRSetRecords(token, libcloudapi.Zone(), libcloudapi.NewRRSetCNAME()),
			)

			Expect(doRecordsRequest(
				ctx, server.URL+"/records/present", username, password,
				rec(libserver.CNAMERecordNameFull, libserver.RecordTypeCNAME, "target.test.tld"),
			)).To(Equal(http.StatusOK))
			Expect(api.ReceivedRequests()).To(HaveLen(3))
		})

		DescribeTable(
			"adding a SRV record to an existing RRSet", func(ctx context.Context, ttl, expectedRequests int) {
				rrSet := libcloudapi.ExistingRRSetSRV()
				rrSet.TTL = &ttl
				api.AppendHandlers(
					libcloudapi.GetZone(token, libcloudapi.Zone()),
					libcloudapi.GetRRSet(token, libcloudapi.Zone(), rrSet, true),
				)
				if ttl != libserver.DefaultTTL {
					api.AppendHandlers(libcloudapi.ChangeRRSetTTL(token, libcloudapi.Zone(), libcloudapi.ExistingRRSetSRV()))
				}
				api.AppendHandlers(
					libcloudapi.AddRRSetRecords(token, libcloudapi.Zone(), rrSet, []schema.ZoneRRSetRecord{
						{Value: libserver.SRVValue},
					}),
				)

				Expect(doRecordsRequest(
					ctx, server.URL+"/records/present", username, password,
					rec(libserver.SRVRecordNameFull, libserver.RecordTypeSRV, srvInput),
				)).To(Equal(http.StatusOK))
				Expect(api.ReceivedRequests()).To(HaveLen(expectedRequests))
			},
			Entry("with matching TTL", libserver.DefaultTTL, 3),
			Entry("with different TTL", libserver.DefaultTTL+1, 4),
		)

		It("replacing an existing SRV RRSet with an unavailable service", func(ctx context.Context) {
			api.AppendHandlers(
				libcloudapi.GetZone(token, libcloudapi.Zone()),
				libcloudapi.GetRRSet(token, libcloudapi.Zone(), libcloudapi.ExistingRRSetSRV(), true),
				libcloudapi.SetRRSetRecords(token, libcloudapi.Zone(), libcloudapi.UnavailableRRSetSRV()),
			)

			Expect(doRecordsRequest(
				ctx, server.URL+"/records/present", username, password,
				rec(libserver.SRVRecordNameFull, libserver.RecordTypeSRV, libserver.SRVUnavailable),
			)).To(Equal(http.StatusOK))
			Expect(api.ReceivedRequests()).To(HaveLen(3))
		})

		DescribeTable(
			"replacing an unavailable service when adding a SRV record", func(ctx context.Context, ttl, expectedRequests int) {
				rrSet := libcloudapi.ExistingRRSetSRVUnavailable()
				rrSet.TTL = &ttl
				api.AppendHandlers(
					libcloudapi.GetZone(token, libcloudapi.Zone()),
					libcloudapi.GetRRSet(token, libcloudapi.Zone(), rrSet, true),
				)
				if ttl != libserver.DefaultTTL {
					api.AppendHandlers(libcloudapi.ChangeRRSetTTL(token, libcloudapi.Zone(), libcloudapi.ExistingRRSetSRV()))
				}
				api.AppendHandlers(libcloudapi.SetRRSetRecords(token, libcloudapi.Zone(), libcloudapi.NewRRSetSRV()))

				Expect(doRecordsRequest(
					ctx, server.URL+"/records/present", username, password,
					rec(libserver.SRVRecordNameFull, libserver.RecordTypeSRV, srvInput),
				)).To(Equal(http.StatusOK))
				Expect(api.ReceivedRequests()).To(HaveLen(expectedRequests))
			},
			Entry("with matching TTL", libserver.DefaultTTL, 3),
			Entry("with different TTL", libserver.DefaultTTL+1, 4),
		)

		It("replacing a SRV RRSet holding an unavailable service among real targets", func(ctx context.Context) {
			rrSet := libcloudapi.ExistingRRSetSRV()
			rrSet.Records = append(rrSet.Records, schema.ZoneRRSetRecord{Value: libserver.SRVUnavailable})
			api.AppendHandlers(
				libcloudapi.GetZone(token, libcloudapi.Zone()),
				libcloudapi.GetRRSet(token, libcloudapi.Zone(), rrSet, true),
				libcloudapi.SetRRSetRecords(token, libcloudapi.Zone(), libcloudapi.NewRRSetSRV()),
			)

			Expect(doRecordsRequest(
				ctx, server.URL+"/records/present", username, password,
				rec(libserver.SRVRecordNameFull, libserver.RecordTypeSRV, srvInput),
			)).To(Equal(http.StatusOK))
			Expect(api.ReceivedRequests()).To(HaveLen(3))
		})

		DescribeTable(
			"not adding a SRV record that already exists", func(ctx context.Context, ttl, expectedRequests int) {
				rrSet := libcloudapi.ExistingRRSetSRV()
				rrSet.Records = append(rrSet.Records, schema.ZoneRRSetRecord{Value: libserver.SRVValue})
				rrSet.TTL = &ttl
				api.AppendHandlers(
					libcloudapi.GetZone(token, libcloudapi.Zone()),
					libcloudapi.GetRRSet(token, libcloudapi.Zone(), rrSet, true),
				)
				if ttl != libserver.DefaultTTL {
					api.AppendHandlers(libcloudapi.ChangeRRSetTTL(token, libcloudapi.Zone(), libcloudapi.ExistingRRSetSRV()))
				}

				Expect(doRecordsRequest(
					ctx, server.URL+"/records/present", username, password,
					rec(libserver.SRVRecordNameFull, libserver.RecordTypeSRV, srvInput),
				)).To(Equal(http.StatusOK))
				Expect(api.ReceivedRequests()).To(HaveLen(expectedRequests))
			},
			Entry("with matching TTL", libserver.DefaultTTL, 2),
			Entry("with different TTL", libserver.DefaultTTL+1, 3),
		)

		DescribeTable(
			"cleaning up a record", func(ctx context.Context, rrSet schema.ZoneRRSet, fqdn, value string) {
				api.AppendHandlers(
					libcloudapi.GetZone(token, libcloudapi.Zone()),
					libcloudapi.GetRRSet(token, libcloudapi.Zone(), rrSet, true),
					libcloudapi.RemoveRRSetRecords(token, libcloudapi.Zone(), rrSet, rrSet.Records),
				)

				Expect(doRecordsRequest(
					ctx, server.URL+"/records/cleanup", username, password,
					rec(fqdn, rrSet.Type, value),
				)).To(Equal(http.StatusOK))
				Expect(api.ReceivedRequests()).To(HaveLen(3))
			},
			Entry("A", libcloudapi.NewRRSetA(), libserver.ARecordNameFull, libserver.AUpdated),
			Entry("AAAA", libcloudapi.NewRRSetAAAA(), libserver.AAAARecordNameFull, libserver.AAAAUpdated),
			Entry("TXT", libcloudapi.NewRRSetTXT(), libserver.TXTRecordNameFull, libserver.TXTUpdated),
			Entry("CNAME", libcloudapi.NewRRSetCNAME(), libserver.CNAMERecordNameFull, "target.test.tld"),
			Entry("SRV", libcloudapi.NewRRSetSRV(), libserver.SRVRecordNameFull, srvInput),
		)

		It("cleaning up a record that does not exist", func(ctx context.Context) {
			api.AppendHandlers(
				libcloudapi.GetZone(token, libcloudapi.Zone()),
				libcloudapi.GetRRSet(token, libcloudapi.Zone(), libcloudapi.NewRRSetCNAME(), false),
			)

			Expect(doRecordsRequest(
				ctx, server.URL+"/records/cleanup", username, password,
				rec(libserver.CNAMERecordNameFull, libserver.RecordTypeCNAME, "target.test.tld"),
			)).To(Equal(http.StatusOK))
			Expect(api.ReceivedRequests()).To(HaveLen(2))
		})

		It("replacing the values of an existing SRV RRSet when replace is set", func(ctx context.Context) {
			api.AppendHandlers(
				libcloudapi.GetZone(token, libcloudapi.Zone()),
				libcloudapi.GetRRSet(token, libcloudapi.Zone(), libcloudapi.ExistingRRSetSRV(), true),
				libcloudapi.SetRRSetRecords(token, libcloudapi.Zone(), libcloudapi.NewRRSetSRV()),
			)

			Expect(doRecordsRequest(
				ctx, server.URL+"/records/present", username, password,
				recReplace(libserver.SRVRecordNameFull, libserver.RecordTypeSRV, srvInput),
			)).To(Equal(http.StatusOK))
			Expect(api.ReceivedRequests()).To(HaveLen(3))
		})

		DescribeTable(
			"replacing the value of an existing RRSet", func(ctx context.Context, existing, expected schema.ZoneRRSet, fqdn, value string) {
				existing.TTL = new(libserver.DefaultTTL)
				api.AppendHandlers(
					libcloudapi.GetZone(token, libcloudapi.Zone()),
					libcloudapi.GetRRSet(token, libcloudapi.Zone(), existing, true),
					libcloudapi.SetRRSetRecords(token, libcloudapi.Zone(), expected),
				)

				Expect(doRecordsRequest(
					ctx, server.URL+"/records/present", username, password,
					rec(fqdn, expected.Type, value),
				)).To(Equal(http.StatusOK))
				Expect(api.ReceivedRequests()).To(HaveLen(3))
			},
			Entry("A", libcloudapi.ExistingRRSetA(), libcloudapi.NewRRSetA(), libserver.ARecordNameFull, libserver.AUpdated),
			Entry("TXT", libcloudapi.ExistingRRSetTXT(), libcloudapi.NewRRSetTXT(), libserver.TXTRecordNameFull, libserver.TXTUpdated),
		)

		It("creating a record next to a non-conflicting RRSet", func(ctx context.Context) {
			api.AppendHandlers(
				libcloudapi.GetZone(token, libcloudapi.Zone()),
				libcloudapi.GetRRSet(token, libcloudapi.Zone(), libcloudapi.NewRRSetA(), false),
				libcloudapi.ListRRSets(token, libcloudapi.Zone(), libserver.ARecordName, []schema.ZoneRRSet{
					func() schema.ZoneRRSet {
						r := libcloudapi.ExistingRRSetTXT()
						r.Name = libserver.ARecordName
						return r
					}(),
				}),
				libcloudapi.CreateRRSet(token, libcloudapi.Zone(), libcloudapi.NewRRSetA()),
			)

			Expect(doRecordsRequest(
				ctx, server.URL+"/records/present", username, password,
				rec(libserver.ARecordNameFull, libserver.RecordTypeA, libserver.AUpdated),
			)).To(Equal(http.StatusOK))
			Expect(api.ReceivedRequests()).To(HaveLen(4))
		})

		DescribeTable(
			"not adding a SRV record that exists in a different format", func(ctx context.Context, stored string) {
				rrSet := libcloudapi.ExistingRRSetSRV()
				rrSet.Records = append(rrSet.Records, schema.ZoneRRSetRecord{Value: stored})
				rrSet.TTL = new(libserver.DefaultTTL + 1)
				api.AppendHandlers(
					libcloudapi.GetZone(token, libcloudapi.Zone()),
					libcloudapi.GetRRSet(token, libcloudapi.Zone(), rrSet, true),
					libcloudapi.ChangeRRSetTTL(token, libcloudapi.Zone(), libcloudapi.ExistingRRSetSRV()),
				)

				Expect(doRecordsRequest(
					ctx, server.URL+"/records/present", username, password,
					rec(libserver.SRVRecordNameFull, libserver.RecordTypeSRV, srvInput),
				)).To(Equal(http.StatusOK))
				Expect(api.ReceivedRequests()).To(HaveLen(3))
			},
			Entry("without trailing dot", "0 5 25567 mc.test.tld"),
			Entry("with upper case and leading zeros", "00 05 25567 MC.Test.TLD."),
			Entry("with extra spaces", "0  5 25567  mc.test.tld."),
		)

		It("cleaning up a SRV record that is stored in a different format", func(ctx context.Context) {
			rrSet := libcloudapi.NewRRSetSRV()
			rrSet.Records = []schema.ZoneRRSetRecord{{Value: "10 5 25567 mc2.test.tld."}, {Value: "00 05 25567 MC.test.tld"}}
			api.AppendHandlers(
				libcloudapi.GetZone(token, libcloudapi.Zone()),
				libcloudapi.GetRRSet(token, libcloudapi.Zone(), rrSet, true),
				libcloudapi.RemoveRRSetRecords(token, libcloudapi.Zone(), rrSet, rrSet.Records[1:]),
			)

			Expect(doRecordsRequest(
				ctx, server.URL+"/records/cleanup", username, password,
				rec(libserver.SRVRecordNameFull, libserver.RecordTypeSRV, srvInput),
			)).To(Equal(http.StatusOK))
			Expect(api.ReceivedRequests()).To(HaveLen(3))
		})

		It("cleaning up a CNAME record that is stored in a different format", func(ctx context.Context) {
			rrSet := libcloudapi.NewRRSetCNAME()
			rrSet.Records = []schema.ZoneRRSetRecord{{Value: "Target.test.tld"}}
			api.AppendHandlers(
				libcloudapi.GetZone(token, libcloudapi.Zone()),
				libcloudapi.GetRRSet(token, libcloudapi.Zone(), rrSet, true),
				libcloudapi.RemoveRRSetRecords(token, libcloudapi.Zone(), rrSet, rrSet.Records),
			)

			Expect(doRecordsRequest(
				ctx, server.URL+"/records/cleanup", username, password,
				rec(libserver.CNAMERecordNameFull, libserver.RecordTypeCNAME, "target.test.tld"),
			)).To(Equal(http.StatusOK))
			Expect(api.ReceivedRequests()).To(HaveLen(3))
		})

		DescribeTable(
			"cleaning up a value that is not in the RRSet", func(ctx context.Context, rrSet schema.ZoneRRSet, fqdn, typ, value string) {
				api.AppendHandlers(
					libcloudapi.GetZone(token, libcloudapi.Zone()),
					libcloudapi.GetRRSet(token, libcloudapi.Zone(), rrSet, true),
				)

				Expect(doRecordsRequest(
					ctx, server.URL+"/records/cleanup", username, password,
					rec(fqdn, typ, value),
				)).To(Equal(http.StatusOK))
				Expect(api.ReceivedRequests()).To(HaveLen(2))
			},
			Entry("A", libcloudapi.ExistingRRSetA(), libserver.ARecordNameFull, libserver.RecordTypeA, libserver.AUpdated),
			Entry("AAAA", libcloudapi.ExistingRRSetAAAA(), libserver.AAAARecordNameFull, libserver.RecordTypeAAAA, libserver.AAAAUpdated),
			Entry("TXT", libcloudapi.ExistingRRSetTXT(), libserver.TXTRecordNameFull, libserver.RecordTypeTXT, libserver.TXTUpdated),
			Entry("CNAME", libcloudapi.ExistingRRSetCNAME(), libserver.CNAMERecordNameFull, libserver.RecordTypeCNAME, "target.test.tld"),
			Entry("SRV", libcloudapi.ExistingRRSetSRV(), libserver.SRVRecordNameFull, libserver.RecordTypeSRV, srvInput),
		)
	})

	Context("should handle concurrently created RRSets", func() {
		It("by replacing the records of a concurrently created RRSet", func(ctx context.Context) {
			api.AppendHandlers(
				libcloudapi.GetZone(token, libcloudapi.Zone()),
				libcloudapi.GetRRSet(token, libcloudapi.Zone(), libcloudapi.NewRRSetCNAME(), false),
				libcloudapi.ListRRSets(token, libcloudapi.Zone(), libserver.CNAMERecordName, nil),
				libcloudapi.CreateRRSetFails(token, libcloudapi.Zone(), libcloudapi.NewRRSetCNAME()),
				libcloudapi.GetRRSet(token, libcloudapi.Zone(), libcloudapi.ExistingRRSetCNAME(), true),
				libcloudapi.SetRRSetRecords(token, libcloudapi.Zone(), libcloudapi.NewRRSetCNAME()),
			)

			Expect(doRecordsRequest(
				ctx, server.URL+"/records/present", username, password,
				rec(libserver.CNAMERecordNameFull, libserver.RecordTypeCNAME, "target.test.tld"),
			)).To(Equal(http.StatusOK))
			Expect(api.ReceivedRequests()).To(HaveLen(6))
		})

		It("by adding to a concurrently created SRV RRSet", func(ctx context.Context) {
			api.AppendHandlers(
				libcloudapi.GetZone(token, libcloudapi.Zone()),
				libcloudapi.GetRRSet(token, libcloudapi.Zone(), libcloudapi.NewRRSetSRV(), false),
				libcloudapi.ListRRSets(token, libcloudapi.Zone(), libserver.SRVRecordName, nil),
				libcloudapi.CreateRRSetFails(token, libcloudapi.Zone(), libcloudapi.NewRRSetSRV()),
				libcloudapi.GetRRSet(token, libcloudapi.Zone(), libcloudapi.ExistingRRSetSRV(), true),
				libcloudapi.AddRRSetRecords(token, libcloudapi.Zone(), libcloudapi.ExistingRRSetSRV(), []schema.ZoneRRSetRecord{
					{Value: libserver.SRVValue},
				}),
			)

			Expect(doRecordsRequest(
				ctx, server.URL+"/records/present", username, password,
				rec(libserver.SRVRecordNameFull, libserver.RecordTypeSRV, srvInput),
			)).To(Equal(http.StatusOK))
			Expect(api.ReceivedRequests()).To(HaveLen(6))
		})

		It("by responding with a conflict for a concurrently created conflicting RRSet", func(ctx context.Context) {
			txt := libcloudapi.ExistingRRSetTXT()
			txt.Name = libserver.CNAMERecordName
			api.AppendHandlers(
				libcloudapi.GetZone(token, libcloudapi.Zone()),
				libcloudapi.GetRRSet(token, libcloudapi.Zone(), libcloudapi.NewRRSetCNAME(), false),
				libcloudapi.ListRRSets(token, libcloudapi.Zone(), libserver.CNAMERecordName, nil),
				libcloudapi.CreateRRSetFails(token, libcloudapi.Zone(), libcloudapi.NewRRSetCNAME()),
			)
			api.AppendHandlers(
				libcloudapi.GetRRSet(token, libcloudapi.Zone(), libcloudapi.NewRRSetCNAME(), false),
				libcloudapi.ListRRSets(token, libcloudapi.Zone(), libserver.CNAMERecordName, []schema.ZoneRRSet{txt}),
			)

			Expect(doRecordsRequest(
				ctx, server.URL+"/records/present", username, password,
				rec(libserver.CNAMERecordNameFull, libserver.RecordTypeCNAME, "target.test.tld"),
			)).To(Equal(http.StatusConflict))
			Expect(api.ReceivedRequests()).To(HaveLen(6))
		})

		It("by failing if the RRSet still does not exist", func(ctx context.Context) {
			api.AppendHandlers(
				libcloudapi.GetZone(token, libcloudapi.Zone()),
				libcloudapi.GetRRSet(token, libcloudapi.Zone(), libcloudapi.NewRRSetCNAME(), false),
				libcloudapi.ListRRSets(token, libcloudapi.Zone(), libserver.CNAMERecordName, nil),
				libcloudapi.CreateRRSetFails(token, libcloudapi.Zone(), libcloudapi.NewRRSetCNAME()),
			)
			// The RRSet is checked again after the failed create.
			api.AppendHandlers(
				libcloudapi.GetRRSet(token, libcloudapi.Zone(), libcloudapi.NewRRSetCNAME(), false),
				libcloudapi.ListRRSets(token, libcloudapi.Zone(), libserver.CNAMERecordName, nil),
			)

			Expect(doRecordsRequest(
				ctx, server.URL+"/records/present", username, password,
				rec(libserver.CNAMERecordNameFull, libserver.RecordTypeCNAME, "target.test.tld"),
			)).To(Equal(http.StatusInternalServerError))
			Expect(api.ReceivedRequests()).To(HaveLen(6))
		})
	})

	Context("should respond with a conflict and not create a record", func() {
		It("for a CNAME next to another RRSet", func(ctx context.Context) {
			api.AppendHandlers(
				libcloudapi.GetZone(token, libcloudapi.Zone()),
				libcloudapi.GetRRSet(token, libcloudapi.Zone(), libcloudapi.NewRRSetCNAME(), false),
				libcloudapi.ListRRSets(token, libcloudapi.Zone(), libserver.CNAMERecordName, []schema.ZoneRRSet{
					func() schema.ZoneRRSet {
						r := libcloudapi.ExistingRRSetTXT()
						r.Name = libserver.CNAMERecordName
						return r
					}(),
				}),
			)

			Expect(doRecordsRequest(
				ctx, server.URL+"/records/present", username, password,
				rec(libserver.CNAMERecordNameFull, libserver.RecordTypeCNAME, "target.test.tld"),
			)).To(Equal(http.StatusConflict))
			Expect(api.ReceivedRequests()).To(HaveLen(3))
		})

		DescribeTable(
			"for a record next to a CNAME", func(ctx context.Context, rrSet schema.ZoneRRSet, value string) {
				cname := libcloudapi.ExistingRRSetCNAME()
				cname.Name = rrSet.Name
				api.AppendHandlers(
					libcloudapi.GetZone(token, libcloudapi.Zone()),
					libcloudapi.GetRRSet(token, libcloudapi.Zone(), rrSet, false),
					libcloudapi.ListRRSets(token, libcloudapi.Zone(), rrSet.Name, []schema.ZoneRRSet{cname}),
				)

				Expect(doRecordsRequest(
					ctx, server.URL+"/records/present", username, password,
					rec(rrSet.Name+"."+libserver.ZoneName, rrSet.Type, value),
				)).To(Equal(http.StatusConflict))
				Expect(api.ReceivedRequests()).To(HaveLen(3))
			},
			Entry("A", libcloudapi.NewRRSetA(), libserver.AUpdated),
			Entry("TXT", libcloudapi.NewRRSetTXT(), libserver.TXTUpdated),
			Entry("SRV", libcloudapi.NewRRSetSRV(), srvInput),
		)
	})

	Context("should make no api calls and should fail", func() {
		AfterEach(func() {
			Expect(api.ReceivedRequests()).To(BeEmpty())
		})

		DescribeTable(
			"on invalid input", func(ctx context.Context, data map[string]any) {
				Expect(doRecordsRequest(
					ctx, server.URL+"/records/present", username, password, data,
				)).To(Equal(http.StatusBadRequest))
			},
			Entry("missing type", map[string]any{keyFQDN: libserver.CNAMERecordNameFull, keyValue: otherHost}),
			Entry("unsupported type", rec(libserver.CNAMERecordNameFull, "MX", otherHost)),
			Entry("CNAME at the zone apex", rec(libserver.ZoneName, libserver.RecordTypeCNAME, otherHost)),
			Entry("CNAME with invalid target", rec(libserver.CNAMERecordNameFull, libserver.RecordTypeCNAME, "a b.test.tld")),
			Entry("SRV with missing field", rec(libserver.SRVRecordNameFull, libserver.RecordTypeSRV, "0 5 mc.test.tld")),
			Entry("SRV with port out of range", rec(libserver.SRVRecordNameFull, libserver.RecordTypeSRV, "0 5 70000 mc.test.tld")),
			Entry("A with invalid address", rec(libserver.ARecordNameFull, "A", "nope")),
		)

		DescribeTable(
			"when access is denied", func(ctx context.Context, path string) {
				server.Close()
				server = libserver.NewNoAllowedDomains(api.URL())
				Expect(doRecordsRequest(
					ctx, server.URL+path, username, password,
					rec(libserver.CNAMERecordNameFull, libserver.RecordTypeCNAME, otherHost),
				)).To(Equal(http.StatusUnauthorized))
			},
			Entry("present", "/records/present"),
			Entry("cleanup", "/records/cleanup"),
		)

		DescribeTable(
			"when the endpoint is not enabled", func(ctx context.Context, path string) {
				server.Close()
				server, _, username, password = libserver.NewWithEndpoints(
					api.URL(), libserver.DefaultTTL, config.NewConfig().Endpoints,
				)
				Expect(doRecordsRequest(
					ctx, server.URL+path, username, password,
					rec(libserver.CNAMERecordNameFull, libserver.RecordTypeCNAME, otherHost),
				)).To(Equal(http.StatusNotFound))
			},
			Entry("present", "/records/present"),
			Entry("cleanup", "/records/cleanup"),
		)
	})
})

func doRecordsRequest(ctx context.Context, serverURL, username, password string, data any) int {
	body, err := json.Marshal(data)
	Expect(err).ToNot(HaveOccurred())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, serverURL, bytes.NewReader(body))
	Expect(err).ToNot(HaveOccurred())
	req.Header.Add("Content-Type", "application/json")
	req.SetBasicAuth(username, password)

	res, err := (&http.Client{}).Do(req)
	Expect(err).ToNot(HaveOccurred())
	Expect(res.Body.Close()).To(Succeed())

	return res.StatusCode
}

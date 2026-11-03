package tests

import (
	"context"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
	"github.com/0xfelix/hetzner-dnsapi-proxy/tests/libcloudapi"
	"github.com/0xfelix/hetzner-dnsapi-proxy/tests/libserver"
)

var _ = Describe("Records with restricted recordTypes", func() {
	var (
		api      *ghttp.Server
		server   *httptest.Server
		token    string
		username string
		password string
	)

	BeforeEach(func() {
		api = ghttp.NewServer()
		server, token, username, password = libserver.NewWithConfig(
			api.URL(), libserver.DefaultTTL,
			config.Endpoints{Records: true},
			func(cfg *config.Config) { cfg.RecordTypes = []string{config.RecordTypeCNAME} },
		)
	})

	AfterEach(func() {
		server.Close()
		api.Close()
	})

	DescribeTable(
		"should reject a disallowed type without calling the API", func(ctx context.Context, path string) {
			Expect(doRecordsRequest(
				ctx, server.URL+path, username, password,
				rec(libserver.SRVRecordNameFull, libserver.RecordTypeSRV, srvInput),
			)).To(Equal(http.StatusForbidden))
			Expect(api.ReceivedRequests()).To(BeEmpty())
		},
		Entry("present", "/records/present"),
		Entry("cleanup", "/records/cleanup"),
	)

	DescribeTable(
		"should answer unauthenticated requests with 401 regardless of the type",
		func(ctx context.Context, path, fqdn, typ, value string) {
			Expect(doRecordsRequest(
				ctx, server.URL+path, username, "wrong", rec(fqdn, typ, value),
			)).To(Equal(http.StatusUnauthorized))
			Expect(api.ReceivedRequests()).To(BeEmpty())
		},
		Entry("disallowed type on present", "/records/present",
			libserver.SRVRecordNameFull, libserver.RecordTypeSRV, srvInput),
		Entry("disallowed type on cleanup", "/records/cleanup",
			libserver.SRVRecordNameFull, libserver.RecordTypeSRV, srvInput),
		Entry("allowed type on present", "/records/present",
			libserver.CNAMERecordNameFull, libserver.RecordTypeCNAME, "target.test.tld"),
		Entry("allowed type on cleanup", "/records/cleanup",
			libserver.CNAMERecordNameFull, libserver.RecordTypeCNAME, "target.test.tld"),
	)

	It("should reject a disallowed type with a malformed value with 400", func(ctx context.Context) {
		Expect(doRecordsRequest(
			ctx, server.URL+"/records/present", username, password,
			rec(libserver.SRVRecordNameFull, libserver.RecordTypeSRV, "garbage"),
		)).To(Equal(http.StatusBadRequest))
		Expect(api.ReceivedRequests()).To(BeEmpty())
	})

	It("should still accept an allowed type", func(ctx context.Context) {
		rrSet := libcloudapi.NewRRSetCNAME()
		api.AppendHandlers(
			libcloudapi.GetZone(token, libcloudapi.Zone()),
			libcloudapi.GetRRSet(token, libcloudapi.Zone(), rrSet, false),
			libcloudapi.ListRRSets(token, libcloudapi.Zone(), rrSet.Name, nil),
			libcloudapi.CreateRRSet(token, libcloudapi.Zone(), rrSet),
		)

		Expect(doRecordsRequest(
			ctx, server.URL+"/records/present", username, password,
			rec(libserver.CNAMERecordNameFull, libserver.RecordTypeCNAME, "target.test.tld"),
		)).To(Equal(http.StatusOK))
		Expect(api.ReceivedRequests()).To(HaveLen(4))
	})
})

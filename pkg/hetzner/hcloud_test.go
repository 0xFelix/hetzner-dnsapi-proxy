package hetzner_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/hetzner"
)

var _ = Describe("QuoteIfRequired", func() {
	DescribeTable(
		"should quote TXT values", func(val, expected string) {
			Expect(hetzner.QuoteIfRequired(val, hcloud.ZoneRRSetTypeTXT)).To(Equal(expected))
		},
		Entry("plain", "some text", `"some text"`),
		Entry("empty", "", `""`),
		Entry("quote", `a"b`, `"a\"b"`),
		Entry("backslash", `a\b`, `"a\\b"`),
		Entry("control characters", "a\x00\x01\tb\n", `"a\000\001\009b\010"`),
		Entry("delete", "a\x7fb", `"a\127b"`),
		Entry("unicode", "ä", "\"ä\""),
	)

	DescribeTable(
		"should not quote other values", func(val string, rrSetType hcloud.ZoneRRSetType) {
			Expect(hetzner.QuoteIfRequired(val, rrSetType)).To(Equal(val))
		},
		Entry("A", "127.0.0.1", hcloud.ZoneRRSetTypeA),
		Entry("CNAME", "target.example.com.", hcloud.ZoneRRSetTypeCNAME),
		Entry("SRV", "0 5 5060 sip.example.com.", hcloud.ZoneRRSetTypeSRV),
	)
})

var _ = Describe("RRSetTypeFromString", func() {
	DescribeTable(
		"should accept supported types", func(rType string, expected hcloud.ZoneRRSetType) {
			rrSetType, err := hetzner.RRSetTypeFromString(rType)
			Expect(err).ToNot(HaveOccurred())
			Expect(rrSetType).To(Equal(expected))
		},
		Entry("A", "A", hcloud.ZoneRRSetTypeA),
		Entry("AAAA", "AAAA", hcloud.ZoneRRSetTypeAAAA),
		Entry("CNAME", "CNAME", hcloud.ZoneRRSetTypeCNAME),
		Entry("SRV", "SRV", hcloud.ZoneRRSetTypeSRV),
		Entry("TXT", "TXT", hcloud.ZoneRRSetTypeTXT),
	)

	DescribeTable(
		"should reject unsupported types", func(rType string) {
			_, err := hetzner.RRSetTypeFromString(rType)
			Expect(err).To(MatchError("unsupported resource record set type " + rType))
		},
		Entry("CAA", "CAA"),
		Entry("MX", "MX"),
		Entry("NS", "NS"),
		Entry("SOA", "SOA"),
	)

	DescribeTable(
		"should reject unrecognized types", func(rType string) {
			_, err := hetzner.RRSetTypeFromString(rType)
			Expect(err).To(MatchError("unrecognized resource record set type " + rType))
		},
		Entry("empty", ""),
		Entry("lower case", "a"),
		Entry("unknown", "FOO"),
	)
})

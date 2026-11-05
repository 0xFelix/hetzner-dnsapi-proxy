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

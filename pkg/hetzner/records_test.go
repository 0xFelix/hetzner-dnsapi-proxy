package hetzner_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/hetzner"
)

const (
	srvUnavailable = "0 0 0 ."
	srvTarget      = "0 5 25567 mc.test.tld."
)

var _ = Describe("NormalizeHostname", func() {
	DescribeTable(
		"should normalize", func(host, expected string) {
			result, err := hetzner.NormalizeHostname(host)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(expected))
		},
		Entry("plain", "a.example.com", "a.example.com"),
		Entry("one trailing dot", "a.example.com.", "a.example.com"),
		Entry("upper case", "A.Example.COM", "a.example.com"),
		Entry("underscore", "_dkim.example.com", "_dkim.example.com"),
	)

	DescribeTable(
		"should reject", func(host string) {
			_, err := hetzner.NormalizeHostname(host)
			Expect(err).To(MatchError("invalid hostname"))
		},
		Entry("two trailing dots", "a.example.com.."),
		Entry("only a dot", "."),
		Entry("empty", ""),
		Entry("empty label", "a..example.com"),
		Entry("space", "a b.example.com"),
	)
})

var _ = Describe("NormalizeSRV", func() {
	DescribeTable(
		"should normalize", func(value, expected string) {
			result, err := hetzner.NormalizeSRV(value)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(expected))
		},
		Entry("relative target", "10 5 5060 sip.example.com", "10 5 5060 sip.example.com."),
		Entry("leading zeros and spaces", " 00  05 05060\tSIP.example.com. ", "0 5 5060 sip.example.com."),
		Entry("unavailable service", srvUnavailable, srvUnavailable),
	)

	DescribeTable(
		"should reject", func(value, message string) {
			_, err := hetzner.NormalizeSRV(value)
			Expect(err).To(MatchError(message))
		},
		Entry("too few fields", "10 5 sip.example.com", "srv value must be 'priority weight port target'"),
		Entry("port out of range", "10 5 65536 sip.example.com", "srv priority, weight and port must be numbers between 0 and 65535"),
		Entry("target with two trailing dots", "10 5 5060 sip.example.com..", "invalid hostname"),
	)
})

var _ = Describe("ValuesEqual", func() {
	DescribeTable(
		"should compare", func(rrSetType hcloud.ZoneRRSetType, a, b string, expected bool) {
			Expect(hetzner.ValuesEqual(rrSetType, a, b)).To(Equal(expected))
			Expect(hetzner.ValuesEqual(rrSetType, b, a)).To(Equal(expected))
		},
		Entry("SRV identical", hcloud.ZoneRRSetTypeSRV, "0 5 25567 mc.test.tld.", "0 5 25567 mc.test.tld.", true),
		Entry("SRV without trailing dot", hcloud.ZoneRRSetTypeSRV, "0 5 25567 mc.test.tld.", "0 5 25567 mc.test.tld", true),
		Entry("SRV with case", hcloud.ZoneRRSetTypeSRV, "0 5 25567 mc.test.tld.", "0 5 25567 MC.Test.TLD.", true),
		Entry("SRV with leading zeros", hcloud.ZoneRRSetTypeSRV, "0 5 25567 mc.test.tld.", "00 05 025567 mc.test.tld.", true),
		Entry("SRV with extra spaces", hcloud.ZoneRRSetTypeSRV, "0 5 25567 mc.test.tld.", " 0  5 25567   mc.test.tld. ", true),
		Entry("SRV unavailable service", hcloud.ZoneRRSetTypeSRV, srvUnavailable, srvUnavailable, true),
		Entry("SRV different priority", hcloud.ZoneRRSetTypeSRV, "0 5 25567 mc.test.tld.", "1 5 25567 mc.test.tld.", false),
		Entry("SRV different port", hcloud.ZoneRRSetTypeSRV, "0 5 25567 mc.test.tld.", "0 5 25568 mc.test.tld.", false),
		Entry("SRV different target", hcloud.ZoneRRSetTypeSRV, "0 5 25567 mc.test.tld.", "0 5 25567 mc2.test.tld.", false),
		Entry("SRV unparsable identical", hcloud.ZoneRRSetTypeSRV, "garbage", "garbage", true),
		Entry("SRV unparsable differing in case", hcloud.ZoneRRSetTypeSRV, "garbage", "GARBAGE", false),
		Entry("SRV unparsable differs", hcloud.ZoneRRSetTypeSRV, "garbage", "0 5 25567 mc.test.tld.", false),
		Entry("CNAME identical", hcloud.ZoneRRSetTypeCNAME, "target.test.tld.", "target.test.tld.", true),
		Entry("CNAME without trailing dot", hcloud.ZoneRRSetTypeCNAME, "target.test.tld.", "target.test.tld", true),
		Entry("CNAME with case", hcloud.ZoneRRSetTypeCNAME, "target.test.tld.", "Target.TEST.tld", true),
		Entry("CNAME different", hcloud.ZoneRRSetTypeCNAME, "target.test.tld.", "other.test.tld.", false),
		Entry("CNAME invalid identical", hcloud.ZoneRRSetTypeCNAME, "a..test.tld.", "a..test.tld.", true),
		Entry("CNAME invalid differing in case", hcloud.ZoneRRSetTypeCNAME, "a..test.tld.", "A..Test.tld.", false),
		Entry("CNAME invalid differs", hcloud.ZoneRRSetTypeCNAME, "a..test.tld.", "a.test.tld.", false),
		Entry("CNAME with surrounding spaces", hcloud.ZoneRRSetTypeCNAME, "target.test.tld.", " target.test.tld. ", false),
		Entry("SRV invalid target identical", hcloud.ZoneRRSetTypeSRV, "0 5 25567 mc..test.tld.", "0 5 25567 mc..test.tld.", true),
		Entry("SRV invalid target differing in case", hcloud.ZoneRRSetTypeSRV, "0 5 25567 mc..test.tld.", "0 5 25567 MC..test.tld.", false),
		Entry("SRV invalid target differs", hcloud.ZoneRRSetTypeSRV, "0 5 25567 mc..test.tld.", "0 5 25567 mc.test.tld.", false),
		Entry("TXT identical", hcloud.ZoneRRSetTypeTXT, `"abc"`, `"abc"`, true),
		Entry("TXT is case-sensitive", hcloud.ZoneRRSetTypeTXT, `"Abc"`, `"abc"`, false),
		Entry("A is compared exactly", hcloud.ZoneRRSetTypeA, "127.0.0.1", "127.0.0.01", false),
		Entry("TXT trailing dot is significant", hcloud.ZoneRRSetTypeTXT, `"abc."`, `"abc"`, false),
	)
})

var _ = Describe("SRVUnavailable", func() {
	DescribeTable(
		"should report whether the SRV target is '.'", func(value string, expected bool) {
			Expect(hetzner.SRVUnavailable(value)).To(Equal(expected))
		},
		Entry("unavailable service", srvUnavailable, true),
		Entry("unavailable service with extra spaces", " 0  0 0 . ", true),
		Entry("real target", srvTarget, false),
		Entry("only a dot", ".", false),
		Entry("garbage", "garbage", false),
	)
})

var _ = Describe("HasSRVUnavailable", func() {
	DescribeTable(
		"should report whether an SRV RRSet holds a '.' target", func(rrSet *hcloud.ZoneRRSet, expected bool) {
			Expect(hetzner.HasSRVUnavailable(rrSet)).To(Equal(expected))
		},
		Entry("SRV with '.' among real targets", &hcloud.ZoneRRSet{
			Type:    hcloud.ZoneRRSetTypeSRV,
			Records: []hcloud.ZoneRRSetRecord{{Value: srvTarget}, {Value: srvUnavailable}},
		}, true),
		Entry("SRV with only real targets", &hcloud.ZoneRRSet{
			Type:    hcloud.ZoneRRSetTypeSRV,
			Records: []hcloud.ZoneRRSetRecord{{Value: srvTarget}},
		}, false),
		Entry("TXT with SRV-like value", &hcloud.ZoneRRSet{
			Type:    hcloud.ZoneRRSetTypeTXT,
			Records: []hcloud.ZoneRRSetRecord{{Value: srvUnavailable}},
		}, false),
	)
})

var _ = Describe("FindRecord", func() {
	It("should return the stored form of an SRV record stored in a different format", func() {
		rrSet := &hcloud.ZoneRRSet{
			Type:    hcloud.ZoneRRSetTypeSRV,
			Records: []hcloud.ZoneRRSetRecord{{Value: "0 5 25567 MC.test.tld."}},
		}
		val, found := hetzner.FindRecord(rrSet, "0 5 25567 mc.test.tld")
		Expect(found).To(BeTrue())
		Expect(val).To(Equal("0 5 25567 MC.test.tld."))
	})

	It("should match TXT records exactly", func() {
		rrSet := &hcloud.ZoneRRSet{
			Type:    hcloud.ZoneRRSetTypeTXT,
			Records: []hcloud.ZoneRRSetRecord{{Value: `"abc"`}, {Value: `"def"`}},
		}
		val, found := hetzner.FindRecord(rrSet, `"def"`)
		Expect(found).To(BeTrue())
		Expect(val).To(Equal(`"def"`))
	})

	It("should not find a mismatching A record", func() {
		rrSet := &hcloud.ZoneRRSet{
			Type:    hcloud.ZoneRRSetTypeA,
			Records: []hcloud.ZoneRRSetRecord{{Value: "127.0.0.1"}},
		}
		val, found := hetzner.FindRecord(rrSet, "127.0.0.2")
		Expect(found).To(BeFalse())
		Expect(val).To(BeEmpty())
	})

	It("should not find anything in an empty RRSet", func() {
		rrSet := &hcloud.ZoneRRSet{Type: hcloud.ZoneRRSetTypeA}
		val, found := hetzner.FindRecord(rrSet, "127.0.0.1")
		Expect(found).To(BeFalse())
		Expect(val).To(BeEmpty())
	})
})

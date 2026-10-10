package sanitize_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/sanitize"
)

var _ = Describe("LogValue", func() {
	DescribeTable(
		"should sanitize", func(value, expected string) {
			Expect(sanitize.LogValue(value)).To(Equal(expected))
		},
		Entry("printable value", "sub.example.com", "sub.example.com"),
		Entry("printable non-ascii value", "caf\u00e9", "caf\u00e9"),
		Entry("newline", "evil\nforged log line", "evilforged log line"),
		Entry("carriage return", "evil\rforged log line", "evilforged log line"),
		Entry("tab", "evil\tvalue", "evilvalue"),
		Entry("escape sequence", "evil\x1b[31mred", "evil[31mred"),
		Entry("null byte", "evil\x00value", "evilvalue"),
		Entry("delete", "evil\x7fvalue", "evilvalue"),
		Entry("line separator", "evil value", "evilvalue"),
	)
})

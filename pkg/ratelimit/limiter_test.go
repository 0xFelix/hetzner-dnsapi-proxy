package ratelimit

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Limiter", func() {
	const ip = "1.2.3.4"

	var (
		now time.Time
		l   *Limiter
	)

	BeforeEach(func() {
		now = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
		l = NewLimiter(1.0, 3, 10*time.Minute)
		l.now = func() time.Time { return now }
	})

	It("allows up to burst immediately", func() {
		Expect(l.Allow(ip)).To(BeTrue())
		Expect(l.Allow(ip)).To(BeTrue())
		Expect(l.Allow(ip)).To(BeTrue())
		Expect(l.Allow(ip)).To(BeFalse())
	})

	It("refills tokens over time", func() {
		for range 3 {
			Expect(l.Allow(ip)).To(BeTrue())
		}
		Expect(l.Allow(ip)).To(BeFalse())

		now = now.Add(2 * time.Second)
		Expect(l.Allow(ip)).To(BeTrue())
		Expect(l.Allow(ip)).To(BeTrue())
		Expect(l.Allow(ip)).To(BeFalse())
	})

	It("caps refill at burst", func() {
		Expect(l.Allow(ip)).To(BeTrue())
		now = now.Add(time.Hour)
		for range 3 {
			Expect(l.Allow(ip)).To(BeTrue())
		}
		Expect(l.Allow(ip)).To(BeFalse())
	})

	It("isolates keys", func() {
		for range 3 {
			Expect(l.Allow(ip)).To(BeTrue())
		}
		Expect(l.Allow(ip)).To(BeFalse())
		Expect(l.Allow("5.6.7.8")).To(BeTrue())
	})

	It("sweeps idle buckets", func() {
		Expect(l.Allow(ip)).To(BeTrue())
		Expect(l.buckets).To(HaveKey(ip))

		now = now.Add(11 * time.Minute)
		Expect(l.Allow("other")).To(BeTrue())
		Expect(l.buckets).NotTo(HaveKey(ip))
	})
})

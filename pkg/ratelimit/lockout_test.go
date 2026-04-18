package ratelimit

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Lockout", func() {
	const ip = "1.2.3.4"

	var (
		now time.Time
		l   *Lockout
	)

	BeforeEach(func() {
		now = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
		l = NewLockout(3, time.Hour, 15*time.Minute)
		l.now = func() time.Time { return now }
	})

	It("is not blocked without recorded failures", func() {
		Expect(l.IsBlocked(ip)).To(BeFalse())
	})

	It("does not block below the threshold", func() {
		Expect(l.RecordFailure(ip)).To(BeFalse())
		Expect(l.RecordFailure(ip)).To(BeFalse())
		Expect(l.IsBlocked(ip)).To(BeFalse())
	})

	It("blocks when the threshold is reached", func() {
		Expect(l.RecordFailure(ip)).To(BeFalse())
		Expect(l.RecordFailure(ip)).To(BeFalse())
		Expect(l.RecordFailure(ip)).To(BeTrue())
		Expect(l.IsBlocked(ip)).To(BeTrue())
	})

	It("isolates keys", func() {
		for range 3 {
			l.RecordFailure(ip)
		}
		Expect(l.IsBlocked(ip)).To(BeTrue())
		Expect(l.IsBlocked("5.6.7.8")).To(BeFalse())
	})

	It("expires the lockout after duration and drops the entry", func() {
		for range 3 {
			l.RecordFailure(ip)
		}
		Expect(l.IsBlocked(ip)).To(BeTrue())

		now = now.Add(time.Hour + time.Second)
		Expect(l.IsBlocked(ip)).To(BeFalse())
		Expect(l.entries).NotTo(HaveKey(ip))
	})

	It("starts a fresh count after the lockout expires", func() {
		for range 3 {
			l.RecordFailure(ip)
		}
		now = now.Add(time.Hour + time.Second)

		Expect(l.RecordFailure(ip)).To(BeFalse())
		Expect(l.IsBlocked(ip)).To(BeFalse())
	})

	It("decays partial failures that fall outside the window", func() {
		Expect(l.RecordFailure(ip)).To(BeFalse())
		Expect(l.RecordFailure(ip)).To(BeFalse())

		now = now.Add(16 * time.Minute)
		Expect(l.RecordFailure(ip)).To(BeFalse())
		Expect(l.IsBlocked(ip)).To(BeFalse())
	})

	It("Reset clears state", func() {
		for range 3 {
			l.RecordFailure(ip)
		}
		Expect(l.IsBlocked(ip)).To(BeTrue())

		l.Reset(ip)
		Expect(l.IsBlocked(ip)).To(BeFalse())
		Expect(l.entries).NotTo(HaveKey(ip))
	})

	It("drops stale entries when IsBlocked is called on them", func() {
		for range 2 {
			l.RecordFailure(ip)
		}
		Expect(l.entries).To(HaveKey(ip))

		now = now.Add(16 * time.Minute)
		Expect(l.IsBlocked(ip)).To(BeFalse())
		Expect(l.entries).NotTo(HaveKey(ip))
	})

	Context("when the entry cap is reached", func() {
		BeforeEach(func() {
			l.maxEntries = 3
		})

		It("sweeps stale entries before admitting a new key", func() {
			l.RecordFailure("a")
			l.RecordFailure("b")
			l.RecordFailure("c")
			Expect(l.entries).To(HaveLen(3))

			now = now.Add(16 * time.Minute)
			// Bypass shouldSweep by forcing lastSweep recent so the cap
			// logic has to run the eager sweep itself.
			l.lastSweep = now
			l.RecordFailure("d")
			Expect(l.entries).To(HaveKey("d"))
			Expect(len(l.entries)).To(BeNumerically("<=", 3))
		})

		It("evicts an existing entry when sweep cannot free space", func() {
			l.RecordFailure("a")
			l.RecordFailure("b")
			l.RecordFailure("c")
			Expect(l.entries).To(HaveLen(3))

			l.lastSweep = now
			l.RecordFailure("d")
			Expect(l.entries).To(HaveKey("d"))
			Expect(len(l.entries)).To(BeNumerically("<=", 3))
		})
	})
})

package ratelimit_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRatelimit(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ratelimit test suite")
}

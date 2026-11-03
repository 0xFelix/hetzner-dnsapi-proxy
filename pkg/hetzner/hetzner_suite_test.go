package hetzner_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHetzner(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "hetzner test suite")
}

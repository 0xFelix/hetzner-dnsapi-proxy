package config_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const exampleDomain = "example.com"

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "config test suite")
}

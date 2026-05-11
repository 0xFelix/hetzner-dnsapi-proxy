package middleware_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	exampleDomain     = "example.com"
	testDomain        = "test.com"
	subExampleDomain  = "sub.example.com"
	wildcardExample   = "*.example.com"
	username          = "username"
	password          = "password"
	invalidAuthMethod = "invalid"
)

func TestMiddleware(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "middleware test suite")
}

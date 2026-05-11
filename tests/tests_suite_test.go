package tests

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	keySubdomain = "subdomain"
	keyTXT       = "txt"
	keyDomain    = "domain"
	keyAction    = "action"
	keyType      = "type"
	keyName      = "name"
	keyValue     = "value"
	keyFQDN      = "fqdn"
	keyHostname  = "hostname"
	keyMyIP      = "myip"
	keyIP        = "ip"
	invalidValue = "invalid"
)

func TestFunctional(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Functional test suite")
}

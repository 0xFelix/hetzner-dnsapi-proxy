package hetzner

import (
	"errors"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

const (
	maxHostnameLength = 253
	srvFieldCount     = 4

	// srvTargetUnavailable is the SRV target meaning that the service is not
	// available (RFC 2782).
	srvTargetUnavailable = "."
)

var hostnameRegexp = regexp.MustCompile(`^([a-z0-9_]([a-z0-9_-]{0,61}[a-z0-9_])?\.)*[a-z0-9_]([a-z0-9_-]{0,61}[a-z0-9_])?$`)

// NormalizeHostname lowercases the host, strips at most one trailing dot and
// validates the result.
func NormalizeHostname(host string) (string, error) {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	if len(host) > maxHostnameLength || !hostnameRegexp.MatchString(host) {
		return "", errors.New("invalid hostname")
	}
	return host, nil
}

// NormalizeSRV validates an SRV value of the form 'priority weight port
// target' and returns it with canonical numbers and an absolute target.
func NormalizeSRV(value string) (string, error) {
	fields := strings.Fields(value)
	if len(fields) != srvFieldCount {
		return "", errors.New("srv value must be 'priority weight port target'")
	}

	for i, f := range fields[:srvFieldCount-1] {
		n, err := strconv.ParseUint(f, 10, 16)
		if err != nil {
			return "", errors.New("srv priority, weight and port must be numbers between 0 and 65535")
		}
		// Canonicalize, e.g. strip leading zeros, so values compare equal
		fields[i] = strconv.FormatUint(n, 10)
	}

	// A target of '.' means that the service is not available.
	if target := fields[srvFieldCount-1]; target != srvTargetUnavailable {
		host, err := NormalizeHostname(target)
		if err != nil {
			return "", err
		}
		fields[srvFieldCount-1] = host + "."
	}

	return strings.Join(fields, " "), nil
}

// SRVUnavailable reports whether value is an SRV value with the target '.',
// which means that the service is not available (RFC 2782).
func SRVUnavailable(value string) bool {
	fields := strings.Fields(value)
	return len(fields) == srvFieldCount && fields[srvFieldCount-1] == srvTargetUnavailable
}

// HasSRVUnavailable reports whether rrSet is an SRV RRSet holding a record
// with the target '.' (see SRVUnavailable).
func HasSRVUnavailable(rrSet *hcloud.ZoneRRSet) bool {
	return rrSet.Type == hcloud.ZoneRRSetTypeSRV && slices.ContainsFunc(rrSet.Records, func(r hcloud.ZoneRRSetRecord) bool {
		return SRVUnavailable(r.Value)
	})
}

// valueNormalizers holds the normalize function for each type whose values
// can be compared in normalized form.
var valueNormalizers = map[hcloud.ZoneRRSetType]func(string) (string, error){
	hcloud.ZoneRRSetTypeCNAME: NormalizeHostname,
	hcloud.ZoneRRSetTypeSRV:   NormalizeSRV,
}

// ValuesEqual reports whether two record values of the given type are equal.
// CNAME and SRV values are normalized with NormalizeHostname or NormalizeSRV
// and the results are compared. Other types, and values that fail to
// normalize, e.g. invalid values stored by hand, are compared exactly.
func ValuesEqual(rrSetType hcloud.ZoneRRSetType, a, b string) bool {
	normalize, ok := valueNormalizers[rrSetType]
	if !ok {
		return a == b
	}

	na, errA := normalize(a)
	nb, errB := normalize(b)
	if errA != nil || errB != nil {
		return a == b
	}
	return na == nb
}

// FindRecord returns the stored value of the record in rrSet that equals val according to ValuesEqual.
func FindRecord(rrSet *hcloud.ZoneRRSet, val string) (string, bool) {
	for _, record := range rrSet.Records {
		if ValuesEqual(rrSet.Type, record.Value, val) {
			return record.Value, true
		}
	}
	return "", false
}

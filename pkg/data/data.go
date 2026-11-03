package data

import (
	"context"
	"errors"
)

// ErrConflict is returned when a record cannot be created because it would
// conflict with an existing record set, e.g. a CNAME next to another type.
var ErrConflict = errors.New("conflicting record set")

// Mode defines how a value is applied to an existing RRSet.
type Mode int

const (
	// ModeReplace sets the RRSet to exactly the given value. It is the zero value.
	ModeReplace Mode = iota
	// ModeAppend adds the value to the RRSet unless it is already present.
	ModeAppend
)

type ReqData struct {
	FullName  string
	Name      string
	Zone      string
	Value     string
	Type      string
	Username  string
	Password  string
	BasicAuth bool
	Mode      Mode
	// CheckConflicts makes the updater reject new RRSets that would conflict
	// with an existing CNAME RRSet (or, for CNAME, with any other RRSet).
	CheckConflicts bool
}

// key is an unexported type for keys defined in this package.
// This prevents collisions with keys defined in other packages.
type key int

// reqDataKey is the key for ReqData values in Contexts.
// It is unexported; clients use newContextWithReqData and reqDataFromContext
// instead of using this key directly.
var reqDataKey key

// newContextWithReqData returns a new Context that stores a ReqData pointer as a value.
func NewContextWithReqData(ctx context.Context, data *ReqData) context.Context {
	return context.WithValue(ctx, reqDataKey, data)
}

// reqDataFromContext returns the pointer to a ReqData stored in a Context.
func ReqDataFromContext(ctx context.Context) (*ReqData, error) {
	data, ok := ctx.Value(reqDataKey).(*ReqData)
	if !ok {
		return nil, errors.New("ReqData not found in context")
	}
	return data, nil
}

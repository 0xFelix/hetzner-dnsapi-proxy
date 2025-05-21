package middleware

import (
	"context"
	"errors"
)

// ReqData is an exported struct holding request-specific data.
// (Assuming it was previously unexported or named differently like `reqData`)
type ReqData struct {
	FullName  string
	Name      string
	Zone      string
	Value     string
	Type      string
	Username  string
	Password  string
	BasicAuth bool
}

// key is an unexported type for keys defined in this package.
// This prevents collisions with keys defined in other packages.
type key int

// reqDataKey is the key for ReqData values in Contexts.
// It remains unexported as access is through exported functions.
var reqDataKey key //nolint:gochecknoglobals // Used for context key

// NewContextWithReqData returns a new Context that stores a ReqData pointer as a value.
// (Exported version of newContextWithReqData)
func NewContextWithReqData(ctx context.Context, data *ReqData) context.Context {
	return context.WithValue(ctx, reqDataKey, data)
}

// ReqDataFromContext returns the pointer to a ReqData stored in a Context.
// (Exported version of reqDataFromContext)
func ReqDataFromContext(ctx context.Context) (*ReqData, error) {
	data, ok := ctx.Value(reqDataKey).(*ReqData)
	if !ok {
		return nil, errors.New("ReqData not found in context")
	}
	return data, nil
}

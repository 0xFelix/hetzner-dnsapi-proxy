package middleware

import (
	"context"
	"errors"
)

type reqData struct {
	FullName string
	Name     string
	Zone     string
	Value    string
	Type     string
}

// key is an unexported type for keys defined in this package.
// This prevents collisions with keys defined in other packages.
type key int

// reqDataKey is the key for reqData values in Contexts.
// It is unexported; clients use newContextWithReqData and reqDataFromContext
// instead of using this key directly.
var reqDataKey key

// newContextWithReqData returns a new Context that stores a reqData pointer as a value.
func newContextWithReqData(ctx context.Context, data *reqData) context.Context {
	return context.WithValue(ctx, reqDataKey, data)
}

// reqDataFromContext returns the pointer to a reqData stored in a Context.
func reqDataFromContext(ctx context.Context) (*reqData, error) {
	data, ok := ctx.Value(reqDataKey).(*reqData)
	if !ok {
		return nil, errors.New("reqData not found in context")
	}
	return data, nil
}

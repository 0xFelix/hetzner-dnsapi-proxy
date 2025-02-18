package middleware

import (
	"context"
	"errors"
)

type ReqData struct {
	FullName string
	Name     string
	Zone     string
	Value    string
	Type     string
	Username string
	Password string
}

// key is an unexported type for keys defined in this package.
// This prevents collisions with keys defined in other packages.
type key int

// reqDataKey is the key for ReqData values in Contexts.
// It is unexported; clients use newContextWithReqData and reqDataFromContext
// instead of using this key directly.
var reqDataKey key

// newContextWithReqData returns a new Context that stores a ReqData pointer as a value.
func newContextWithReqData(ctx context.Context, data *ReqData) context.Context {
	return context.WithValue(ctx, reqDataKey, data)
}

// reqDataFromContext returns the pointer to a ReqData stored in a Context.
func reqDataFromContext(ctx context.Context) (*ReqData, error) {
	data, ok := ctx.Value(reqDataKey).(*ReqData)
	if !ok {
		return nil, errors.New("ReqData not found in context")
	}
	return data, nil
}

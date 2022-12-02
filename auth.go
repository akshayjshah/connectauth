// Package connectauth provides a flexible authentication interceptor for
// [github.com/bufbuild/connect-go].
package connectauth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bufbuild/connect-go"
)

type key int

const identityKey key = iota

// GetIdentity retrieves the authenticated identity, if any, from the request
// context.
func GetIdentity(ctx context.Context) any {
	return ctx.Value(identityKey)
}

// WithoutIdentity strips the authenticated identity, if any, from the provided
// context.
func WithoutIdentity(ctx context.Context) context.Context {
	return context.WithValue(ctx, identityKey, nil)
}

// Errorf is a convenience function that returns an error coded with
// [connect.CodeUnauthenticated].
func Errorf(template string, args ...any) *connect.Error {
	return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf(template, args...))
}

// Request describes a single RPC invocation.
type Request struct {
	Spec   connect.Spec
	Peer   connect.Peer
	Header http.Header
}

// Interceptor is a server-side authentication interceptor. In addition to
// rejecting unauthenticated requests, it can optionally attach an identity to
// context of authenticated requests.
type Interceptor struct {
	auth func(context.Context, *Request) (any, error)
}

// New constructs a new Interceptor using the supplied authentication function.
// The authentication function must return an error if the request cannot be
// authenticated. The error is typically produced with [Errorf], but any error
// will do.
//
// If requests are successfully authenticated, the authentication function may
// return the authenticated identity. The identity will be attached to the
// context, so subsequent interceptors and application code may access it with
// [GetIdentity].
//
// Authentication functions must be safe to call concurrently.
func New(f func(context.Context, *Request) (any, error)) *Interceptor {
	return &Interceptor{f}
}

// WrapUnary implements connect.Interceptor.
func (i *Interceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		identity, err := i.auth(ctx, &Request{
			Spec:   req.Spec(),
			Peer:   req.Peer(),
			Header: req.Header(),
		})
		if err != nil {
			return nil, err
		}
		return next(context.WithValue(ctx, identityKey, identity), req)
	}
}

// WrapStreamingClient implements connect.Interceptor with a no-op.
func (i *Interceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

// WrapStreamingHandler implements connect.Interceptor.
func (i *Interceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		identity, err := i.auth(ctx, &Request{
			Spec:   conn.Spec(),
			Peer:   conn.Peer(),
			Header: conn.RequestHeader(),
		})
		if err != nil {
			return err
		}
		return next(context.WithValue(ctx, identityKey, identity), conn)
	}
}

// Package connectauth provides flexible authentication middleware for
// [connect].
package connectauth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
)

type key int

const infoKey key = iota

// An AuthFunc authenticates an RPC. The function must return an error if the
// request cannot be authenticated. The error is typically produced with
// [Errorf], but any error will do.
//
// If requests are successfully authenticated, the authentication function may
// return some information about the authenticated caller (or nil).
// Authentication functions must be safe to call concurrently.
type AuthFunc = func(context.Context, *Request) (any, error)

// SetInfo attaches authentication information to the context. It's often
// useful in tests.
func SetInfo(ctx context.Context, info any) context.Context {
	if info == nil {
		return ctx
	}
	return context.WithValue(ctx, infoKey, info)
}

// GetInfo retrieves authentication information, if any, from the request
// context.
func GetInfo(ctx context.Context) any {
	return ctx.Value(infoKey)
}

// WithoutInfo strips the authentication information, if any, from the provided
// context.
func WithoutInfo(ctx context.Context) context.Context {
	return context.WithValue(ctx, infoKey, nil)
}

// Errorf is a convenience function that returns an error coded with
// [connect.CodeUnauthenticated].
func Errorf(template string, args ...any) *connect.Error {
	return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf(template, args...))
}

// Request describes a single RPC invocation.
type Request struct {
	Procedure  string // for example, "/acme.foo.v1.FooService/Bar"
	ClientAddr string // client address, in IP:port format
	Protocol   string // connect.ProtocolConnect, connect.ProtocolGRPC, or connect.ProtocolGRPCWeb
	Header     http.Header
}

// Middleware is server-side HTTP middleware that authenticates RPC requests.
// In addition to rejecting unauthenticated requests, it can optionally attach
// arbitrary information to the context of authenticated requests. Any non-RPC
// requests (as determined by their Content-Type) are forwarded directly to the
// wrapped handler without authentication.
//
// Middleware operates at a lower level than [Interceptor]. For most
// applications, Middleware is preferable because it defers decompressing and
// unmarshaling the request until after the caller has been authenticated.
type Middleware struct {
	auth AuthFunc
	errW *connect.ErrorWriter
}

// NewMiddleware constructs HTTP middleware using the supplied authentication
// function. If authentication succeeds, the authentication information (if
// any) will be attached to the context. Subsequent HTTP middleware, all RPC
// interceptors, and application code may access it with [GetInfo].
//
// In order to properly identify RPC requests and marshal errors, applications
// must pass NewMiddleware the same handler options used when constructing
// Connect handlers.
func NewMiddleware(auth AuthFunc, opts ...connect.HandlerOption) *Middleware {
	return &Middleware{
		auth: auth,
		errW: connect.NewErrorWriter(opts...),
	}
}

// Wrap decorates an HTTP handler with authentication logic.
func (m *Middleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.errW.IsSupported(r) {
			next.ServeHTTP(w, r)
			return
		}
		ctx := r.Context()
		info, err := m.auth(ctx, &Request{
			Procedure:  procedureFromHTTP(r),
			ClientAddr: r.RemoteAddr,
			Protocol:   protocolFromHTTP(r),
			Header:     r.Header,
		})
		if err != nil {
			m.errW.Write(w, r, err)
			return
		}
		if info != nil {
			r = r.WithContext(SetInfo(ctx, info))
		}
		next.ServeHTTP(w, r)
	})
}

// Interceptor is a server-side authentication interceptor. In addition to
// rejecting unauthenticated requests, it can optionally attach arbitrary
// information to the context of authenticated requests.
//
// Because RPC interceptors run after the request has already been decompressed
// and unmarshaled, it's inefficient (and potentially dangerous) to rely on
// interceptors for authentication. Most applications should use [Middleware]
// instead, unless they:
//   - Mount Connect HTTP handlers on routes that don't end in the procedure
//     name (e.g., "/user.v1/GetUser"). This is unusual, since it breaks
//     generated Connect and gRPC clients.
//   - Use authentication logic that relies on other interceptors (e.g.,
//     authenticating requests relies on a struct attached to the context by a
//     previous interceptor).
//
// Attach interceptors to your RPC handlers using [connect.WithInterceptors].
type Interceptor struct {
	auth func(context.Context, *Request) (any, error)
}

// NewInterceptor constructs a Connect interceptor using the supplied
// authentication function. If authentication succeeds, the authentication
// information (if any) will be attached to the context. Subsequent
// interceptors and application code may access it with [GetInfo].
//
// Most applications should use [Middleware] instead.
func NewInterceptor(auth AuthFunc) *Interceptor {
	return &Interceptor{auth}
}

// WrapUnary implements connect.Interceptor.
func (i *Interceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		spec := req.Spec()
		peer := req.Peer()
		info, err := i.auth(ctx, &Request{
			Procedure:  spec.Procedure,
			ClientAddr: peer.Addr,
			Protocol:   peer.Protocol,
			Header:     req.Header(),
		})
		if err != nil {
			return nil, err
		}
		return next(SetInfo(ctx, info), req)
	}
}

// WrapStreamingClient implements connect.Interceptor with a no-op.
func (i *Interceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

// WrapStreamingHandler implements connect.Interceptor.
func (i *Interceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		spec := conn.Spec()
		peer := conn.Peer()
		info, err := i.auth(ctx, &Request{
			Procedure:  spec.Procedure,
			ClientAddr: peer.Addr,
			Protocol:   peer.Protocol,
			Header:     conn.RequestHeader(),
		})
		if err != nil {
			return err
		}
		return next(SetInfo(ctx, info), conn)
	}
}

func procedureFromHTTP(r *http.Request) string {
	path := strings.TrimSuffix(r.URL.Path, "/")
	ultimate := strings.LastIndex(path, "/")
	if ultimate < 0 {
		return ""
	}
	penultimate := strings.LastIndex(path[:ultimate], "/")
	if penultimate < 0 {
		return ""
	}
	procedure := path[penultimate:]
	if len(procedure) < 4 { // two slashes + service + method
		return ""
	}
	return procedure
}

func protocolFromHTTP(r *http.Request) string {
	ct := r.Header.Get("Content-Type")
	switch {
	case strings.HasPrefix(ct, "application/grpc-web"):
		return connect.ProtocolGRPCWeb
	case strings.HasPrefix(ct, "application/grpc"):
		return connect.ProtocolGRPC
	default:
		return connect.ProtocolConnect
	}
}

package connectauth_test

import (
	"context"
	"fmt"
	"net/http"

	"github.com/akshayjshah/connectauth"
	"github.com/bufbuild/connect-go"
	"google.golang.org/protobuf/types/known/emptypb"
)

// TestServiceHandler is an interface describing the server-side implementation
// of our example RPC service. It would typically be generated from a protobuf
// schema.
type TestServiceHandler interface {
	GetEmpty(context.Context, *connect.Request[emptypb.Empty]) (*connect.Response[emptypb.Empty], error)
}

// NewTestServiceHandler constructs an HTTP handler. It would typically be
// generated from a protobuf schema.
func NewTestServiceHandler(svc TestServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	const root = "/connectauth.example.v1.TestService/"
	mux := http.NewServeMux()
	mux.Handle(root+"GetEmpty", connect.NewUnaryHandler(
		root+"GetEmpty",
		svc.GetEmpty,
		opts...,
	))
	return root, mux
}

// service implements TestServiceHandler. You'd typically hand-write this to
// implement your service's application logic.
type service struct{}

func (s *service) GetEmpty(
	ctx context.Context,
	_ *connect.Request[emptypb.Empty],
) (*connect.Response[emptypb.Empty], error) {
	// Your application logic has access to the authenticated identity.
	fmt.Println(connectauth.GetIdentity(ctx))
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func Example() {
	// We express our authentication logic with a small function.
	authenticate := func(_ context.Context, req *connectauth.Request) (any, error) {
		const magic = "open-sesame"
		if req.Header.Get("Authorization") != "Bearer "+magic {
			// If authentication fails, we return an error. connectauth.Errorf is a
			// convenient shortcut to produce an error coded with
			// connect.CodeUnauthenticated.
			return nil, connectauth.Errorf("try %q as a bearer token instead", magic)
		}
		// Once we've authenticated the request, we can return the authenticated
		// identity. The identity gets attached to the context passed to subsequent
		// interceptors and our service implementation.
		return "Ali Baba", nil
	}
	mux := http.NewServeMux()
	mux.Handle(NewTestServiceHandler(
		&service{},
		connect.WithInterceptors(connectauth.New(authenticate)),
	))
	http.ListenAndServe(":8080", mux)
}

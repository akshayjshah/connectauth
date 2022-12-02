package connectauth_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/akshayjshah/attest"
	"github.com/akshayjshah/connectauth"
	"github.com/bufbuild/connect-go"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestInterceptor(t *testing.T) {
	const hero = "Ali Baba"

	checkIdentity := func(ctx context.Context) error {
		identity := connectauth.GetIdentity(ctx)
		if identity == nil {
			return errors.New("no authenticated identity")
		}
		name, ok := identity.(string)
		if !ok {
			return fmt.Errorf("got identity of type %T, expected string", identity)
		}
		if name != hero {
			return fmt.Errorf("got identity %q, expected %q", name, hero)
		}
		if id := connectauth.GetIdentity(connectauth.WithoutIdentity(ctx)); id != nil {
			return fmt.Errorf("got identity %v after WithoutIdentity", id)
		}
		return nil
	}

	auth := connectauth.New(func(_ context.Context, r *connectauth.Request) (any, error) {
		parts := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
		if len(parts) < 2 || parts[0] != "Bearer" {
			err := connectauth.Errorf("expected Bearer authentication scheme")
			err.Meta().Set("WWW-Authenticate", "Bearer")
			return nil, err
		}
		if tok := parts[1]; tok != "opensesame" {
			return nil, connectauth.Errorf("%q is not the magic passphrase", tok)
		}
		return hero, nil
	})

	mux := http.NewServeMux()
	mux.Handle("/unary", connect.NewUnaryHandler(
		"unary",
		func(ctx context.Context, _ *connect.Request[emptypb.Empty]) (*connect.Response[emptypb.Empty], error) {
			if err := checkIdentity(ctx); err != nil {
				return nil, err
			}
			return connect.NewResponse(&emptypb.Empty{}), nil
		},
		connect.WithInterceptors(auth),
	))
	mux.Handle("/clientstream", connect.NewClientStreamHandler(
		"clientstream",
		func(ctx context.Context, _ *connect.ClientStream[emptypb.Empty]) (*connect.Response[emptypb.Empty], error) {
			if err := checkIdentity(ctx); err != nil {
				return nil, err
			}
			return connect.NewResponse(&emptypb.Empty{}), nil
		},
		connect.WithInterceptors(auth),
	))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	t.Run("unary", func(t *testing.T) {
		client := connect.NewClient[emptypb.Empty, emptypb.Empty](
			srv.Client(),
			srv.URL+"/unary",
		)
		req := connect.NewRequest(&emptypb.Empty{})
		_, err := client.CallUnary(context.Background(), req)
		attest.Error(t, err)
		attest.Equal(t, connect.CodeOf(err), connect.CodeUnauthenticated)
		req.Header().Set("Authorization", "Bearer opensesame")
		_, err = client.CallUnary(context.Background(), req)
		attest.Ok(t, err)
	})

	t.Run("streaming", func(t *testing.T) {
		client := connect.NewClient[emptypb.Empty, emptypb.Empty](
			srv.Client(),
			srv.URL+"/clientstream",
		)
		stream := client.CallClientStream(context.Background())
		stream.Send(nil)
		_, err := stream.CloseAndReceive()
		attest.Error(t, err)
		attest.Equal(t, connect.CodeOf(err), connect.CodeUnauthenticated)

		stream = client.CallClientStream(context.Background())
		stream.RequestHeader().Set("Authorization", "Bearer opensesame")
		stream.Send(nil)
		_, err = stream.CloseAndReceive()
		attest.Ok(t, err)
	})
}

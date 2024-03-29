package connectauth

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"go.akshayshah.org/attest"
	"go.akshayshah.org/memhttp/memhttptest"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	hero       = "Ali Baba"
	passphrase = "opensesame"
)

func assertInfo(tb testing.TB, ctx context.Context) {
	tb.Helper()
	info := GetInfo(ctx)
	if info == nil {
		tb.Fatal("no authentication info")
	}
	name, ok := info.(string)
	attest.True(tb, ok, attest.Sprintf("got info of type %T, expected string", info))
	attest.Equal(tb, name, hero)
	if id := GetInfo(WithoutInfo(ctx)); id != nil {
		tb.Fatalf("got info %v after WithoutInfo", id)
	}
}

func authenticate(ctx context.Context, r *Request) (any, error) {
	parts := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(parts) < 2 || parts[0] != "Bearer" {
		err := Errorf("expected Bearer authentication scheme")
		err.Meta().Set("WWW-Authenticate", "Bearer")
		return nil, err
	}
	if tok := parts[1]; tok != passphrase {
		return nil, Errorf("%q is not the magic passphrase", tok)
	}
	return hero, nil
}

func TestInterceptor(t *testing.T) {
	auth := NewInterceptor(authenticate)
	mux := http.NewServeMux()
	mux.Handle("/unary", connect.NewUnaryHandler(
		"unary",
		func(ctx context.Context, _ *connect.Request[emptypb.Empty]) (*connect.Response[emptypb.Empty], error) {
			assertInfo(t, ctx)
			return connect.NewResponse(&emptypb.Empty{}), nil
		},
		connect.WithInterceptors(auth),
	))
	mux.Handle("/clientstream", connect.NewClientStreamHandler(
		"clientstream",
		func(ctx context.Context, _ *connect.ClientStream[emptypb.Empty]) (*connect.Response[emptypb.Empty], error) {
			assertInfo(t, ctx)
			return connect.NewResponse(&emptypb.Empty{}), nil
		},
		connect.WithInterceptors(auth),
	))
	srv := memhttptest.New(t, mux)

	t.Run("unary", func(t *testing.T) {
		client := connect.NewClient[emptypb.Empty, emptypb.Empty](
			srv.Client(),
			srv.URL()+"/unary",
		)
		req := connect.NewRequest(&emptypb.Empty{})
		_, err := client.CallUnary(context.Background(), req)
		attest.Error(t, err)
		attest.Equal(t, connect.CodeOf(err), connect.CodeUnauthenticated)
		req.Header().Set("Authorization", "Bearer "+passphrase)
		_, err = client.CallUnary(context.Background(), req)
		attest.Ok(t, err)
	})

	t.Run("streaming", func(t *testing.T) {
		client := connect.NewClient[emptypb.Empty, emptypb.Empty](
			srv.Client(),
			srv.URL()+"/clientstream",
		)
		stream := client.CallClientStream(context.Background())
		stream.Send(nil)
		_, err := stream.CloseAndReceive()
		attest.Error(t, err)
		attest.Equal(t, connect.CodeOf(err), connect.CodeUnauthenticated)

		stream = client.CallClientStream(context.Background())
		stream.RequestHeader().Set("Authorization", "Bearer "+passphrase)
		stream.Send(nil)
		_, err = stream.CloseAndReceive()
		attest.Ok(t, err)
	})
}

func TestMiddleware(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Check-Info") != "" {
			assertInfo(t, r.Context())
		}
		io.WriteString(w, "ok")
	})
	srv := memhttptest.New(t, NewMiddleware(authenticate).Wrap(mux))

	assertResponse := func(headers http.Header, expectCode int) {
		req, err := http.NewRequest(
			http.MethodPost,
			srv.URL()+"/empty.v1/GetEmpty",
			strings.NewReader("{}"),
		)
		attest.Ok(t, err)
		for k, vals := range headers {
			for _, v := range vals {
				req.Header.Add(k, v)
			}
		}
		res, err := srv.Client().Do(req)
		attest.Ok(t, err)
		attest.Equal(t, res.StatusCode, expectCode)
	}
	// Middleware should ignore non-RPC requests.
	assertResponse(http.Header{}, 200)
	// RPCs without the right bearer token should be rejected.
	assertResponse(
		http.Header{"Content-Type": []string{"application/json"}},
		http.StatusUnauthorized,
	)
	// RPCs with the right token should be allowed.
	assertResponse(
		http.Header{
			"Content-Type":  []string{"application/json"},
			"Authorization": []string{"Bearer " + passphrase},
			"Check-Info":    []string{"1"}, // verify that auth info is attached to context
		},
		http.StatusOK,
	)
}

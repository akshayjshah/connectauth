connectauth
===========

[![Build](https://github.com/akshayjshah/connectauth/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/akshayjshah/connectauth/actions/workflows/ci.yaml)
[![Report Card](https://goreportcard.com/badge/github.com/akshayjshah/connectauth)](https://goreportcard.com/report/github.com/akshayjshah/connectauth)
[![GoDoc](https://pkg.go.dev/badge/github.com/akshayjshah/connectauth.svg)](https://pkg.go.dev/github.com/akshayjshah/connectauth)

`connectauth` provides an authentication interceptor for
[`connect-go`][connect]. It works with any authentication function and covers
both unary and streaming RPCs.

Using an RPC interceptor makes it easy to send detailed errors to gRPC and
Connect clients: you can assign error codes, add metadata, or attach error
details. However, keep in mind that Connect produces plain `net/http` handlers
that work with _any_ HTTP middleware: for example, you could use [Auth0's HTTP
middleware](https://github.com/auth0/go-jwt-middleware) for JWT validation.

## Status and support

`connectauth` supports the [most recent major release][go-versions] of Go. It's
currently _unstable_, but I hope to cut a stable 1.0 by mid-2023.

## Legal

Offered under the [Apache 2 license][license].

[connect]: https://github.com/bufbuild/connect-go
[go-versions]: https://golang.org/doc/devel/release#policy
[license]: https://github.com/akshayjshah/connectauth/blob/main/LICENSE

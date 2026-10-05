/*
Copyright 2025 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package zitadel

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"
)

// A fake Zitadel to test against.
//
// The client layer is mostly thin gRPC calls, and the interesting part of it -
// which fields are read, which request is built, what a Zitadel failure is
// turned into - is only reachable by actually calling it. Mocking the client
// out would leave all of that untested, so instead a real gRPC server runs in
// the test process and the client is pointed at it over a loopback connection.
//
// A TCP listener rather than bufconn on purpose: a listener can count the
// connections accepted, which is how the "one client, one connection" property
// is checked against what actually happens rather than by reading the code.
//
// Authentication is a static bearer token, so no token exchange is attempted and
// the tests need no issuer.

// callTimeout bounds every call made against the fake, so a test that blocks
// fails rather than hangs.
const callTimeout = 10 * time.Second

// countingListener counts the connections gRPC accepts.
type countingListener struct {
	net.Listener
	accepted *atomic.Int64
}

func (l *countingListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	l.accepted.Add(1)

	return conn, nil
}

// fakeZitadel is an in-process Zitadel gRPC server.
type fakeZitadel struct {
	t        *testing.T
	lis      net.Listener
	srv      *grpc.Server
	accepted atomic.Int64
}

// newFakeZitadel starts a server on loopback and returns it. register wires the
// services a test wants answers from; a surface that is not registered answers
// Unimplemented, which is also what a Zitadel without that surface does.
func newFakeZitadel(t *testing.T, register func(*grpc.Server)) *fakeZitadel {
	t.Helper()

	var lc net.ListenConfig

	lis, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("cannot listen: %v", err)
	}

	f := &fakeZitadel{t: t, lis: lis, srv: grpc.NewServer()}

	if register != nil {
		register(f.srv)
	}

	go func() { _ = f.srv.Serve(&countingListener{Listener: lis, accepted: &f.accepted}) }()

	t.Cleanup(func() {
		f.srv.Stop()
		_ = f.lis.Close()
	})

	return f
}

// address is the host:port the server listens on.
func (f *fakeZitadel) address() string { return f.lis.Addr().String() }

// url is the instance URL a client is configured with.
func (f *fakeZitadel) url() string { return "http://" + f.address() }

// connections reports how many connections have been accepted.
func (f *fakeZitadel) connections() int64 { return f.accepted.Load() }

// config is a client configuration pointing at the fake, authenticating with a
// static token so that nothing has to be issued.
func (f *fakeZitadel) config() Config {
	return Config{
		URL:         f.url(),
		Insecure:    true,
		Credentials: Credentials{Token: "a-personal-access-token"},
	}
}

// client builds a client for the fake.
func (f *fakeZitadel) client(t *testing.T) *Client {
	t.Helper()

	zc, err := NewClient(context.Background(), f.config())
	if err != nil {
		t.Fatalf("cannot build a client for the fake: %v", err)
	}

	t.Cleanup(func() { _ = zc.Close() })

	return zc
}

// testContext returns a context bounded by callTimeout.
func testContext(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	t.Cleanup(cancel)

	return ctx
}

// waitFor polls until cond holds or the deadline passes. It is for waiting on
// something asynchronous in a fake, such as a connection being accepted, which
// happens on the server's goroutine rather than the test's.
func waitFor(t *testing.T, cond func() bool) bool {
	t.Helper()

	deadline := time.Now().Add(callTimeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}

		time.Sleep(10 * time.Millisecond)
	}

	return cond()
}

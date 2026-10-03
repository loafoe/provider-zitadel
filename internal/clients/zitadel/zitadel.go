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

// Package zitadel provides a thin wrapper around the Zitadel Go SDK which
// exposes the API surfaces used by the managed resources of this provider.
//
// The client deliberately hides the gRPC plumbing: every method takes a
// context and typed structs and returns typed structs, so that the controllers
// never have to deal with protobuf one-of wrappers or gRPC status codes
// directly.
package zitadel

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"

	adminapi "github.com/zitadel/zitadel-go/v3/pkg/client/admin"
	apiv2 "github.com/zitadel/zitadel-go/v3/pkg/client/application/v2"
	"github.com/zitadel/zitadel-go/v3/pkg/client/management"
	"github.com/zitadel/zitadel-go/v3/pkg/client/middleware"
	orgv2 "github.com/zitadel/zitadel-go/v3/pkg/client/org/v2"
	projectv2 "github.com/zitadel/zitadel-go/v3/pkg/client/project/v2"
	userv2 "github.com/zitadel/zitadel-go/v3/pkg/client/user/v2"
	"github.com/zitadel/zitadel-go/v3/pkg/client/zitadel"
	actionapi "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/action/v2"
	settingsapi "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/settings/v2"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Scopes requested when authenticating with a service account key.
const (
	// ScopeOpenID is the OIDC openid scope.
	ScopeOpenID = "openid"

	// ScopeZitadelAPI is the audience scope of the ZITADEL project. It puts the
	// ID of the ZITADEL project into the `aud` claim of the issued access token,
	// which is exactly what the Zitadel API checks when it validates a token
	// (`verifyAudience` accepts either the requesting application's client ID or
	// the ZITADEL project ID). Without this scope the token carries the user
	// name as its audience and every API call fails with
	// `Unauthenticated: Errors.Token.Invalid`.
	ScopeZitadelAPI = "urn:zitadel:iam:org:project:id:zitadel:aud"
)

// ErrNotFound is returned by the client when the requested external resource
// does not exist. Controllers translate it into "the managed resource does not
// exist yet" (and therefore triggers Create).
var ErrNotFound = errors.New("zitadel resource not found")

// Credentials holds the authentication material for a Zitadel client. Exactly
// one of ServiceAccountKey and Token must be set.
type Credentials struct {
	// ServiceAccountKey is the JSON encoded machine key of a Zitadel service
	// account, i.e. the key file downloaded from the Zitadel console.
	ServiceAccountKey []byte

	// Token is a Personal Access Token of a Zitadel service account.
	Token string
}

// Config holds everything needed to build a Client.
type Config struct {
	// URL is the base URL (issuer) of the Zitadel instance, e.g.
	// https://my-instance.zitadel.cloud.
	URL string

	// Credentials used to authenticate against the instance.
	Credentials Credentials

	// Insecure disables TLS. Only useful for local development.
	Insecure bool

	// InsecureSkipTLSVerify skips verification of the server certificate.
	InsecureSkipTLSVerify bool

	// OrganizationID sets the organization context used for API calls that do
	// not carry an explicit organization ID.
	OrganizationID string
}

// Client is a Zitadel API client.
type Client struct {
	user        *userv2.Client
	project     *projectv2.Client
	application *apiv2.Client
	org         *orgv2.Client

	// The v2 action service manages where actions are sent and when they run.
	action actionapi.ActionServiceClient

	// The v2 settings service is the only Zitadel surface that reads and writes
	// the instance wide security settings.
	settings     settingsapi.SettingsServiceClient
	settingsConn *zitadel.Connection
	actionConn   *zitadel.Connection

	// The v1 admin API is the only API that manages instance memberships (IAM
	// roles), instance settings and the domain policy. Most of it is instance
	// scoped, but the domain policy and the custom policies take an organization
	// through the connection, so those clients are created per organization and
	// cached the same way the management ones are.
	admin       *adminapi.Client
	adminPerOrg map[string]*adminapi.Client
	adminMu     sync.Mutex

	// The v1 management API is organization scoped and needs a connection per
	// organization, so its clients are created lazily and cached here. See
	// management.go for why the v1 API is used at all.
	issuer       string
	api          string
	options      []zitadel.Option
	management   map[string]*management.Client
	managementMu sync.Mutex
}

// hostFromURL extracts the host (and optional port) from the supplied URL,
// stripping any scheme and path.
func hostFromURL(rawURL string) (host string, err error) {
	u := rawURL
	if !strings.Contains(u, "://") {
		u = "https://" + u
	}

	parsed, err := url.Parse(u)
	if err != nil {
		return "", fmt.Errorf("cannot parse Zitadel URL %q: %w", rawURL, err)
	}

	if parsed.Host == "" {
		return "", fmt.Errorf("zitadel URL %q does not contain a host", rawURL)
	}

	return parsed.Host, nil
}

// issuerFromURL normalises the configured URL into an issuer URL, i.e. it
// strips any path and trailing slash but keeps the scheme.
func issuerFromURL(rawURL string, insecure bool) (string, error) {
	host, err := hostFromURL(rawURL)
	if err != nil {
		return "", err
	}

	scheme := "https://"
	if insecure {
		scheme = "http://"
	}

	return scheme + host, nil
}

// endpoints derives the issuer and the gRPC API endpoint from the configured
// URL, and validates that the configuration carries credentials.
func endpoints(cfg Config) (issuer, api string, err error) {
	if cfg.URL == "" {
		return "", "", errors.New("zitadel URL is required")
	}

	if len(cfg.Credentials.ServiceAccountKey) == 0 && cfg.Credentials.Token == "" {
		return "", "", errors.New("either a service account key or a token is required")
	}

	issuer, err = issuerFromURL(cfg.URL, cfg.Insecure)
	if err != nil {
		return "", "", err
	}

	host, err := hostFromURL(cfg.URL)
	if err != nil {
		return "", "", err
	}

	// The gRPC endpoint is the issuer host on port 443, or on port 80 for
	// insecure (plaintext) connections. An explicitly configured port always
	// wins.
	api = host + ":443"

	switch {
	case strings.Contains(host, ":"):
		api = host
	case cfg.Insecure:
		api = host + ":80"
	}

	return issuer, api, nil
}

// NewClient creates a new Zitadel API client. The returned client opens gRPC
// connections lazily; use Close to release them.
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	issuer, api, err := endpoints(cfg)
	if err != nil {
		return nil, err
	}

	opts := connectionOptions(ctx, cfg, issuer, api)
	scopes := defaultScopes()

	// Every client that is built successfully is tracked, so that a later
	// failure does not leak the connections opened so far.
	var open []*grpc.ClientConn
	closeAll := func() {
		for _, c := range open {
			_ = c.Close()
		}
	}

	u, err := userv2.NewClient(ctx, issuer, api, scopes, opts...)
	if err != nil {
		return nil, fmt.Errorf("cannot create zitadel user client: %w", err)
	}
	open = append(open, u.Connection.ClientConn)

	p, err := projectv2.NewClient(ctx, issuer, api, scopes, opts...)
	if err != nil {
		closeAll()
		return nil, fmt.Errorf("cannot create zitadel project client: %w", err)
	}
	open = append(open, p.Connection.ClientConn)

	a, err := apiv2.NewClient(ctx, issuer, api, scopes, opts...)
	if err != nil {
		closeAll()
		return nil, fmt.Errorf("cannot create zitadel application client: %w", err)
	}
	open = append(open, a.Connection.ClientConn)

	o, err := orgv2.NewClient(ctx, issuer, api, scopes, opts...)
	if err != nil {
		closeAll()
		return nil, fmt.Errorf("cannot create zitadel organization client: %w", err)
	}
	open = append(open, o.Connection.ClientConn)

	ad, err := adminapi.NewClient(ctx, issuer, api, scopes, opts...)
	if err != nil {
		closeAll()
		return nil, fmt.Errorf("cannot create zitadel admin client: %w", err)
	}
	open = append(open, ad.Connection.ClientConn)

	// The action service is generated without a convenience constructor either.
	actConn, err := zitadel.NewConnection(ctx, issuer, api, scopes, opts...)
	if err != nil {
		closeAll()
		return nil, fmt.Errorf("cannot create zitadel action connection: %w", err)
	}

	act := actionapi.NewActionServiceClient(actConn.ClientConn)

	// The settings service is generated without a convenience constructor, so it
	// gets its own connection here rather than sharing one of the others'.
	stConn, err := zitadel.NewConnection(ctx, issuer, api, scopes, opts...)
	if err != nil {
		closeAll()
		return nil, fmt.Errorf("cannot create zitadel settings connection: %w", err)
	}

	st := settingsapi.NewSettingsServiceClient(stConn.ClientConn)

	return &Client{
		user:         u,
		project:      p,
		application:  a,
		org:          o,
		admin:        ad,
		action:       act,
		actionConn:   actConn,
		settings:     st,
		settingsConn: stConn,
		issuer:       issuer,
		api:          api,
		options:      opts,
		management:   map[string]*management.Client{},
	}, nil
}

// defaultScopes are the scopes requested when authenticating against a Zitadel
// instance. The Zitadel API audience scope is required for the issued token to
// be accepted by the API - see ScopeZitadelAPI.
func defaultScopes() []string { return []string{ScopeOpenID, ScopeZitadelAPI} }

// connectionOptions builds the SDK dial options for a client configuration: TLS
// settings, the optional organization context and exactly one authentication
// method - either a static bearer token or a service account key exchanged
// through the JWT Profile grant.
func connectionOptions(ctx context.Context, cfg Config, issuer, api string) []zitadel.Option {
	opts := []zitadel.Option{zitadel.WithCustomURL(issuer, api)}

	if cfg.Insecure {
		opts = append(opts, zitadel.WithInsecure())
	}

	if cfg.InsecureSkipTLSVerify {
		opts = append(opts, zitadel.WithInsecureSkipVerifyTLS())
	}

	if cfg.OrganizationID != "" {
		opts = append(opts, zitadel.WithOrgID(cfg.OrganizationID))
	}

	if cfg.Credentials.Token != "" {
		return append(opts, zitadel.WithTokenSource(oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: strings.TrimSpace(cfg.Credentials.Token),
			TokenType:   "Bearer",
		})))
	}

	return append(opts, zitadel.WithJWTProfileTokenSource(
		middleware.JWTProfileFromFileData(ctx, cfg.Credentials.ServiceAccountKey),
	))
}

// Close releases the gRPC connections held by the client.
func (c *Client) Close() error {
	var errs []error

	for _, conn := range []interface{ Close() error }{c.user.Connection, c.project.Connection, c.application.Connection, c.org.Connection, c.admin.Connection, c.settingsConn, c.actionConn} {
		if conn == nil {
			continue
		}
		if err := conn.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	c.managementMu.Lock()
	for _, mc := range c.management {
		if err := mc.Connection.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	c.management = nil
	c.managementMu.Unlock()

	return errors.Join(errs...)
}

// IsNotFound reports whether err indicates that the external resource does not
// exist.
// IsNotChanged reports whether Zitadel refused a write because it would have
// changed nothing.
//
// Zitadel guards several update endpoints with a precondition that fails when
// the submitted state matches the stored state, which would otherwise put a
// managed resource into a permanent error loop: the controller observes the
// desired state, decides an update is needed, and Zitadel refuses it forever.
func IsNotChanged(err error) bool {
	if err == nil {
		return false
	}

	st, ok := grpcStatus(err)
	if !ok {
		return false
	}

	if st.Code() != codes.FailedPrecondition && st.Code() != codes.InvalidArgument {
		return false
	}

	// Zitadel spells this differently per endpoint: some say "No changes", some
	// carry a "NotChanged" error code, the private label policy phrases it as
	// "has not been changed", and an action that is already active says it "is
	// not inactive".
	msg := st.Message()

	return strings.Contains(msg, "NotChanged") ||
		strings.Contains(msg, "No changes") ||
		strings.Contains(msg, "has not been changed") ||
		strings.Contains(msg, "is not inactive")
}

// grpcStatus extracts the gRPC status of an error.
//
// The error may reach us straight from a gRPC call, in which case it is the
// status itself, or wrapped with context by one of the methods here, in which
// case the status is one level down. Both have to work, because a test that
// recognises a specific Zitadel failure has to recognise it however the caller
// wrapped it.
func grpcStatus(err error) (*status.Status, bool) {
	if err == nil {
		return nil, false
	}

	if st, ok := status.FromError(err); ok {
		return st, true
	}

	if inner := errors.Unwrap(err); inner != nil {
		if st, ok := status.FromError(inner); ok {
			return st, true
		}
	}

	return nil, false
}

func IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, ErrNotFound) {
		return true
	}

	st, ok := grpcStatus(err)
	if !ok {
		return false
	}

	return st.Code() == codes.NotFound
}

// IsNoChanges reports whether err is Zitadel's way of saying that an update
// would not have changed anything. Callers treat that as success.
func IsNoChanges(err error) bool {
	if err == nil {
		return false
	}

	st, ok := status.FromError(err)
	if !ok {
		return false
	}

	return st.Code() == codes.FailedPrecondition && strings.Contains(st.Message(), "No changes")
}

// IsInvalidToken reports whether err indicates that Zitadel rejected the
// credentials of the provider. The most common cause is a service account that
// issues encrypted ("Bearer") access tokens instead of signed ("Jwt") ones: the
// Zitadel API only accepts the latter.
func IsInvalidToken(err error) bool {
	if err == nil {
		return false
	}

	st, ok := status.FromError(err)
	if !ok {
		return false
	}

	return st.Code() == codes.Unauthenticated
}

// ErrInvalidTokenHint is appended to authentication errors so that the most
// common misconfiguration is actionable without reading the Zitadel docs.
const ErrInvalidTokenHint = "Zitadel rejected the credentials of the provider. " +
	"When using a service account key, make sure the machine user is configured " +
	`with the "Jwt" access token type: the Zitadel API rejects the encrypted "Bearer" tokens.`

// WrapError enriches authentication failures with ErrInvalidTokenHint so that
// the most common misconfiguration is actionable without reading the Zitadel
// docs. Every other error is returned unchanged.
func WrapError(err error) error {
	if err == nil {
		return nil
	}

	if IsInvalidToken(err) {
		return errors.New(ErrInvalidTokenHint + ": " + err.Error())
	}

	return err
}

// IsAlreadyExists reports whether Zitadel refused a write because the thing
// being written is already there.
//
// It matters when a policy is written for the first time: Zitadel reports an
// organization as still on the instance default in some cases even though it
// already holds a custom policy, and the write then fails as an "already
// exists". Falling back to the update call is what makes creating a policy
// idempotent rather than dependent on that flag being accurate.
func IsAlreadyExists(err error) bool {
	if err == nil {
		return false
	}

	st, ok := grpcStatus(err)
	if !ok {
		return false
	}

	if st.Code() == codes.AlreadyExists {
		return true
	}

	// Some Zitadel errors carry an AlreadyExists sense but a precondition code.
	return strings.Contains(st.Message(), "AlreadyExists")
}

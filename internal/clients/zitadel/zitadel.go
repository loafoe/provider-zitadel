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
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	adminapi "github.com/zitadel/zitadel-go/v3/pkg/client/admin"
	apiv2 "github.com/zitadel/zitadel-go/v3/pkg/client/application/v2"
	"github.com/zitadel/zitadel-go/v3/pkg/client/management"
	"github.com/zitadel/zitadel-go/v3/pkg/client/middleware"
	orgv2 "github.com/zitadel/zitadel-go/v3/pkg/client/org/v2"
	projectv2 "github.com/zitadel/zitadel-go/v3/pkg/client/project/v2"
	userv2 "github.com/zitadel/zitadel-go/v3/pkg/client/user/v2"
	"github.com/zitadel/zitadel-go/v3/pkg/client/zitadel"
	actionapi "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/action/v2"
	featureapi "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/feature/v2"
	instanceapi "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/instance/v2"
	permissionapi "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/internal_permission/v2"
	orgv2api "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/org/v2"
	settingsapi "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/settings/v2"
	webkeyapi "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/webkey/v2"
	"golang.org/x/oauth2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	adminapiadmin "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/admin"
	applicationv2app "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/application/v2"
	projectv2project "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/project/v2"
	userv2user "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/user/v2"
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
//
// Every service it exposes shares a single gRPC connection and a single token
// source. gRPC multiplexes concurrent RPCs over one HTTP/2 connection, so the
// twelve ZITADEL surfaces this client wraps need one connection between them,
// not one each. See NewClient.
type Client struct {
	user        *userv2.Client
	project     *projectv2.Client
	application *apiv2.Client
	org         *orgv2.Client

	// The v2 action service manages where actions are sent and when they run.
	action actionapi.ActionServiceClient

	// The v2 settings service is the only Zitadel surface that reads and writes
	// the instance wide security settings.
	settings settingsapi.SettingsServiceClient

	// The v2 feature service holds the instance and system wide feature flags,
	// which are the only Zitadel surface that reads or writes them.
	feature featureapi.FeatureServiceClient

	// The v2 instance service holds the custom and trusted domain lists.
	instance instanceapi.InstanceServiceClient

	// The v2beta organization service holds the domains an organization owns.
	// It is on a different service from the v2 organization client above, and is
	// the only surface that can add, list and remove one.
	orgDomain orgv2api.OrganizationServiceClient

	// The v2 webkey service holds Zitadel's own signing keys, which are what an
	// application verifies a token with.
	webkey webkeyapi.WebKeyServiceClient

	// The v2 internal permission service grants roles against a resource, which
	// is how a project membership is written.
	permission permissionapi.InternalPermissionServiceClient

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

	// conn is the one connection every service above shares, and the only one
	// Close has to release.
	conn *zitadel.Connection

	// orgID is the default organization context of the configuration the client
	// was built from, kept so that controllers can read it without a second read
	// of the ProviderConfig.
	orgID string

	// shared is set on a client that a ClientCache owns. Such a client outlives
	// the reconcile that obtained it, so Release must not close it.
	shared bool

	// closeOnce and closeErr make Close idempotent: closing a connection twice
	// fails, and the cache that owns this client may have closed it already.
	closeOnce sync.Once
	closed    atomic.Bool
	closeErr  error
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
//
// NewClient creates a new Zitadel API client. The returned client opens a
// single gRPC connection, shared by every service it exposes; use Close (or
// Release, for a client owned by a ClientCache) to release it.
//
// The twelve ZITADEL surfaces are gRPC services on the same endpoint. The SDK
// offers a convenience constructor per surface, but each of those dials its own
// connection, and every one of them carries its own copy of the authentication
// interceptor. A client built that way would pay twelve TLS handshakes and
// twelve token exchanges for a reconcile that typically touches one or two of
// the surfaces, and then throw all twelve away at the end of it. So the
// connection is built once here and the service stubs are constructed from its
// ClientConn, which is exactly what the convenience constructors do internally.
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	issuer, api, err := endpoints(cfg)
	if err != nil {
		return nil, err
	}

	conn, err := zitadel.NewConnection(ctx, issuer, api, defaultScopes(), connectionOptions(ctx, cfg, issuer, api)...)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to Zitadel at %s: %w", api, err)
	}

	cc := conn.ClientConn

	return &Client{
		conn:       conn,
		issuer:     issuer,
		api:        api,
		orgID:      cfg.OrganizationID,
		options:    connectionOptions(ctx, cfg, issuer, api),
		management: map[string]*management.Client{},

		// The convenience constructors below only wrap a service stub built from a
		// connection, so they are spelled out rather than called: calling them is
		// what dialed a connection per service in the first place.
		user:        &userv2.Client{Connection: conn, UserServiceClient: userv2user.NewUserServiceClient(cc)},
		project:     &projectv2.Client{Connection: conn, ProjectServiceClient: projectv2project.NewProjectServiceClient(cc)},
		application: &apiv2.Client{Connection: conn, ApplicationServiceClient: applicationv2app.NewApplicationServiceClient(cc)},
		org:         &orgv2.Client{Connection: conn, OrganizationServiceClient: orgv2api.NewOrganizationServiceClient(cc)},

		// The v1 admin API is the only API that manages instance memberships (IAM
		// roles), instance settings and the domain policy.
		admin: &adminapi.Client{
			Connection:         conn,
			AdminServiceClient: adminapiadmin.NewAdminServiceClient(cc),
		},

		// The remaining v2 services are generated without a convenience
		// constructor, so their stubs are built directly.
		action:     actionapi.NewActionServiceClient(cc),
		settings:   settingsapi.NewSettingsServiceClient(cc),
		feature:    featureapi.NewFeatureServiceClient(cc),
		instance:   instanceapi.NewInstanceServiceClient(cc),
		orgDomain:  orgv2api.NewOrganizationServiceClient(cc),
		webkey:     webkeyapi.NewWebKeyServiceClient(cc),
		permission: permissionapi.NewInternalPermissionServiceClient(cc),
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

// Close releases the gRPC connections held by the client, including the
// per organization management and admin clients it created lazily.
//
// The main connection is shared by every service, so one call to its Close is
// all that is needed for them. The per organization clients carry their own
// connection because the organization is applied as an interceptor at connect
// time, so they are still one connection each - but they are created only on
// demand and held until the client is closed.
//
// Close is idempotent. A client can be closed by the cache that owns it and then
// closed again by a reconcile that was still holding it, and a second failure to
// close an already closed connection would surface as a reconcile error that
// says nothing useful.
func (c *Client) Close() error {
	c.closeOnce.Do(func() {
		var errs []error

		if c.conn != nil {
			if err := c.conn.Close(); err != nil {
				errs = append(errs, err)
			}
		}

		c.adminMu.Lock()
		for _, ac := range c.adminPerOrg {
			if err := ac.Connection.Close(); err != nil {
				errs = append(errs, err)
			}
		}
		c.adminPerOrg = nil
		c.adminMu.Unlock()

		c.managementMu.Lock()
		for _, mc := range c.management {
			if err := mc.Connection.Close(); err != nil {
				errs = append(errs, err)
			}
		}
		c.management = nil
		c.managementMu.Unlock()

		c.closed.Store(true)
		c.closeErr = errors.Join(errs...)
	})

	return c.closeErr
}

// Closed reports whether the client has been closed and can no longer be used.
//
// A client handed out by a ClientCache can be retired while a reconcile is
// still holding it, so this is how a caller tells a live client from one it
// must rebuild.
func (c *Client) Closed() bool { return c.closed.Load() }

// Release gives up the caller's claim on the client. Controllers call this
// instead of Close, because a client may be shared: one obtained from a
// ClientCache is reused across reconciles and is closed by the cache when the
// provider shuts down, so closing it here would break every other resource
// using the same configuration.
//
// For a client the caller owns, Release is exactly Close.
func (c *Client) Release() {
	if c.shared {
		return
	}

	_ = c.Close()
}

// SetShared marks the client as owned by a cache, which makes Release a no-op.
// It is called by ClientCache and is not part of the controller facing API.
func (c *Client) SetShared() { c.shared = true }

// DefaultOrganizationID reports the organization the ProviderConfig set as the
// default context, or an empty string when it set none. Controllers that need
// the default read it from here rather than fetching the ProviderConfig a
// second time.
func (c *Client) DefaultOrganizationID() string { return c.orgID }

// Config.Key derives the identity of a configuration: two configurations that
// produce the same key are interchangeable and can share one Client.
//
// The credentials are hashed rather than included, so a key can be logged or
// compared without carrying secret material. A rotation of either credential
// changes the key, which is what makes a cached client get rebuilt when the
// secret behind it changes.
func (c Config) Key() string {
	h := sha256.New()

	// Length prefixed, so that no two different configurations can be spelled
	// the same by moving a separator between fields.
	for _, part := range []string{
		c.URL, c.OrganizationID,
		strconv.FormatBool(c.Insecure),
		strconv.FormatBool(c.InsecureSkipTLSVerify),
	} {
		_, _ = h.Write([]byte(strconv.Itoa(len(part))))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(part))
	}

	// The credentials decide which of the two authentication methods is used,
	// and the token is trimmed exactly as connectionOptions trims it, so that a
	// token differing only in surrounding whitespace shares a client.
	switch {
	case c.Credentials.Token != "":
		_, _ = h.Write([]byte("token"))
		_, _ = h.Write([]byte(strings.TrimSpace(c.Credentials.Token)))
	case len(c.Credentials.ServiceAccountKey) > 0:
		_, _ = h.Write([]byte("key"))
		_, _ = h.Write(c.Credentials.ServiceAccountKey)
	default:
		_, _ = h.Write([]byte("none"))
	}

	return hex.EncodeToString(h.Sum(nil))
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
//
// The original error is wrapped rather than flattened into a new message. The
// reconciler sees every error from every controller through this function, so
// flattening would strip the gRPC status off every credential rejection - and
// with it the ability to tell an authentication problem from any other error.
func WrapError(err error) error {
	if err == nil {
		return nil
	}

	if IsInvalidToken(err) {
		return fmt.Errorf("%s: %w", ErrInvalidTokenHint, err)
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

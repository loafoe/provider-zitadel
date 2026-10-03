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
	"fmt"

	"github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/admin"
	apiv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/application/v2"
	settings "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/settings"
)

// SAML applications
//
// Zitadel has one application API and three application types, so a SAML
// application is made and read through the same calls as the API and OIDC ones
// rather than through the older endpoints that only know about SAML.

// SAMLApplication is a SAML application's configuration.
//
// The metadata is what the identity provider publishes, and Zitadel keeps
// whichever form it was given: the document itself or a URL it fetched.
type SAMLApplication struct {
	ID           string
	Name         string
	ProjectID    string
	State        string
	MetadataXML  string
	MetadataURL  string
	LoginVersion string
}

// Input returns the application in the shape the write API accepts.
func (a SAMLApplication) Input() any { return a }

// CreateSAMLApplication makes a SAML application in a project.
func (c *Client) CreateSAMLApplication(ctx context.Context, projectID, name string, in SAMLApplication) (string, error) {
	// Zitadel requires one form of the metadata or the other, so the request is
	// built from whichever the manifest carries.
	req := &apiv2.CreateSAMLApplicationRequest{LoginVersion: loginVersion(in.LoginVersion)}
	switch {
	case in.MetadataXML != "":
		req.Metadata = &apiv2.CreateSAMLApplicationRequest_MetadataXml{MetadataXml: []byte(in.MetadataXML)}
	case in.MetadataURL != "":
		req.Metadata = &apiv2.CreateSAMLApplicationRequest_MetadataUrl{MetadataUrl: in.MetadataURL}
	}

	resp, err := c.application.CreateApplication(ctx, &apiv2.CreateApplicationRequest{
		ProjectId: projectID,
		Name:      name,
		ApplicationType: &apiv2.CreateApplicationRequest_SamlConfiguration{
			SamlConfiguration: req,
		},
	})
	if err != nil {
		return "", err
	}

	return resp.GetApplicationId(), nil
}

// GetSAMLApplication reads a SAML application, or nil when Zitadel does not
// have one with that identifier.
//
// An application of another type answers without a SAML configuration rather
// than with a not found, so a missing configuration is what absence looks like.
func (c *Client) GetSAMLApplication(ctx context.Context, id string) (*SAMLApplication, error) {
	resp, err := c.application.GetApplication(ctx, &apiv2.GetApplicationRequest{ApplicationId: id})
	if err != nil {
		if IsNotFound(err) {
			return nil, nil //nolint:nilnil // an absent application reads as absent
		}

		return nil, err
	}

	app := resp.GetApplication()
	if app == nil {
		return nil, nil //nolint:nilnil // an absent application reads as absent
	}

	cfg, ok := app.GetConfiguration().(*apiv2.Application_SamlConfiguration)
	if !ok || cfg.SamlConfiguration == nil {
		return nil, nil //nolint:nilnil // an application that is not a SAML one has no SAML configuration
	}

	// The state is an enumeration sent as a number, so it is read through its
	// name map rather than converted to a string, which would yield the
	// character with that code point.
	state, known := apiv2.ApplicationState_name[int32(app.GetState())]
	if !known {
		return nil, fmt.Errorf("zitadel reported the unknown application state %d", int32(app.GetState()))
	}

	saml := cfg.SamlConfiguration

	return &SAMLApplication{
		ID:           app.GetApplicationId(),
		Name:         app.GetName(),
		ProjectID:    app.GetProjectId(),
		State:        state,
		MetadataXML:  string(saml.GetMetadataXml()),
		MetadataURL:  saml.GetMetadataUrl(),
		LoginVersion: loginVersionName(saml.GetLoginVersion()),
	}, nil
}

// UpdateSAMLApplication rewrites a SAML application's configuration.
//
// The identifier is not part of the request, so it is carried by the request's
// own field rather than by the path, as Zitadel's v2 API does for updates.
func (c *Client) UpdateSAMLApplication(ctx context.Context, id string, in SAMLApplication) error {
	req := &apiv2.UpdateSAMLApplicationConfigurationRequest{LoginVersion: loginVersion(in.LoginVersion)}
	switch {
	case in.MetadataXML != "":
		req.Metadata = &apiv2.UpdateSAMLApplicationConfigurationRequest_MetadataXml{MetadataXml: []byte(in.MetadataXML)}
	case in.MetadataURL != "":
		req.Metadata = &apiv2.UpdateSAMLApplicationConfigurationRequest_MetadataUrl{MetadataUrl: in.MetadataURL}
	}

	_, err := c.application.UpdateApplication(ctx, &apiv2.UpdateApplicationRequest{
		ApplicationId: id,
		ApplicationType: &apiv2.UpdateApplicationRequest_SamlConfiguration{
			SamlConfiguration: req,
		},
	})

	return err
}

// loginVersion reads the login version Zitadel offers for a SAML application.
//
// It is a one of two rather than a number, so it cannot be looked up in a map
// the way the enumerations elsewhere in this package are.
func loginVersion(name string) *apiv2.LoginVersion {
	switch name {
	case "LoginV1":
		return &apiv2.LoginVersion{Version: &apiv2.LoginVersion_LoginV1{}}
	case "LoginV2":
		return &apiv2.LoginVersion{Version: &apiv2.LoginVersion_LoginV2{}}
	default:
		// Zitadel's own default, which is what an unset version means.
		return &apiv2.LoginVersion{Version: &apiv2.LoginVersion_LoginV1{}}
	}
}

// loginVersionName is loginVersion read the other way.
func loginVersionName(v *apiv2.LoginVersion) string {
	switch v.GetVersion().(type) {
	case *apiv2.LoginVersion_LoginV2:
		return "LoginV2"
	case *apiv2.LoginVersion_LoginV1:
		return "LoginV1"
	default:
		return ""
	}
}

// Message providers
//
// Zitadel sends its codes and notifications through one of these, and each has
// an identity of its own: Zitadel generates a provider ID and it is that ID,
// rather than anything in the configuration, which decides whether a read finds
// the right one.
//
// The read is the same shape for all four - an identifier, a state and a
// description - so one type carries them and each provider only fills in what
// makes it different.

// MessageProvider is what Zitadel reports about a provider it sends through.
type MessageProvider struct {
	// Kind is which of the four this is: `smtp`, `http`, `twilio` or `smsHttp`.
	Kind string

	ID          string
	State       string
	Description string

	// The fields below are what each kind carries, and are empty for the rest.
	SenderAddress    string
	SenderName       string
	ReplyToAddress   string
	Host             string
	User             string
	TLS              bool
	Endpoint         string
	SID              string
	SenderNumber     string
	VerifyServiceSID string

	// Password is the credential the provider authenticates with: an SMTP or
	// Twilio password or token. Zitadel never returns it, so it is carried into
	// a write and never read back.
	Password string
}

// Input returns the provider in the shape the write API accepts.
func (p MessageProvider) Input() any { return p }

// ListEmailProviders returns the providers Zitadel sends email through.
func (c *Client) ListEmailProviders(ctx context.Context) ([]MessageProvider, error) {
	resp, err := c.admin.ListEmailProviders(ctx, &admin.ListEmailProvidersRequest{}) //nolint:staticcheck // the admin API is the only surface that carries the providers
	if err != nil {
		return nil, err
	}

	out := make([]MessageProvider, 0, len(resp.GetResult()))
	for _, p := range resp.GetResult() {
		// The state is an enumeration sent as a number.
		state, known := settings.EmailProviderState_name[int32(p.GetState())]
		if !known {
			return nil, fmt.Errorf("zitadel reported the unknown email provider state %d", int32(p.GetState()))
		}

		m := MessageProvider{
			ID:          p.GetId(),
			State:       state,
			Description: p.GetDescription(),
		}

		switch cfg := p.GetConfig().(type) {
		case *settings.EmailProvider_Smtp:
			m.Kind = "smtp"
			c := cfg.Smtp
			m.SenderAddress, m.SenderName, m.ReplyToAddress = c.GetSenderAddress(), c.GetSenderName(), c.GetReplyToAddress()
			m.Host, m.User, m.TLS = c.GetHost(), c.GetUser(), c.GetTls()

		case *settings.EmailProvider_Http:
			m.Kind = "http"
			m.Endpoint = cfg.Http.GetEndpoint()

		default:
			// A provider of a kind this provider does not model: reported by its
			// identifier and state so it is visible, without guessing at its shape.
			m.Kind = "unknown"
		}

		out = append(out, m)
	}

	return out, nil
}

// ListSMSProviders returns the providers Zitadel sends SMS through.
func (c *Client) ListSMSProviders(ctx context.Context) ([]MessageProvider, error) {
	resp, err := c.admin.ListSMSProviders(ctx, &admin.ListSMSProvidersRequest{}) //nolint:staticcheck // the admin API is the only surface that carries the providers
	if err != nil {
		return nil, err
	}

	out := make([]MessageProvider, 0, len(resp.GetResult()))
	for _, p := range resp.GetResult() {
		// The state is an enumeration sent as a number.
		state, known := settings.SMSProviderConfigState_name[int32(p.GetState())]
		if !known {
			return nil, fmt.Errorf("zitadel reported the unknown sms provider state %d", int32(p.GetState()))
		}

		m := MessageProvider{
			ID:          p.GetId(),
			State:       state,
			Description: p.GetDescription(),
		}

		switch cfg := p.GetConfig().(type) {
		case *settings.SMSProvider_Twilio:
			m.Kind = "twilio"
			c := cfg.Twilio
			m.SID, m.SenderNumber, m.VerifyServiceSID = c.GetSid(), c.GetSenderNumber(), c.GetVerifyServiceSid()

		case *settings.SMSProvider_Http:
			m.Kind = "smsHttp"
			m.Endpoint = cfg.Http.GetEndpoint()

		default:
			m.Kind = "unknown"
		}

		out = append(out, m)
	}

	return out, nil
}

// AddEmailProviderSMTP makes Zitadel send email through an SMTP server.
//
// Zitadel does not connect to the server to accept it: a provider is added and
// only tested when something is sent. So this can be set up, and read back,
// without the server being reachable.
func (c *Client) AddEmailProviderSMTP(ctx context.Context, p MessageProvider) (string, error) {
	req := &admin.AddEmailProviderSMTPRequest{ //nolint:staticcheck // the admin API is the only surface that carries the providers
		User:           p.User,
		SenderAddress:  p.SenderAddress,
		SenderName:     p.SenderName,
		Tls:            p.TLS,
		Host:           p.Host,
		ReplyToAddress: p.ReplyToAddress,
		Description:    p.Description,
	}

	// The user is a field of its own and the oneof carries only the password.
	// Zitadel never returns the password, so an update that carries none leaves
	// the stored one alone.
	if p.Password != "" {
		req.Auth = &admin.AddEmailProviderSMTPRequest_Plain{ //nolint:staticcheck // see above
			Plain: &admin.SMTPPlainAuth{Password: p.Password},
		}
	}

	resp, err := c.admin.AddEmailProviderSMTP(ctx, req) //nolint:staticcheck // see above
	if err != nil {
		return "", err
	}

	return resp.GetId(), nil
}

// UpdateEmailProviderSMTP rewrites an SMTP provider.
func (c *Client) UpdateEmailProviderSMTP(ctx context.Context, id string, p MessageProvider) error {
	req := &admin.UpdateEmailProviderSMTPRequest{ //nolint:staticcheck // the admin API is the only surface that carries the providers
		Id:             id,
		User:           p.User,
		SenderAddress:  p.SenderAddress,
		SenderName:     p.SenderName,
		Tls:            p.TLS,
		Host:           p.Host,
		ReplyToAddress: p.ReplyToAddress,
		Description:    p.Description,
	}

	// The password is only sent when the manifest carries one, because Zitadel
	// never returns it and sending an empty one would clear the stored password.
	if p.Password != "" {
		req.Auth = &admin.UpdateEmailProviderSMTPRequest_Plain{ //nolint:staticcheck // see above
			Plain: &admin.SMTPPlainAuth{Password: p.Password},
		}
	}

	_, err := c.admin.UpdateEmailProviderSMTP(ctx, req) //nolint:staticcheck // see above
	return err
}

// AddEmailProviderHTTP makes Zitadel send email by posting it to an endpoint.
func (c *Client) AddEmailProviderHTTP(ctx context.Context, p MessageProvider) (string, error) {
	resp, err := c.admin.AddEmailProviderHTTP(ctx, &admin.AddEmailProviderHTTPRequest{ //nolint:staticcheck // the admin API is the only surface that carries the providers
		Endpoint:    p.Endpoint,
		Description: p.Description,
	})
	if err != nil {
		return "", err
	}

	return resp.GetId(), nil
}

// UpdateEmailProviderHTTP rewrites an HTTP email provider.
func (c *Client) UpdateEmailProviderHTTP(ctx context.Context, id string, p MessageProvider) error {
	_, err := c.admin.UpdateEmailProviderHTTP(ctx, &admin.UpdateEmailProviderHTTPRequest{ //nolint:staticcheck // the admin API is the only surface that carries the providers
		Id:          id,
		Endpoint:    p.Endpoint,
		Description: p.Description,
	})
	return err
}

// AddSMSProviderHTTP makes Zitadel send SMS by posting it to an endpoint.
func (c *Client) AddSMSProviderHTTP(ctx context.Context, p MessageProvider) (string, error) {
	resp, err := c.admin.AddSMSProviderHTTP(ctx, &admin.AddSMSProviderHTTPRequest{ //nolint:staticcheck // the admin API is the only surface that carries the providers
		Endpoint:    p.Endpoint,
		Description: p.Description,
	})
	if err != nil {
		return "", err
	}

	return resp.GetId(), nil
}

// UpdateSMSProviderHTTP rewrites an HTTP SMS provider.
func (c *Client) UpdateSMSProviderHTTP(ctx context.Context, id string, p MessageProvider) error {
	_, err := c.admin.UpdateSMSProviderHTTP(ctx, &admin.UpdateSMSProviderHTTPRequest{ //nolint:staticcheck // the admin API is the only surface that carries the providers
		Id:          id,
		Endpoint:    p.Endpoint,
		Description: p.Description,
	})
	return err
}

// AddSMSProviderTwilio makes Zitadel send SMS through Twilio.
func (c *Client) AddSMSProviderTwilio(ctx context.Context, p MessageProvider) (string, error) {
	// The Twilio token is a plain field rather than a credential oneof, so it is
	// only sent when the manifest carries one: Zitadel never returns it back.
	req := &admin.AddSMSProviderTwilioRequest{ //nolint:staticcheck // the admin API is the only surface that carries the providers
		Sid:              p.SID,
		Token:            p.Password,
		SenderNumber:     p.SenderNumber,
		Description:      p.Description,
		VerifyServiceSid: p.VerifyServiceSID,
	}

	resp, err := c.admin.AddSMSProviderTwilio(ctx, req) //nolint:staticcheck // see above
	if err != nil {
		return "", err
	}

	return resp.GetId(), nil
}

// UpdateSMSProviderTwilio rewrites a Twilio provider.
func (c *Client) UpdateSMSProviderTwilio(ctx context.Context, id string, p MessageProvider) error {
	_, err := c.admin.UpdateSMSProviderTwilio(ctx, &admin.UpdateSMSProviderTwilioRequest{ //nolint:staticcheck // the admin API is the only surface that carries the providers
		Id:               id,
		Sid:              p.SID,
		SenderNumber:     p.SenderNumber,
		Description:      p.Description,
		VerifyServiceSid: p.VerifyServiceSID,
	})
	if err != nil {
		return err
	}

	// The token is not a field of the update request; it has a call of its own,
	// and is only made when the manifest carries one because Zitadel never
	// returns the stored token.
	if p.Password == "" {
		return nil
	}

	_, err = c.admin.UpdateSMSProviderTwilioToken(ctx, &admin.UpdateSMSProviderTwilioTokenRequest{ //nolint:staticcheck // see above
		Id:    id,
		Token: p.Password,
	})

	return err
}

// RemoveEmailProvider takes an email provider away.
func (c *Client) RemoveEmailProvider(ctx context.Context, id string) error {
	_, err := c.admin.RemoveEmailProvider(ctx, &admin.RemoveEmailProviderRequest{Id: id}) //nolint:staticcheck // the admin API is the only surface that carries the providers
	return err
}

// RemoveSMSProvider takes an SMS provider away.
func (c *Client) RemoveSMSProvider(ctx context.Context, id string) error {
	_, err := c.admin.RemoveSMSProvider(ctx, &admin.RemoveSMSProviderRequest{Id: id}) //nolint:staticcheck // the admin API is the only surface that carries the providers
	return err
}

// SetEmailProviderState activates or deactivates an email provider.
//
// Zitadel only ever sends through an active one, and refuses to deactivate the
// last one it has, so activation is a call of its own rather than part of an
// update.
func (c *Client) SetEmailProviderState(ctx context.Context, id, state string) error {
	active, known := settings.EmailProviderState_value[state]
	if !known {
		return fmt.Errorf("zitadel has no email provider state %q: it must be one of %v",
			state, settings.EmailProviderState_name)
	}

	switch settings.EmailProviderState(active) {
	case settings.EmailProviderState_EMAIL_PROVIDER_ACTIVE:
		_, err := c.admin.ActivateEmailProvider(ctx, &admin.ActivateEmailProviderRequest{Id: id}) //nolint:staticcheck // the admin API is the only surface that carries the providers
		return err

	case settings.EmailProviderState_EMAIL_PROVIDER_INACTIVE,
		// The unspecified state is Zitadel saying nothing, which is left alone
		// rather than guessed at: it would activate or deactivate on a guess.
		settings.EmailProviderState_EMAIL_PROVIDER_STATE_UNSPECIFIED:
		if active == int32(settings.EmailProviderState_EMAIL_PROVIDER_STATE_UNSPECIFIED) {
			return nil
		}

		_, err := c.admin.DeactivateEmailProvider(ctx, &admin.DeactivateEmailProviderRequest{Id: id}) //nolint:staticcheck // see above
		return err

	default:
		return fmt.Errorf("zitadel has no email provider state %q to apply", state)
	}
}

// SetSMSProviderState activates or deactivates an SMS provider.
func (c *Client) SetSMSProviderState(ctx context.Context, id, state string) error {
	active, known := settings.SMSProviderConfigState_value[state]
	if !known {
		return fmt.Errorf("zitadel has no sms provider state %q: it must be one of %v",
			state, settings.SMSProviderConfigState_name)
	}

	switch settings.SMSProviderConfigState(active) {
	case settings.SMSProviderConfigState_SMS_PROVIDER_CONFIG_ACTIVE:
		_, err := c.admin.ActivateSMSProvider(ctx, &admin.ActivateSMSProviderRequest{Id: id}) //nolint:staticcheck // the admin API is the only surface that carries the providers
		return err

	case settings.SMSProviderConfigState_SMS_PROVIDER_CONFIG_INACTIVE,
		// As above: the unspecified state is Zitadel saying nothing.
		settings.SMSProviderConfigState_SMS_PROVIDER_CONFIG_STATE_UNSPECIFIED:
		if active == int32(settings.SMSProviderConfigState_SMS_PROVIDER_CONFIG_STATE_UNSPECIFIED) {
			return nil
		}

		_, err := c.admin.DeactivateSMSProvider(ctx, &admin.DeactivateSMSProviderRequest{Id: id}) //nolint:staticcheck // see above
		return err

	default:
		return fmt.Errorf("zitadel has no sms provider state %q to apply", state)
	}
}

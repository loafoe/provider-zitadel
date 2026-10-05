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

// Package messaging holds the controllers for the four providers Zitadel sends
// its codes and notifications through, and for the SAML application beside the
// API and OIDC ones.
//
// The four providers are one thing four times over: a provider with an identity
// Zitadel generates, a state, and a configuration. They share a harness and
// differ only in which of the six calls they make and which fields they carry.
package messaging

import (
	"context"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
	"github.com/loafoe/provider-zitadel/internal/controller/common"
)

// Setup registers the messaging and SAML controllers.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, s := range []func(ctrl.Manager, controller.Options) error{
		SetupApplicationSAML,
		SetupActiveWebKey,
		SetupEmailProviderSMTP,
		SetupEmailProviderHTTP,
		SetupSMSProviderTwilio,
		SetupSMSProviderHTTP,
	} {
		if err := s(mgr, o); err != nil {
			return err
		}
	}

	return nil
}

// SetupApplicationSAML registers the ApplicationSAML controller.
func SetupApplicationSAML(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		v1alpha1.ApplicationSAMLGroupKind,
		v1alpha1.ApplicationSAMLGroupVersionKind,
		&v1alpha1.ApplicationSAML{},
		&v1alpha1.ApplicationSAMLList{},
		newSAMLExternal,
	)
}

// SetupActiveWebKey registers the ActiveWebKey controller.
func SetupActiveWebKey(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		v1alpha1.ActiveWebKeyGroupKind,
		v1alpha1.ActiveWebKeyGroupVersionKind,
		&v1alpha1.ActiveWebKey{},
		&v1alpha1.ActiveWebKeyList{},
		newActiveWebKeyExternal,
	)
}

// SetupEmailProviderSMTP registers the EmailProviderSMTP controller.
func SetupEmailProviderSMTP(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		v1alpha1.EmailProviderSMTPGroupKind,
		v1alpha1.EmailProviderSMTPGroupVersionKind,
		&v1alpha1.EmailProviderSMTP{},
		&v1alpha1.EmailProviderSMTPList{},
		newProviderExternal(smtpDriver{}),
	)
}

// SetupEmailProviderHTTP registers the EmailProviderHTTP controller.
func SetupEmailProviderHTTP(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		v1alpha1.EmailProviderHTTPGroupKind,
		v1alpha1.EmailProviderHTTPGroupVersionKind,
		&v1alpha1.EmailProviderHTTP{},
		&v1alpha1.EmailProviderHTTPList{},
		newProviderExternal(emailHTTPDriver{}),
	)
}

// SetupSMSProviderTwilio registers the SMSProviderTwilio controller.
func SetupSMSProviderTwilio(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		v1alpha1.SMSProviderTwilioGroupKind,
		v1alpha1.SMSProviderTwilioGroupVersionKind,
		&v1alpha1.SMSProviderTwilio{},
		&v1alpha1.SMSProviderTwilioList{},
		newProviderExternal(twilioDriver{}),
	)
}

// SetupSMSProviderHTTP registers the SMSProviderHTTP controller.
func SetupSMSProviderHTTP(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		v1alpha1.SMSProviderHTTPGroupKind,
		v1alpha1.SMSProviderHTTPGroupVersionKind,
		&v1alpha1.SMSProviderHTTP{},
		&v1alpha1.SMSProviderHTTPList{},
		newProviderExternal(smsHTTPDriver{}),
	)
}

// SAML applications
//
// Zitadel has one application service with three types in it, so this is made
// and read through the same calls as the API and OIDC applications rather than
// through the older endpoints that only know about SAML.

type samlExternal struct {
	kube   client.Client
	client *zitadel.Client
}

func newSAMLExternal(_ context.Context, kube client.Client, _ resource.ModernManaged, zc *zitadel.Client) (managed.ExternalClient, error) {
	return &samlExternal{kube: kube, client: zc}, nil
}

func (e *samlExternal) Disconnect(_ context.Context) error {
	e.client.Release()

	return nil
}

func (e *samlExternal) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.ApplicationSAML)
	if !ok {
		return managed.ExternalObservation{}, errors.New("managed resource is not an ApplicationSAML")
	}

	if _, err := e.project(ctx, cr); err != nil {
		if meta.WasDeleted(cr) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, err
	}

	id := meta.GetExternalName(cr)
	if !common.IsZitadelID(id) {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	app, err := e.client.GetSAMLApplication(ctx, id)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	if app == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cr.Status.AtProvider.ID = app.ID
	cr.Status.AtProvider.Name = app.Name
	// Zitadel addresses the delete by project as well as by application, so the
	// project is recorded while the application is live: a terminating object
	// cannot count on its references still resolving.
	cr.Status.AtProvider.ProjectID = app.ProjectID
	cr.Status.AtProvider.State = app.State
	cr.Status.AtProvider.MetadataXML = app.MetadataXML
	cr.Status.AtProvider.MetadataURL = app.MetadataURL
	cr.Status.AtProvider.LoginVersion = app.LoginVersion
	cr.SetConditions(xpv1.Available())

	fp := cr.Spec.ForProvider
	want := zitadel.SAMLApplication{
		Name:         common.Deref(fp.Name),
		MetadataXML:  common.Deref(fp.MetadataXML),
		MetadataURL:  common.Deref(fp.MetadataURL),
		LoginVersion: common.Deref(fp.LoginVersion),
	}

	return managed.ExternalObservation{
		ResourceExists: true,
		ResourceUpToDate: want.Name == app.Name &&
			want.LoginVersion == app.LoginVersion &&
			// The metadata is compared in whichever form it was given, because
			// Zitadel stores the form it was sent and reads back only that one.
			metadataMatches(want, *app),
	}, nil
}

// metadataMatches compares the metadata in the form it was written.
//
// Zitadel keeps whichever form it was given and reports only that one, so an
// XML document and the URL that serves it are not interchangeable and comparing
// across forms would report drift on a match.
func metadataMatches(want, observed zitadel.SAMLApplication) bool {
	switch {
	case want.MetadataXML != "":
		return want.MetadataXML == observed.MetadataXML
	case want.MetadataURL != "":
		return want.MetadataURL == observed.MetadataURL
	default:
		return true
	}
}

func (e *samlExternal) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.ApplicationSAML)
	if !ok {
		return managed.ExternalCreation{}, errors.New("managed resource is not an ApplicationSAML")
	}

	cr.SetConditions(xpv1.Creating())

	projectID, err := e.project(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	fp := cr.Spec.ForProvider

	id, err := e.client.CreateSAMLApplication(ctx, projectID, common.Deref(fp.Name), zitadel.SAMLApplication{
		MetadataXML:  common.Deref(fp.MetadataXML),
		MetadataURL:  common.Deref(fp.MetadataURL),
		LoginVersion: common.Deref(fp.LoginVersion),
	})
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	meta.SetExternalName(cr, id)

	return managed.ExternalCreation{}, nil
}

func (e *samlExternal) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.ApplicationSAML)
	if !ok {
		return managed.ExternalUpdate{}, errors.New("managed resource is not an ApplicationSAML")
	}

	fp := cr.Spec.ForProvider

	return managed.ExternalUpdate{}, e.client.UpdateSAMLApplication(ctx, meta.GetExternalName(cr), zitadel.SAMLApplication{
		MetadataXML:  common.Deref(fp.MetadataXML),
		MetadataURL:  common.Deref(fp.MetadataURL),
		LoginVersion: common.Deref(fp.LoginVersion),
	})
}

func (e *samlExternal) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.ApplicationSAML)
	if !ok {
		return managed.ExternalDelete{}, errors.New("managed resource is not an ApplicationSAML")
	}

	cr.SetConditions(xpv1.Deleting())

	// The project was recorded while the application was live, because Zitadel
	// refuses a delete that does not carry it and a terminating object cannot
	// count on its references still resolving.
	projectID := cr.Status.AtProvider.ProjectID
	if projectID == "" {
		resolved, err := e.project(ctx, cr)
		switch {
		case err == nil:
			projectID = resolved

		case common.IsReferenceGone(err):
			// The project is gone, and with it the application.
			return managed.ExternalDelete{}, nil

		default:
			return managed.ExternalDelete{}, err
		}
	}

	return managed.ExternalDelete{}, e.client.DeleteApplication(ctx, meta.GetExternalName(cr), projectID)
}

func (e *samlExternal) project(ctx context.Context, cr *v1alpha1.ApplicationSAML) (string, error) {
	fp := cr.Spec.ForProvider

	orgDefault, err := common.ProviderConfigOrganizationID(ctx, e.kube, cr)
	if err != nil {
		return "", common.Join(common.ErrNoOrganizationID, err)
	}

	// A terminating object acts on the organization it recorded: it exists to
	// undo what it did, not one belonging to a replacement.
	if _, err := common.ResolveOrganizationID(ctx, e.kube, cr, fp.OrganizationRef,
		fp.OrganizationSelector, fp.OrganizationID, orgDefault,
		common.CurrentIfDeleting(meta.WasDeleted(cr), fp.OrganizationID)); err != nil {
		return "", common.Join(common.ErrNoOrganizationID, err)
	}

	projectID, err := common.ResolveProjectID(ctx, e.kube, cr, fp.ProjectRef,
		fp.ProjectSelector, fp.ProjectID, "")
	if err != nil {
		return "", common.Join(common.ErrNoProjectID, err)
	}

	return projectID, nil
}

// The active signing key
//
// There is one active key on an instance, so this is a pointer rather than a
// resource of its own: activating the key it names is all it does, and there is
// nothing to create or remove.

type activeWebKeyExternal struct {
	kube   client.Client
	client *zitadel.Client
}

func newActiveWebKeyExternal(_ context.Context, kube client.Client, _ resource.ModernManaged, zc *zitadel.Client) (managed.ExternalClient, error) {
	return &activeWebKeyExternal{kube: kube, client: zc}, nil
}

func (e *activeWebKeyExternal) Disconnect(_ context.Context) error {
	e.client.Release()

	return nil
}

func (e *activeWebKeyExternal) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.ActiveWebKey)
	if !ok {
		return managed.ExternalObservation{}, errors.New("managed resource is not an ActiveWebKey")
	}

	keyID, err := e.key(ctx, cr)
	if err != nil {
		if meta.WasDeleted(cr) {
			// Nothing resolves while an object is terminating, so report the key
			// as gone.
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, err
	}

	// This resource names a key rather than being one, so there is nothing for a
	// terminating object to wait for. Reporting the key as still there would
	// hold the finalizer open for a resource that has nothing to remove.
	if meta.WasDeleted(cr) {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	k, err := e.client.GetWebKey(ctx, keyID)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	if k == nil {
		return managed.ExternalObservation{}, errors.Wrapf(errNoSuchKey, "%s", keyID)
	}

	cr.Status.AtProvider.ID = k.ID
	cr.Status.AtProvider.Algorithm = k.Algorithm
	cr.Status.AtProvider.State = k.State
	cr.Status.WebKeyID = k.ID
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists: true,
		// An active key is the one Zitadel signs with, so this is up to date
		// exactly when the named key is the active one.
		ResourceUpToDate: k.State == "STATE_ACTIVE",
	}, nil
}

var errNoSuchKey = errors.New("zitadel has no signing key with that identifier")

func (e *activeWebKeyExternal) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.ActiveWebKey)
	if !ok {
		return managed.ExternalCreation{}, errors.New("managed resource is not an ActiveWebKey")
	}

	cr.SetConditions(xpv1.Creating())

	keyID, err := e.key(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	// There is nothing to record as an external name: the key already exists and
	// is identified by its own identifier, not by anything this resource made.
	meta.SetExternalName(cr, keyID)
	cr.Status.WebKeyID = keyID

	return managed.ExternalCreation{}, nil
}

func (e *activeWebKeyExternal) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.ActiveWebKey)
	if !ok {
		return managed.ExternalUpdate{}, errors.New("managed resource is not an ActiveWebKey")
	}

	keyID, err := e.key(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	cr.Status.WebKeyID = keyID

	return managed.ExternalUpdate{}, e.client.ActivateWebKey(ctx, keyID)
}

// Delete is a no-op: there is nothing to remove. A key is deactivated by another
// key being activated, and Zitadel has no call to say "activate nothing", so
// this resource can only ever name one.
//
// Deleting it therefore leaves the instance signing with whatever key was last
// activated, which is recorded here rather than guessed at.
func (e *activeWebKeyExternal) Delete(_ context.Context, _ resource.Managed) (managed.ExternalDelete, error) {
	return managed.ExternalDelete{}, nil
}

func (e *activeWebKeyExternal) key(ctx context.Context, cr *v1alpha1.ActiveWebKey) (string, error) {
	fp := cr.Spec.ForProvider

	// A terminating object must not follow a reference that has since moved: it
	// acted on the key it recorded.
	recorded := cr.Status.WebKeyID
	if meta.WasDeleted(cr) && recorded != "" {
		return recorded, nil
	}

	// The reference and the selector are values rather than pointers, so an
	// unset one is not nil. Passing the empty value on would send the resolver
	// looking for a WebKey with no name in no namespace, rather than falling
	// through to the identifier.
	return common.ResolveWebKeyID(ctx, e.kube, cr,
		refOrNil(fp.WebKeyRef), selOrNil(fp.WebKeySelector), fp.WebKeyID)
}

// The four messaging providers
//
// A provider is a thing Zitadel generates an identifier for and then holds as
// one of a list per kind, so the shape is its own rather than either of the two
// harnesses: not a singleton, because there can be several, and not a name in a
// list of names, because a provider has a configuration and a state.
//
// What is shared is everything around the calls - how the provider is found, what
// a credential that Zitadel never returns means for comparison, and why the
// state is applied separately - so that lives in one external client and the four
// kinds supply the five calls.

type providerExternal struct {
	kube   client.Client
	client *zitadel.Client
	d      providerDriver
}

// providerDriver is everything that differs between one provider kind and the
// next.
type providerDriver interface {
	// Kind names the provider, so errors and events say which one failed.
	Kind() string

	// List reads every provider of this kind.
	List(ctx context.Context, c *zitadel.Client) ([]zitadel.MessageProvider, error)

	// Add makes a provider of this kind and returns the identifier Zitadel gave
	// it.
	Add(ctx context.Context, c *zitadel.Client, p zitadel.MessageProvider) (string, error)

	// Update rewrites a provider of this kind.
	Update(ctx context.Context, c *zitadel.Client, id string, p zitadel.MessageProvider) error

	// Remove takes a provider away.
	Remove(ctx context.Context, c *zitadel.Client, id string) error

	// SetState activates or deactivates one.
	SetState(ctx context.Context, c *zitadel.Client, id, state string) error

	// Desired reads the configuration the manifest asks for, resolving a
	// credential from a secret.
	Desired(ctx context.Context, kube client.Client, cr resource.Managed) (zitadel.MessageProvider, error)

	// Report writes what Zitadel reported into the status.
	Report(cr resource.Managed, observed zitadel.MessageProvider)

	// State reads the state the manifest asks for, or the empty string when it
	// asks for none.
	State(cr resource.Managed) string
}

func newProviderExternal(d providerDriver) func(context.Context, client.Client, resource.ModernManaged, *zitadel.Client) (managed.ExternalClient, error) {
	return func(_ context.Context, kube client.Client, _ resource.ModernManaged, zc *zitadel.Client) (managed.ExternalClient, error) {
		return &providerExternal{kube: kube, client: zc, d: d}, nil
	}
}

func (e *providerExternal) Disconnect(_ context.Context) error {
	e.client.Release()

	return nil
}

func (e *providerExternal) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	id := meta.GetExternalName(mg)

	// The identifier has to be one Zitadel handed out. Crossplane seeds the
	// external name with the object's own name, so anything else means this has
	// not created a provider yet.
	if !common.IsZitadelID(id) {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	providers, err := e.d.List(ctx, e.client)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	var found *zitadel.MessageProvider
	for i := range providers {
		if providers[i].ID == id {
			found = &providers[i]

			break
		}
	}

	if found == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	e.d.Report(mg, *found)
	mg.SetConditions(xpv1.Available())

	want, err := e.d.Desired(ctx, e.kube, mg)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	// The state is compared separately because it is applied by a call of its
	// own, and a provider can be correctly configured and switched off.
	state := e.d.State(mg)

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: equal(want, *found, state, e.d),
	}, nil
}

// equal compares a desired provider against what Zitadel reports.
//
// The credential is left out: Zitadel never returns it, so comparing it would
// report drift on every reconcile for a provider that is configured correctly,
// and there would be no way to settle it.
//
//nolint:gocyclo // one arm per provider kind, which is the shape of the comparison itself.
func equal(want, observed zitadel.MessageProvider, state string, _ providerDriver) bool {
	if state != "" && observed.State != state {
		return false
	}

	// Only the fields this kind actually carries are compared, because Zitadel
	// reports the rest empty and a provider written without them would be turned
	// off on every reconcile.
	switch want.Kind {
	case "smtp":
		return want.Description == observed.Description &&
			want.Host == observed.Host &&
			want.User == observed.User &&
			want.SenderAddress == observed.SenderAddress &&
			want.SenderName == observed.SenderName &&
			want.ReplyToAddress == observed.ReplyToAddress &&
			// The TLS flag is only compared when the manifest says something
			// about it: leaving it out must not mean "off".
			(!want.TLSSet || want.TLS == observed.TLS)

	case "twilio":
		return want.Description == observed.Description &&
			want.SID == observed.SID &&
			want.SenderNumber == observed.SenderNumber &&
			want.VerifyServiceSID == observed.VerifyServiceSID

	default:
		return want.Description == observed.Description &&
			want.Endpoint == observed.Endpoint
	}
}

func (e *providerExternal) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	mg.SetConditions(xpv1.Creating())

	want, err := e.d.Desired(ctx, e.kube, mg)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	id, err := e.d.Add(ctx, e.client, want)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	meta.SetExternalName(mg, id)

	// Zitadel adds a provider active or inactive depending on what it is, so the
	// state is applied afterwards rather than assumed.
	if state := e.d.State(mg); state != "" {
		if err := e.d.SetState(ctx, e.client, id, state); err != nil {
			return managed.ExternalCreation{}, err
		}
	}

	return managed.ExternalCreation{}, nil
}

func (e *providerExternal) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	want, err := e.d.Desired(ctx, e.kube, mg)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	id := meta.GetExternalName(mg)

	// Zitadel refuses to rewrite a provider that is active - AlreadyActive for
	// SMTP, and the same rule for the others - so an update to one has to take it
	// out of use, write it, and put it back. Doing it in that order rather than
	// fighting the refusal is also what an operator would do by hand.
	if e.active(ctx, mg) {
		if err := e.d.SetState(ctx, e.client, id, string(v1alpha1.MessageProviderStateInactive)); err != nil {
			return managed.ExternalUpdate{}, err
		}
	}

	if err := e.d.Update(ctx, e.client, id, want); err != nil {
		return managed.ExternalUpdate{}, err
	}

	if state := e.d.State(mg); state != "" {
		if err := e.d.SetState(ctx, e.client, id, state); err != nil {
			return managed.ExternalUpdate{}, err
		}
	}

	return managed.ExternalUpdate{}, nil
}

// active reports whether Zitadel is sending through this provider right now.
func (e *providerExternal) active(ctx context.Context, mg resource.Managed) bool {
	providers, err := e.d.List(ctx, e.client)
	if err != nil {
		// A read that failed is not evidence of anything, so an update is left to
		// make its own attempt rather than being skipped.
		return false
	}

	id := meta.GetExternalName(mg)
	for _, p := range providers {
		if p.ID == id {
			return p.State == string(v1alpha1.MessageProviderStateActive)
		}
	}

	return false
}

func (e *providerExternal) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	mg.SetConditions(xpv1.Deleting())

	return managed.ExternalDelete{}, e.d.Remove(ctx, e.client, meta.GetExternalName(mg))
}

// The four drivers
//
// They differ only in which five calls they make and which fields they carry, so
// each is a handful of lines.

type smtpDriver struct{}

var _ providerDriver = smtpDriver{}

func (smtpDriver) Kind() string { return "EmailProviderSMTP" }

func (smtpDriver) List(ctx context.Context, c *zitadel.Client) ([]zitadel.MessageProvider, error) {
	providers, err := c.ListEmailProviders(ctx)
	if err != nil {
		return nil, err
	}

	// Zitadel lists an instance wide provider and an organization one in the same
	// call. Only the SMTP ones are this kind's business.
	return ofKind(providers, "smtp"), nil
}

func (smtpDriver) Add(ctx context.Context, c *zitadel.Client, p zitadel.MessageProvider) (string, error) {
	return c.AddEmailProviderSMTP(ctx, p)
}

func (smtpDriver) Update(ctx context.Context, c *zitadel.Client, id string, p zitadel.MessageProvider) error {
	return c.UpdateEmailProviderSMTP(ctx, id, p)
}

func (smtpDriver) Remove(ctx context.Context, c *zitadel.Client, id string) error {
	return c.RemoveEmailProvider(ctx, id)
}

func (smtpDriver) SetState(ctx context.Context, c *zitadel.Client, id, state string) error {
	return c.SetEmailProviderState(ctx, id, state)
}

func (smtpDriver) Desired(ctx context.Context, kube client.Client, mg resource.Managed) (zitadel.MessageProvider, error) {
	cr, ok := mg.(*v1alpha1.EmailProviderSMTP)
	if !ok {
		return zitadel.MessageProvider{}, errors.New("managed resource is not an EmailProviderSMTP")
	}

	fp := cr.Spec.ForProvider

	return zitadel.MessageProvider{
		Kind:           "smtp",
		Description:    common.Deref(fp.Description),
		Host:           common.Deref(fp.Host),
		User:           common.Deref(fp.User),
		Password:       common.SecretValue(ctx, kube, cr.GetNamespace(), fp.PasswordSecretRef),
		SenderAddress:  common.Deref(fp.SenderAddress),
		SenderName:     common.Deref(fp.SenderName),
		ReplyToAddress: common.Deref(fp.ReplyToAddress),
		TLS:            common.DerefBool(fp.TLS),
		TLSSet:         fp.TLS != nil,
	}, nil
}

func (smtpDriver) Report(mg resource.Managed, observed zitadel.MessageProvider) {
	cr, ok := mg.(*v1alpha1.EmailProviderSMTP)
	if !ok {
		return
	}

	cr.Status.AtProvider.ID = observed.ID
	cr.Status.AtProvider.State = observed.State
	cr.Status.AtProvider.Kind = observed.Kind
	cr.Status.AtProvider.Description = observed.Description
	cr.Status.AtProvider.Host = observed.Host
	cr.Status.AtProvider.User = observed.User
	cr.Status.AtProvider.SenderAddress = observed.SenderAddress
	cr.Status.AtProvider.SenderName = observed.SenderName
	cr.Status.AtProvider.ReplyToAddress = observed.ReplyToAddress
	cr.Status.AtProvider.TLS = observed.TLS
}

func (smtpDriver) State(mg resource.Managed) string {
	cr, ok := mg.(*v1alpha1.EmailProviderSMTP)
	if !ok {
		return ""
	}

	return providerState(cr.Spec.ForProvider.State)
}

type emailHTTPDriver struct{}

var _ providerDriver = emailHTTPDriver{}

func (emailHTTPDriver) Kind() string { return "EmailProviderHTTP" }

func (emailHTTPDriver) List(ctx context.Context, c *zitadel.Client) ([]zitadel.MessageProvider, error) {
	providers, err := c.ListEmailProviders(ctx)
	if err != nil {
		return nil, err
	}

	return ofKind(providers, "http"), nil
}

func (emailHTTPDriver) Add(ctx context.Context, c *zitadel.Client, p zitadel.MessageProvider) (string, error) {
	return c.AddEmailProviderHTTP(ctx, p)
}

func (emailHTTPDriver) Update(ctx context.Context, c *zitadel.Client, id string, p zitadel.MessageProvider) error {
	return c.UpdateEmailProviderHTTP(ctx, id, p)
}

func (emailHTTPDriver) Remove(ctx context.Context, c *zitadel.Client, id string) error {
	return c.RemoveEmailProvider(ctx, id)
}

func (emailHTTPDriver) SetState(ctx context.Context, c *zitadel.Client, id, state string) error {
	return c.SetEmailProviderState(ctx, id, state)
}

func (emailHTTPDriver) Desired(_ context.Context, _ client.Client, mg resource.Managed) (zitadel.MessageProvider, error) {
	cr, ok := mg.(*v1alpha1.EmailProviderHTTP)
	if !ok {
		return zitadel.MessageProvider{}, errors.New("managed resource is not an EmailProviderHTTP")
	}

	fp := cr.Spec.ForProvider

	return zitadel.MessageProvider{
		Kind:        "http",
		Description: common.Deref(fp.Description),
		Endpoint:    common.Deref(fp.Endpoint),
	}, nil
}

func (emailHTTPDriver) Report(mg resource.Managed, observed zitadel.MessageProvider) {
	cr, ok := mg.(*v1alpha1.EmailProviderHTTP)
	if !ok {
		return
	}

	cr.Status.AtProvider.ID = observed.ID
	cr.Status.AtProvider.State = observed.State
	cr.Status.AtProvider.Kind = observed.Kind
	cr.Status.AtProvider.Description = observed.Description
	cr.Status.AtProvider.Endpoint = observed.Endpoint
}

func (emailHTTPDriver) State(mg resource.Managed) string {
	cr, ok := mg.(*v1alpha1.EmailProviderHTTP)
	if !ok {
		return ""
	}

	return providerState(cr.Spec.ForProvider.State)
}

type twilioDriver struct{}

var _ providerDriver = twilioDriver{}

func (twilioDriver) Kind() string { return "SMSProviderTwilio" }

func (twilioDriver) List(ctx context.Context, c *zitadel.Client) ([]zitadel.MessageProvider, error) {
	providers, err := c.ListSMSProviders(ctx)
	if err != nil {
		return nil, err
	}

	return ofKind(providers, "twilio"), nil
}

func (twilioDriver) Add(ctx context.Context, c *zitadel.Client, p zitadel.MessageProvider) (string, error) {
	return c.AddSMSProviderTwilio(ctx, p)
}

func (twilioDriver) Update(ctx context.Context, c *zitadel.Client, id string, p zitadel.MessageProvider) error {
	return c.UpdateSMSProviderTwilio(ctx, id, p)
}

func (twilioDriver) Remove(ctx context.Context, c *zitadel.Client, id string) error {
	return c.RemoveSMSProvider(ctx, id)
}

func (twilioDriver) SetState(ctx context.Context, c *zitadel.Client, id, state string) error {
	return c.SetSMSProviderState(ctx, id, state)
}

func (twilioDriver) Desired(ctx context.Context, kube client.Client, mg resource.Managed) (zitadel.MessageProvider, error) {
	cr, ok := mg.(*v1alpha1.SMSProviderTwilio)
	if !ok {
		return zitadel.MessageProvider{}, errors.New("managed resource is not an SMSProviderTwilio")
	}

	fp := cr.Spec.ForProvider

	return zitadel.MessageProvider{
		Kind:             "twilio",
		Description:      common.Deref(fp.Description),
		SID:              common.Deref(fp.SID),
		Password:         common.SecretValue(ctx, kube, cr.GetNamespace(), fp.TokenSecretRef),
		SenderNumber:     common.Deref(fp.SenderNumber),
		VerifyServiceSID: common.Deref(fp.VerifyServiceSID),
	}, nil
}

func (twilioDriver) Report(mg resource.Managed, observed zitadel.MessageProvider) {
	cr, ok := mg.(*v1alpha1.SMSProviderTwilio)
	if !ok {
		return
	}

	cr.Status.AtProvider.ID = observed.ID
	cr.Status.AtProvider.State = observed.State
	cr.Status.AtProvider.Kind = observed.Kind
	cr.Status.AtProvider.Description = observed.Description
	cr.Status.AtProvider.SID = observed.SID
	cr.Status.AtProvider.SenderNumber = observed.SenderNumber
	cr.Status.AtProvider.VerifyServiceSID = observed.VerifyServiceSID
}

func (twilioDriver) State(mg resource.Managed) string {
	cr, ok := mg.(*v1alpha1.SMSProviderTwilio)
	if !ok {
		return ""
	}

	return providerState(cr.Spec.ForProvider.State)
}

type smsHTTPDriver struct{}

var _ providerDriver = smsHTTPDriver{}

func (smsHTTPDriver) Kind() string { return "SMSProviderHTTP" }

func (smsHTTPDriver) List(ctx context.Context, c *zitadel.Client) ([]zitadel.MessageProvider, error) {
	providers, err := c.ListSMSProviders(ctx)
	if err != nil {
		return nil, err
	}

	return ofKind(providers, "smsHttp"), nil
}

func (smsHTTPDriver) Add(ctx context.Context, c *zitadel.Client, p zitadel.MessageProvider) (string, error) {
	return c.AddSMSProviderHTTP(ctx, p)
}

func (smsHTTPDriver) Update(ctx context.Context, c *zitadel.Client, id string, p zitadel.MessageProvider) error {
	return c.UpdateSMSProviderHTTP(ctx, id, p)
}

func (smsHTTPDriver) Remove(ctx context.Context, c *zitadel.Client, id string) error {
	return c.RemoveSMSProvider(ctx, id)
}

func (smsHTTPDriver) SetState(ctx context.Context, c *zitadel.Client, id, state string) error {
	return c.SetSMSProviderState(ctx, id, state)
}

func (smsHTTPDriver) Desired(_ context.Context, _ client.Client, mg resource.Managed) (zitadel.MessageProvider, error) {
	cr, ok := mg.(*v1alpha1.SMSProviderHTTP)
	if !ok {
		return zitadel.MessageProvider{}, errors.New("managed resource is not an SMSProviderHTTP")
	}

	fp := cr.Spec.ForProvider

	return zitadel.MessageProvider{
		Kind:        "smsHttp",
		Description: common.Deref(fp.Description),
		Endpoint:    common.Deref(fp.Endpoint),
	}, nil
}

func (smsHTTPDriver) Report(mg resource.Managed, observed zitadel.MessageProvider) {
	cr, ok := mg.(*v1alpha1.SMSProviderHTTP)
	if !ok {
		return
	}

	cr.Status.AtProvider.ID = observed.ID
	cr.Status.AtProvider.State = observed.State
	cr.Status.AtProvider.Kind = observed.Kind
	cr.Status.AtProvider.Description = observed.Description
	cr.Status.AtProvider.Endpoint = observed.Endpoint
}

func (smsHTTPDriver) State(mg resource.Managed) string {
	cr, ok := mg.(*v1alpha1.SMSProviderHTTP)
	if !ok {
		return ""
	}

	return providerState(cr.Spec.ForProvider.State)
}

// refOrNil and selOrNil turn an unset reference or selector into nil, which is
// how the resolver is told to fall through to the plain identifier.
func refOrNil(r xpv1.Reference) *xpv1.Reference {
	if r.Name == "" {
		return nil
	}

	return &r
}

func selOrNil(s xpv1.Selector) *xpv1.Selector {
	if s.MatchLabels == nil && s.MatchControllerRef == nil {
		return nil
	}

	return &s
}

// ofKind keeps only the providers of one kind.
//
// Zitadel lists an instance wide provider and an organization one in the same
// call, and they share an identifier space, so a driver that took all of them
// would claim both.
func ofKind(providers []zitadel.MessageProvider, kind string) []zitadel.MessageProvider {
	out := make([]zitadel.MessageProvider, 0, len(providers))
	for _, p := range providers {
		if p.Kind == kind {
			out = append(out, p)
		}
	}

	return out
}

// providerState is the state a manifest asks for, or the empty string when it
// asks for none: Zitadel's own choice is left alone rather than fought over.
func providerState(s *v1alpha1.MessageProviderState) string {
	if s == nil {
		return ""
	}

	return string(*s)
}

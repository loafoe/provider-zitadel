#!/usr/bin/env python3
"""Generate the identity provider managed resources.

Zitadel has twelve provider types at two levels - instance wide and per
organization - which is twenty three kinds that are the same resource with a
different field set. They are generated from the table below so that a kind's
spec, its status, its controller and its example cannot drift apart.

Two things follow from Zitadel and are encoded in the table rather than left to
each kind:

* only an OIDC or a JWT provider has settings Zitadel reads back, so those are
  the only kinds whose settings are compared and can be updated;
* a provider that belongs to an organization carries a reference to it; an
  instance wide one does not.
"""

import os
import sys

LICENSE = """/*
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
"""

IMPORTS = '''import (
	"reflect"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)'''


def wrap(doc, indent="\t"):
    import textwrap
    return "\n".join(indent + "// " + l for l in textwrap.wrap(" ".join(doc.split()), width=76 - len(indent)))


def f(name, json_name, type_, doc):
    return {"name": name, "json": json_name, "type": type_, "doc": doc}


def fs(go, json_, type_, doc, client=None):
    """A field of a provider type.

    `client` is the name of the field on the client input, which is the Go name
    unless Zitadel spells it differently.
    """
    return {"name": go, "json": json_, "type": type_, "doc": doc, "client": client or go}


STR = "*string"
BOOL = "*bool"
INT = "*int64"
STRS = "[]string"

# Fields every provider has.
COMMON = [
    f("Name", "name", STR,
      "Name of the provider, as it appears on the login page."),
    f("State", "state", "*IDPState",
      "Whether the provider is offered on the login page. One of `Active` or "
      "`Inactive`. An inactive provider keeps its settings and its user links. "
      "Defaults to `Active`."),
    f("Scopes", "scopes", STRS,
      "The OAuth scopes requested from the provider, such as `openid email profile`."),
    f("IsLinkingAllowed", "isLinkingAllowed", BOOL,
      "Let an existing Zitadel account be linked to a new external one that has "
      "the same email address."),
    f("IsCreationAllowed", "isCreationAllowed", BOOL,
      "Let a user be created on first login through this provider."),
    f("IsAutoCreation", "isAutoCreation", BOOL,
      "Create the user without asking, when the provider says the email address "
      "is verified. Needs `isCreationAllowed`."),
    f("IsAutoUpdate", "isAutoUpdate", BOOL,
      "Refresh a linked user's profile on every login, rather than only when "
      "they log in to an existing account."),
    f("AutoLinking", "autoLinking", "*IDPAutoLinking",
      "How an account is matched without asking: by `Email` or by `Username`. "
      "Needs `isLinkingAllowed`."),
]

# The fields of each provider type, beyond the common ones.
# (field, json, go type, doc, provider field name in the client)
TYPES = {
    "OIDC": dict(
        kind="OIDC", readable=True,
        purpose="log in through any OpenID Connect provider",
        fields=[
            fs("Issuer", "issuer", STR, "The issuer URL. Zitadel discovers the provider's endpoints from it.", "Issuer"),
            fs("ClientID", "clientID", STR, "The OAuth client ID issued by the provider.", "ClientID"),
            fs("ClientSecret", "clientSecretSecretRef", "*SecretKeySelector",
               "The OAuth client secret, read from a secret. Zitadel never returns "
               "it, so it is applied but never compared.", "ClientSecret"),
            fs("IsIDTokenMapping", "isIDTokenMapping", BOOL,
               "Build the user's profile from the ID token instead of calling the "
               "userinfo endpoint.", "IsIDTokenMapping"),
            fs("UsePKCE", "usePKCE", BOOL,
               "Use proof key for code exchange, which public clients require.", "UsePKCE"),
        ],
        example=[
            ("issuer", '"https://accounts.example.com"'),
            ("clientID", '"my-client-id"'),
            ("scopes", '["openid", "email", "profile"]'),
        ],
    ),
    "OAuth": dict(
        kind="OAuth", readable=False,
        purpose="log in through a generic OAuth 2.0 provider",
        fields=[
            fs("AuthorizationEndpoint", "authorizationEndpoint", STR,
               "Where the browser is sent to log in.", "AuthorizationEndpoint"),
            fs("TokenEndpoint", "tokenEndpoint", STR,
               "Where the code is exchanged for a token.", "TokenEndpoint"),
            fs("UserEndpoint", "userEndpoint", STR,
               "Where the token is exchanged for the user's profile.", "UserEndpoint"),
            fs("ClientID", "clientID", STR, "The OAuth client ID issued by the provider.", "ClientID"),
            fs("ClientSecret", "clientSecretSecretRef", "*SecretKeySelector",
               "The OAuth client secret, read from a secret.", "ClientSecret"),
            fs("IDAttribute", "idAttribute", STR,
               "The claim in the profile that identifies the user.", "IDAttribute"),
            fs("UsePKCE", "usePKCE", BOOL,
               "Use proof key for code exchange.", "UsePKCE"),
        ],
        example=[
            ("authorizationEndpoint", '"https://example.com/oauth/authorize"'),
            ("tokenEndpoint", '"https://example.com/oauth/token"'),
            ("userEndpoint", '"https://example.com/oauth/user"'),
            ("clientID", '"my-client-id"'),
        ],
    ),
    "JWT": dict(
        kind="JWT", readable=True, org_only=True,
        purpose="authenticate a machine by a JWT it presents, rather than a person logging in",
        fields=[
            fs("JWTEndpoint", "jwtEndpoint", STR,
               "The URL Zitadel calls to turn a JWT into a profile.", "JWTEndpoint"),
            fs("KeysEndpoint", "keysEndpoint", STR,
               "Where the signing keys are published. An empty value means they are "
               "taken from the issuer instead.", "KeysEndpoint"),
            fs("Issuer", "issuer", STR,
               "The issuer the JWT must claim.", "Issuer"),
            fs("HeaderName", "headerName", STR,
               "The header the JWT arrives in, such as `Authorization`.", "HeaderName"),
            fs("Audience", "audience", STR,
               "The audience the JWT must claim, when it must be one.", "Audience"),
        ],
        example=[
            ("jwtEndpoint", '"https://example.com/jwt"'),
            ("issuer", '"https://example.com"'),
            ("headerName", '"Authorization"'),
        ],
    ),
    "Apple": dict(
        kind="Apple", readable=False,
        purpose="sign in with Apple",
        fields=[
            fs("ClientID", "clientID", STR, "The Apple service ID.", "ClientID"),
            fs("TeamID", "teamID", STR, "The Apple developer team ID.", "TeamID"),
            fs("KeyID", "keyID", STR, "The key ID of the signing key.", "KeyID"),
            fs("PrivateKey", "privateKeySecretRef", "*SecretKeySelector",
               "The PEM private key Apple signed the client with, read from a secret.", "PrivateKey"),
        ],
        example=[
            ("clientID", '"com.example.service"'),
            ("teamID", '"ABCDE12345"'),
            ("keyID", '"FGHIJ67890"'),
        ],
    ),
    "AzureAD": dict(
        kind="AzureAD", readable=False,
        purpose="sign in with Microsoft Entra ID",
        fields=[
            fs("ClientID", "clientID", STR, "The application ID of the Entra app registration.", "ClientID"),
            fs("ClientSecret", "clientSecretSecretRef", "*SecretKeySelector",
               "The client secret, read from a secret.", "ClientSecret"),
            fs("Tenant", "tenant", "*AzureADTenant",
               "Which Entra tenants may log in: `Common` for any organization, "
               "`Consumers` for personal accounts, or an exact tenant ID. "
               "Defaults to `Common`.", "Tenant"),
            fs("EmailVerified", "emailVerified", BOOL,
               "Treat the address Entra reports as verified.", "EmailVerified"),
        ],
        example=[
            ("clientID", '"00000000-0000-0000-0000-000000000000"'),
            ("tenant", '"Common"'),
        ],
    ),
    "GitHub": dict(
        kind="GitHub", readable=False,
        purpose="sign in with GitHub",
        fields=[
            fs("ClientID", "clientID", STR, "The OAuth app's client ID.", "ClientID"),
            fs("ClientSecret", "clientSecretSecretRef", "*SecretKeySelector",
               "The OAuth app's client secret, read from a secret.", "ClientSecret"),
        ],
        example=[("clientID", '"Iv1.0000000000000000"')],
    ),
    "GitHubEnterpriseServer": dict(
        kind="GitHubEnterpriseServer", readable=False,
        purpose="sign in with a self hosted GitHub Enterprise Server",
        fields=[
            fs("ClientID", "clientID", STR, "The OAuth app's client ID.", "ClientID"),
            fs("ClientSecret", "clientSecretSecretRef", "*SecretKeySelector",
               "The OAuth app's client secret, read from a secret.", "ClientSecret"),
            fs("AuthorizationEndpoint", "authorizationEndpoint", STR,
               "Where the browser is sent to log in.", "AuthorizationEndpoint"),
            fs("TokenEndpoint", "tokenEndpoint", STR,
               "Where the code is exchanged for a token.", "TokenEndpoint"),
            fs("UserEndpoint", "userEndpoint", STR,
               "Where the token is exchanged for the user's profile.", "UserEndpoint"),
        ],
        example=[
            ("clientID", '"my-client-id"'),
            ("authorizationEndpoint", '"https://github.example.com/login/oauth/authorize"'),
            ("tokenEndpoint", '"https://github.example.com/login/oauth/access_token"'),
            ("userEndpoint", '"https://github.example.com/api/v3/user"'),
        ],
    ),
    "GitLab": dict(
        kind="GitLab", readable=False,
        purpose="sign in with gitlab.com",
        fields=[
            fs("ClientID", "clientID", STR, "The OAuth app's client ID.", "ClientID"),
            fs("ClientSecret", "clientSecretSecretRef", "*SecretKeySelector",
               "The OAuth app's client secret, read from a secret.", "ClientSecret"),
        ],
        example=[("clientID", '"my-client-id"')],
    ),
    "GitLabSelfHosted": dict(
        kind="GitLabSelfHosted", readable=False,
        purpose="sign in with a self hosted GitLab",
        fields=[
            fs("Issuer", "issuer", STR, "The URL of the GitLab instance.", "Issuer"),
            fs("ClientID", "clientID", STR, "The OAuth app's client ID.", "ClientID"),
            fs("ClientSecret", "clientSecretSecretRef", "*SecretKeySelector",
               "The OAuth app's client secret, read from a secret.", "ClientSecret"),
        ],
        example=[
            ("issuer", '"https://gitlab.example.com"'),
            ("clientID", '"my-client-id"'),
        ],
    ),
    "Google": dict(
        kind="Google", readable=False,
        purpose="sign in with Google",
        fields=[
            fs("ClientID", "clientID", STR, "The OAuth app's client ID.", "ClientID"),
            fs("ClientSecret", "clientSecretSecretRef", "*SecretKeySelector",
               "The OAuth app's client secret, read from a secret.", "ClientSecret"),
        ],
        example=[("clientID", '"000000000000.apps.googleusercontent.com"')],
    ),
    "LDAP": dict(
        kind="LDAP", readable=False,
        purpose="sign in against a directory server",
        fields=[
            fs("Servers", "servers", STRS, "The server URLs, such as `ldap://ldap.example.com:389`.", "Servers"),
            fs("StartTLS", "startTLS", BOOL, "Upgrade the connection with StartTLS.", "StartTLS"),
            fs("BaseDN", "baseDN", STR, "The base of the tree to search, such as `ou=people,dc=example,dc=com`.", "BaseDN"),
            fs("BindDN", "bindDN", STR, "The account Zitadel binds as to search, such as `cn=admin,dc=example,dc=com`. Zitadel requires it, together with its password.", "BindDN"),
            fs("BindPassword", "bindPasswordSecretRef", "*SecretKeySelector",
               "The password for the bind account, read from a secret.", "BindPassword"),
            fs("UserBase", "userBase", STR, "The subtree to search under `baseDN`.", "UserBase"),
            fs("UserObjectClasses", "userObjectClasses", STRS,
               "The object classes a user entry has, such as `inetOrgPerson`.", "UserObjectClasses"),
            fs("UserFilters", "userFilters", STRS,
               "Filters applied to the search, such as `(objectClass=person)`. Zitadel requires at least one.", "UserFilters"),
            fs("IDAttribute", "idAttribute", STR,
               "The attribute that identifies the user.", "IDAttribute"),
            fs("RootCA", "rootCASecretRef", "*SecretKeySelector",
               "The CA certificate that signs the server's, read from a secret.", "RootCA"),
        ],
        example=[
            ("servers", '["ldap://ldap.example.com:389"]'),
            ("baseDN", '"ou=people,dc=example,dc=com"'),
            ("userBase", '"ou=people"'),
            ("userObjectClasses", '["inetOrgPerson"]'),
            ("userFilters", '["(objectClass=inetOrgPerson)"]'),
        ],
    ),
    "SAML": dict(
        kind="SAML", readable=False,
        purpose="sign in with a SAML 2.0 identity provider",
        fields=[
            fs("Metadata", "metadata", STR,
               "The identity provider's metadata: either the XML document itself, or a "
               "URL Zitadel fetches it from. A URL is only worth using when Zitadel can "
               "reach it; a document always works.", "Metadata"),
            fs("Binding", "binding", "*SAMLBinding",
               "How the authentication request is sent: `Redirect` or `POST`. "
               "Defaults to `Redirect`.", "Binding"),
            fs("WithSignedRequest", "withSignedRequest", BOOL,
               "Sign the authentication request, which the identity provider "
               "requires when it will not accept an unsigned one.", "WithSignedRequest"),
            fs("NameIDFormat", "nameIDFormat", "*SAMLNameIDFormat",
               "The name identifier format the provider uses: `EmailAddress`, "
               "`Persistent` or `Transient`.", "NameIDFormat"),
            fs("SignatureAlgorithm", "signatureAlgorithm", "*SAMLSignatureAlgorithm",
               "How assertions are signed: `RSA-SHA1`, `RSA-SHA256` or `RSA-SHA512`.", "SignatureAlgorithm"),
            fs("FederatedLogoutEnabled", "federatedLogoutEnabled", BOOL,
               "Support single logout, so signing out propagates to the provider.", "FederatedLogoutEnabled"),
        ],
        example=[
            # The document itself, so the example works without Zitadel being able
            # to reach an identity provider first.
            ("metadata", "|" + "\n      " + """<?xml version="1.0"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata"
                  entityID="https://idp.example.com"/>"""),
            ("binding", '"POST"'),
            ("nameIDFormat", '"EmailAddress"'),
        ],
    ),
}

ORG_REFERENCE = [
    f("OrganizationID", "organizationID", STR,
      "The ID of the organization the provider belongs to. Either `organizationID`, "
      "`organizationRef` or `organizationSelector` must be set."),
    f("OrganizationRef", "organizationRef", "*xpv1.Reference",
      "OrganizationRef references an Organization managed by this provider and uses its ID."),
    f("OrganizationSelector", "organizationSelector", "*xpv1.Selector",
      "OrganizationSelector selects an Organization managed by this provider and uses its ID."),
]


# The file name each kind's example gets, following the Terraform provider's so
# the two are recognisably the same set.
EXAMPLE_FILE = {
    "IDPOIDC": "idp_oidc", "IDPOAuth": "idp_oauth", "IDPApple": "idp_apple",
    "IDPAzureAD": "idp_azure_ad", "IDPGitHub": "idp_github",
    "IDPGitHubEnterpriseServer": "idp_github_es", "IDPGitLab": "idp_gitlab",
    "IDPGitLabSelfHosted": "idp_gitlab_self_hosted", "IDPGoogle": "idp_google",
    "IDPLDAP": "idp_ldap", "IDPSAML": "idp_saml",
    "OrgIDPOIDC": "org_idp_oidc", "OrgIDPOAuth": "org_idp_oauth", "OrgIDPJWT": "org_idp_jwt",
    "OrgIDPApple": "org_idp_apple", "OrgIDPAzureAD": "org_idp_azure_ad",
    "OrgIDPGitHub": "org_idp_github", "OrgIDPGitHubEnterpriseServer": "org_idp_github_es",
    "OrgIDPGitLab": "org_idp_gitlab", "OrgIDPGitLabSelfHosted": "org_idp_gitlab_self_hosted",
    "OrgIDPGoogle": "org_idp_google", "OrgIDPLDAP": "org_idp_ldap", "OrgIDPSAML": "org_idp_saml",
}


def snake(name):
    import re
    parts = re.findall(r"[A-Z]+(?![a-z])|[A-Z][a-z0-9]*|[a-z0-9]+", name)
    return "_".join(p.lower() for p in parts)


def kind_name(provider, org):
    """The kind of one identity provider.

    Zitadel and the Terraform provider both distinguish the two levels by prefix,
    so the names here follow suit: an instance wide one has no prefix, an
    organization's has `Org`.
    """
    base = "IDP" + provider
    return ("Org" + base) if org else base


def spec_fields(spec, org):
    fields = (ORG_REFERENCE if org else []) + spec["fields"] + COMMON
    # The organization reference is the same three fields in every kind; the
    # provider's own fields come first so the example reads naturally.
    if org:
        fields = spec["fields"] + ORG_REFERENCE + COMMON
    return fields


# Every identity provider reports the same things, so all twenty three kinds
# share one observation type. Each kind declares its own only so that the status
# it writes is named after it, which is what makes `status.atProvider` readable
# in a manifest and in the CRD.
SHARED_OBSERVATION = "IDPObservation"


def render_types(kind, spec, org):
    fields = spec_fields(spec, org)

    out = [LICENSE, "", "package v1alpha1", "", IMPORTS, ""]
    out.append("// %sParameters are the configurable fields of an %s." % (kind, kind))
    out.append("type %sParameters struct {" % kind)
    for f_ in fields:
        out.append(wrap(f_["doc"]))
        out.append("\t// +optional")
        out.append("\t%s %s `json:\"%s,omitempty\"`" % (f_["name"], f_["type"], f_["json"]))
        out.append("")
    out.append("}")
    out.append("")
    out.append("//")
    out.append("// Every identity provider reports the same things, and reports a")
    out.append("// configuration only for an OIDC or a JWT one, so all twenty three kinds")
    out.append("// share one observation type rather than each declaring an identical copy.")
    out.append("type %sObservation = IDPObservation" % kind)
    out.append("")

    out.append("// An %s lets users %s." % (kind, spec["purpose"]))
    out.append("//")
    if org:
        out.append(wrap("It belongs to one organization. Deleting it removes the provider "
                        "from that organization; Zitadel refuses to remove one that is "
                        "still offered on a login page, so it has to be unbound from "
                        "the login policy first."))
    else:
        out.append(wrap("It is available to every organization that has not set up its "
                        "own. Deleting it removes it from the instance."))
    out.append("//")
    if not spec["readable"]:
        out.append(wrap("Zitadel does not report this kind's settings back, so they are "
                        "applied once at creation and are not drift detected: changing "
                        "one means deleting the provider and making it again."))
        out.append("//")
    out.append("// +kubebuilder:object:root=true")
    out.append("// +kubebuilder:subresource:status")
    out.append("// +kubebuilder:storageversion")
    if org:
        out.append("// +kubebuilder:printcolumn:name=\"Organization\",type=\"string\",JSONPath=\".status.atProvider.organizationID\"")
    out.append("// +kubebuilder:printcolumn:name=\"State\",type=\"string\",JSONPath=\".status.atProvider.state\"")
    out.append("// +kubebuilder:printcolumn:name=\"Age\",type=\"date\",JSONPath=\".metadata.creationTimestamp\"")
    out.append("// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}")
    out.append("type %s struct {" % kind)
    out.append("\tmetav1.TypeMeta   `json:\",inline\"`")
    out.append("\tmetav1.ObjectMeta `json:\"metadata,omitempty\"`")
    out.append("")
    out.append("\tSpec   %sSpec   `json:\"spec\"`" % kind)
    out.append("\tStatus %sStatus `json:\"status,omitempty\"`" % kind)
    out.append("}")
    out.append("")
    out.append("// %sSpec defines the desired state of an %s." % (kind, kind))
    out.append("type %sSpec struct {" % kind)
    out.append("\txpv2.ManagedResourceSpec `json:\",inline\"`")
    out.append("\tForProvider              %sParameters `json:\"forProvider\"`" % kind)
    out.append("}")
    out.append("")
    out.append("// %sStatus is the observed state of an %s." % (kind, kind))
    out.append("type %sStatus struct {" % kind)
    out.append("\txpv1.ResourceStatus `json:\",inline\"`")
    out.append("\tAtProvider               %sObservation `json:\"atProvider,omitempty\"`" % kind)
    out.append("}")
    out.append("")
    out.append("// %sList contains a list of %s." % (kind, kind))
    out.append("//")
    out.append("// +kubebuilder:object:root=true")
    out.append("type %sList struct {" % kind)
    out.append("\tmetav1.TypeMeta `json:\",inline\"`")
    out.append("\tmetav1.ListMeta `json:\"metadata,omitempty\"`")
    out.append("\tItems           []%s `json:\"items\"`" % kind)
    out.append("}")
    out.append("")
    out.append("// %s type metadata." % kind)
    out.append("var (")
    out.append("\t%sKind             = reflect.TypeOf(%s{}).Name()" % (kind, kind))
    out.append("\t%sGroupKind        = schema.GroupKind{Group: Group, Kind: %sKind}.String()" % (kind, kind))
    out.append("\t%sKindAPIVersion   = %sKind + \".\" + SchemeGroupVersion.String()" % (kind, kind))
    out.append("\t%sGroupVersionKind = SchemeGroupVersion.WithKind(%sKind)" % (kind, kind))
    out.append(")")
    out.append("")
    out.append("func init() {")
    out.append("\tSchemeBuilder.Register(&%s{}, &%sList{})" % (kind, kind))
    out.append("}")
    out.append("")
    # The four settings every provider carries are read through one interface,
    # because there are twenty three kinds and they are identical in all of them.
    for go, typ in (("IsLinkingAllowed", "bool"), ("IsCreationAllowed", "bool"),
                    ("IsAutoCreation", "bool"), ("IsAutoUpdate", "bool"),
                    ("AutoLinking", "IDPAutoLinking"), ("State", "IDPState")):
        out.append("// Get%s returns the setting every identity provider carries." % go)
        out.append("func (mg *%s) Get%s() *%s { return mg.Spec.ForProvider.%s }" % (kind, go, typ, go))
        out.append("")

    out.append("// SetIDPOrganization records the organization the provider belongs to, or the")
    out.append("// empty string for an instance wide one.")
    out.append("func (mg *%s) SetIDPOrganization(orgID string) { mg.Status.AtProvider.Scope = StringPtr(orgID) }" % kind)
    out.append("")
    out.append("// IDPOrganization returns the organization the provider was last read from.")
    out.append("func (mg *%s) IDPOrganization() string { return Deref(mg.Status.AtProvider.Scope) }" % kind)
    out.append("")

    if org:
        out.append("// GetOrganizationRef returns the reference the provider's organization is")
        out.append("// resolved from.")
        out.append("func (mg *%s) GetOrganizationRef() *xpv1.Reference { return mg.Spec.ForProvider.OrganizationRef }" % kind)
        out.append("")
        out.append("// GetOrganizationSelector returns the selector the provider's organization")
        out.append("// is resolved from.")
        out.append("func (mg *%s) GetOrganizationSelector() *xpv1.Selector { return mg.Spec.ForProvider.OrganizationSelector }" % kind)
        out.append("")
        out.append("// GetOrganizationID returns the pinned organization ID, if any.")
        out.append("func (mg *%s) GetOrganizationID() *string { return mg.Spec.ForProvider.OrganizationID }" % kind)
        out.append("")
        out.append("// GetObservedOrganization returns the organization the provider was last read")
        out.append("// from, which is what a terminating object acts on.")
        out.append("func (mg *%s) GetObservedOrganization() *string { return mg.Status.AtProvider.OrganizationID }" % kind)
        out.append("")

    return "\n".join(out)


# The resource name each example uses, by provider type.
EXAMPLE_NAME = {
    "OIDC": "example-oidc", "OAuth": "example-oauth", "JWT": "example-jwt",
    "Apple": "example-apple", "AzureAD": "example-entra", "GitHub": "example-github",
    "GitHubEnterpriseServer": "example-github-enterprise",
    "GitLab": "example-gitlab", "GitLabSelfHosted": "example-gitlab-self-hosted",
    "Google": "example-google", "LDAP": "example-ldap", "SAML": "example-saml",
}


# The resource name each example uses, by provider type.
EXAMPLE_NAME = {
    "OIDC": "example-oidc", "OAuth": "example-oauth", "JWT": "example-jwt",
    "Apple": "example-apple", "AzureAD": "example-entra", "GitHub": "example-github",
    "GitHubEnterpriseServer": "example-github-enterprise",
    "GitLab": "example-gitlab", "GitLabSelfHosted": "example-gitlab-self-hosted",
    "Google": "example-google", "LDAP": "example-ldap", "SAML": "example-saml",
}


def render_example(kind, spec, org):
    name = EXAMPLE_NAME[spec["kind"]]

    out = ["# An identity provider that lets users %s." % spec["purpose"], "#"]

    if org:
        out += ["# It belongs to one organization. Zitadel refuses to remove a provider that is",
                "# still offered on a login page, so it has to be unbound from the login policy",
                "# first."]
    else:
        out.append("# It is available to every organization that has not set up its own.")

    if not spec["readable"]:
        out.append("#")
        out += ["# Zitadel does not report this kind's settings back, so they are applied once at",
                "# creation and are not drift detected: changing one means deleting the provider",
                "# and making it again. Only the name, the state and whether it registers users",
                "# automatically are compared."]

    out.append("#")
    out.append("apiVersion: zitadel.m.crossplane.io/v1alpha1")
    out.append("kind: %s" % kind)
    out.append("metadata:")
    out.append("  name: %s" % name)
    out.append("  namespace: zitadel")
    out.append("spec:")
    out.append("  providerConfigRef: {name: zitadel, kind: ProviderConfig}")
    out.append("  forProvider:")

    for json_name, value in spec["example"]:
        out.append("    %s: %s" % (json_name, value))

    for f in spec["fields"]:
        if f["type"] == "*SecretKeySelector":
            out.append("    %s: {name: %s-credentials, key: %s}"
                       % (f["json"], name, f["name"].lower()))

    if org:
        out.append("    organizationRef: {name: platform}")

    out.append("    state: Active")

    # A provider with credentials says how to make the secret they come from,
    # because a manifest that only names a secret is not enough to get going.
    if any(f["type"] == "*SecretKeySelector" for f in spec["fields"]):
        out.append("#")
        out.append("# A credential is read from a secret rather than written here, and Zitadel")
        out.append("# never returns one either, so it is applied but never compared. Make the")
        out.append("# secret with a command like the one below, then point the references above")
        out.append("# at it.")
        out.append("#")
        out.append("#   kubectl -n zitadel create secret generic %s-credentials \\"
                   % name)
        for f in spec["fields"]:
            if f["type"] == "*SecretKeySelector":
                out.append("#     --from-literal=%s=... \\" % f["name"].lower())

    out.append("")
    return "\n".join(out)

if __name__ == "__main__":
    types_dir = sys.argv[1]
    examples_dir = sys.argv[2] if len(sys.argv) > 2 else None

    written_types = []
    for provider, spec in sorted(TYPES.items()):
        scopes = [True] if spec.get("org_only") else [False, True]
        for org in scopes:
            kind = kind_name(provider, org)
            path = os.path.join(types_dir, snake(kind) + "_types.go")
            with open(path, "w") as fh:
                fh.write(render_types(kind, spec, org))

            if examples_dir:
                ex = os.path.join(examples_dir, EXAMPLE_FILE[kind] + ".yaml")
                with open(ex, "w") as fh:
                    fh.write(render_example(kind, spec, org))

            written_types.append((kind, spec, org))

    print("%d identity provider types written" % len(written_types))
    for kind, spec, org in written_types:
        print("  %-34s %s" % (kind, "org" if org else "instance"))

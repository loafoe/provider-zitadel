#!/usr/bin/env python3
"""Generate the SAML application, active web key and messaging provider types.

These are the last six resources: a SAML application beside the API and OIDC
ones, a pointer at the signing key Zitadel signs with, and the four providers
Zitadel sends its codes and notifications through.

The four messaging providers are the same thing four times over - a provider
with an identity, a state and a description - so they are generated from one
table rather than written out four times.
"""

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

STR = "*string"
BOOL = "*bool"
ORG = '''	// OrganizationID is the ID of the organization the resource belongs to.
	// Either `organizationID`, `organizationRef` or `organizationSelector` must
	// be set.
	//
	// +optional
	OrganizationID *string `json:"organizationID,omitempty"`

	// OrganizationRef references an Organization managed by this provider and
	// uses its ID.
	//
	// +optional
	OrganizationRef *xpv1.Reference `json:"organizationRef,omitempty"`

	// OrganizationSelector selects an Organization managed by this provider and
	// uses its ID.
	//
	// +optional
	OrganizationSelector *xpv1.Selector `json:"organizationSelector,omitempty"`
'''

PROJECT = ORG + '''
	// ProjectID is the ID of the project the resource belongs to. Either
	// `projectID`, `projectRef` or `projectSelector` must be set.
	//
	// +optional
	ProjectID *string `json:"projectID,omitempty"`

	// ProjectRef references a Project managed by this provider and uses its ID.
	//
	// +optional
	ProjectRef *xpv1.Reference `json:"projectRef,omitempty"`

	// ProjectSelector selects a Project managed by this provider and uses its ID.
	//
	// +optional
	ProjectSelector *xpv1.Selector `json:"projectSelector,omitempty"`
'''

COMMON = ['''
// MessageProviderState is whether Zitadel sends through a provider. A provider
// that is inactive is configured but not offered to anyone.
type MessageProviderState string

const (
	// MessageProviderStateActive means Zitadel sends through this provider.
	MessageProviderStateActive MessageProviderState = "Active"

	// MessageProviderStateInactive means the provider is configured but not
	// used. Zitadel refuses to deactivate the last one it has, so an instance
	// needs another before this can be set.
	MessageProviderStateInactive MessageProviderState = "Inactive"
)
''']

# The four messaging providers. They differ in their fields and in which calls
# they make, and agree on everything else, so they share one shape.
MESSAGING = [
    {
        "kind": "EmailProviderSMTP",
        "title": "Zitadel SMTP email provider",
        "purpose": "have Zitadel send email through an SMTP server",
        "fields": [
            ("Host", "host", STR,
             "The SMTP server, as `host:port`, such as `smtp.example.com:587`. "
             "Zitadel does not connect to it when the provider is added, only "
             "when something is sent, so it can be set up ahead of a server that "
             "is not reachable yet."),
            ("User", "user", STR, "The account Zitadel authenticates to the server as."),
            ("PasswordSecretRef", "passwordSecretRef", "*SecretKeySelector",
             "That account's password. Zitadel never returns it, so it is applied "
             "and then left alone: an update that carries none leaves the stored "
             "password as it is. It is a secret rather than a field so that it is "
             "not readable by anyone who can read this object."),
            ("SenderAddress", "senderAddress", STR,
             "The address Zitadel sends from, such as `noreply@example.com`. It "
             "has to be a domain the organization owns."),
            ("SenderName", "senderName", STR,
             "The name shown as the sender of an email Zitadel sends."),
            ("ReplyToAddress", "replyToAddress", STR,
             "Where a reply should go, when that is not the sender address."),
            ("TLS", "tls", BOOL,
             "Connect over TLS. Zitadel calls this `tls` rather than `starttls`: "
             "it means the connection is wrapped from the start, which is what "
             "port 465 expects."),
            ("Description", "description", STR,
             "What this provider is for, shown in the Zitadel console."),
        ],
        "required": ["Host"],
        "example": [
            ("host", '"smtp.example.com:587"'),
            ("user", '"zitadel@example.com"'),
            ("passwordSecretRef", "{name: zitadel-smtp, key: password}"),
            ("senderAddress", '"noreply@example.com"'),
            ("senderName", '"Example"'),
            ("tls", "true"),
            ("description", '"The instance wide email provider"'),
        ],
    },
    {
        "kind": "EmailProviderHTTP",
        "title": "Zitadel HTTP email provider",
        "purpose": "have Zitadel send email by posting it to an endpoint",
        "fields": [
            ("Endpoint", "endpoint", STR,
             "Where Zitadel posts the message. It receives the same payload Zitadel "
             "would hand an SMTP server, so the endpoint has to understand it."),
            ("Description", "description", STR,
             "What this provider is for, shown in the Zitadel console."),
        ],
        "required": ["Endpoint"],
        "example": [
            ("endpoint", '"https://email.example.com/send"'),
            ("description", '"The instance wide email provider"'),
        ],
    },
    {
        "kind": "SMSProviderTwilio",
        "title": "Zitadel Twilio SMS provider",
        "purpose": "have Zitadel send SMS through Twilio",
        "fields": [
            ("SID", "sid", STR, "The Twilio account SID."),
            ("TokenSecretRef", "tokenSecretRef", "*SecretKeySelector",
             "The Twilio auth token. Zitadel never returns it and sets it through a "
             "call of its own, so it is applied and then left alone. It is a secret "
             "rather than a field so that it is not readable by anyone who can read "
             "this object."),
            ("SenderNumber", "senderNumber", STR,
             "The number messages are sent from, in E.164 form such as "
             "`+441234567890`."),
            ("VerifyServiceSID", "verifyServiceSID", STR,
             "The SID of a Twilio Verify service, when messages go through one "
             "rather than through the account directly."),
            ("Description", "description", STR,
             "What this provider is for, shown in the Zitadel console."),
        ],
        "required": ["SID"],
        "example": [
            ("sid", '"ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"'),
            ("tokenSecretRef", "{name: zitadel-twilio, key: token}"),
            ("senderNumber", '"+441234567890"'),
            ("description", '"The instance wide SMS provider"'),
        ],
    },
    {
        "kind": "SMSProviderHTTP",
        "title": "Zitadel HTTP SMS provider",
        "purpose": "have Zitadel send SMS by posting it to an endpoint",
        "fields": [
            ("Endpoint", "endpoint", STR,
             "Where Zitadel posts the message. It receives the same payload Zitadel "
             "would hand an SMS gateway, so the endpoint has to understand it."),
            ("Description", "description", STR,
             "What this provider is for, shown in the Zitadel console."),
        ],
        "required": ["Endpoint"],
        "example": [
            ("endpoint", '"https://sms.example.com/send"'),
            ("description", '"The instance wide SMS provider"'),
        ],
    },
]

SAML = {
    "kind": "ApplicationSAML",
    "title": "Zitadel SAML application",
    "purpose": "let an application sign users in through SAML",
    "fields": PROJECT,
    "params": [
        ("Name", "name", STR, "The application's name, shown on the login page."),
        ("MetadataXML", "metadata", STR,
         "The identity provider's metadata document itself. This is the form that "
         "always works, because Zitadel does not have to fetch anything."),
        ("MetadataURL", "metadataURL", STR,
         "Where Zitadel fetches the metadata from. Only worth using when Zitadel "
         "can reach that URL: a document given as `metadata` always works."),
        ("LoginVersion", "loginVersion", STR,
         "Which login screen the user is sent to: `LoginV1` or `LoginV2`. Empty "
         "leaves the instance default."),
    ],
    "observation": [
        ("ID", "string"),
        ("Name", "string"),
        ("ProjectID", "string"),
        ("State", "string"),
        ("MetadataXML", "string"),
        ("MetadataURL", "string"),
        ("LoginVersion", "string"),
    ],
    "example": [
        ("name", '"Example SAML"'),
        ("metadata", '|\n      ' + '<?xml version="1.0"?>\n<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata"\n                  entityID="https://idp.example.com"/>'),
        ("loginVersion", '"LoginV1"'),
    ],
    "doc": (
        "// A SAML application is made and read through the same application API as\n"
        "// the API and OIDC ones: Zitadel has one application service with three\n"
        "// types in it, not one service per type. The older SAML-only endpoints still\n"
        "// exist and are not used.\n"
        "//\n"
        "// Zitadel requires one form of the metadata or the other, so exactly one of\n"
        "// `metadata` and `metadataURL` must be set.\n"
    ),
}

ACTIVE_WEBKEY = {
    "kind": "ActiveWebKey",
    "title": "Zitadel active signing key",
    "purpose": "point at the signing key Zitadel signs tokens with",
    "params": [
        ("WebKeyID", "webKeyID", STR,
         "The signing key to activate, as an ID or a reference to a WebKey managed "
         "by this provider."),
        ("WebKeyRef", "webKeyRef", "xpv1.Reference",
         "References a WebKey managed by this provider and uses its key."),
        ("WebKeySelector", "webKeySelector", "xpv1.Selector",
         "Selects a WebKey managed by this provider and uses its key."),
    ],
    "observation": [
        ("ID", "string"),
        ("Algorithm", "string"),
        ("State", "string"),
    ],
    "example": [("webKeyRef", "{name: zitadel-web-key}")],
    "doc": (
        "// Zitadel signs its tokens with one of its keys, and this is what decides\n"
        "// which. There is only ever one active key: activating another makes this\n"
        "// stale and the former one inactive, so two of these in different namespaces\n"
        "// would be fighting over the same setting.\n"
        "//\n"
        "// Zitadel will not remove a key that is active, so a key this resource\n"
        "// points at has to be activated away before it can be deleted.\n"
    ),
}


def go_json_name(name):
    """The JSON name of a Go field.

    A leading run of capitals is one initialism, so it lowercases together:
    `ID` is `id` and `SID` is `sid`. A run that ends a longer name keeps its last
    capital, because that capital starts the next word: `ProjectID` is
    `projectID` and `MetadataXML` is `metadataXML`.
    """
    n = 0
    while n < len(name) and name[n].isupper():
        n += 1

    if n == 0:
        return name[0].lower() + name[1:]

    # The whole name is one initialism, so it lowercases together.
    if n == len(name):
        return name.lower()

    # A single capital is just the first letter of a word, so the rest is left
    # alone: ProjectID is projectID and not projectid.
    if n == 1:
        return name[0].lower() + name[1:]

    # A longer run keeps its last capital, because that capital starts the next
    # word: MetadataXML is metadataXML and not metadataxmL.
    return name[: n - 1].lower() + name[n - 1:]


def render_simple(spec):
    kind = spec["kind"]
    out = [LICENSE, ""]
    out.append("package v1alpha1\n")
    out.append("import (")
    out.append('\t"reflect"\n')
    out.append('\txpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"')
    out.append('\txpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"')
    out.append('\tmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"')
    out.append('\t"k8s.io/apimachinery/pkg/runtime/schema"\n)\n')

    out.append("// %sSpec defines the desired state of a %s." % (kind, kind))
    out.append("type %sSpec struct {" % kind)
    out.append('\txpv2.ManagedResourceSpec `json:",inline"`')
    out.append('\tForProvider %sParameters `json:"forProvider"`' % kind)
    out.append("}\n")

    out.append("// %sParameters is the desired configuration of a %s." % (kind, spec["title"]))
    out.append("type %sParameters struct {" % kind)
    if spec.get("fields") == PROJECT:
        out.append(spec["fields"])
    for name, json_name, go_type, doc in spec["params"]:
        out.append("\t// %s controls the following. %s" % (name, doc.rstrip(".")))
        out.append("\t//")
        out.append("\t// +optional")
        out.append('\t%s %s `json:"%s,omitempty"`' % (name, go_type, json_name))
    out.append("}\n")

    out.append("// %sObservation is what Zitadel reports about a %s." % (kind, spec["title"]))
    out.append("type %sObservation struct {" % kind)
    for name, go_type in spec["observation"]:
        out.append('\t%s %s `json:"%s,omitempty"`' % (name, go_type, go_json_name(name)))
    out.append("}\n")

    out.append("// %sStatus reports the observed state of a %s." % (kind, spec["title"]))
    out.append("type %sStatus struct {" % kind)
    out.append('\txpv1.ResourceStatus `json:",inline"`\n')
    out.append("\t// AtProvider is the observed state.")
    out.append('\tAtProvider %sObservation `json:"atProvider,omitempty"`' % kind)
    out.append("")
    out.append("\t// WebKeyID is the key that was last found to be active, recorded so that a")
    out.append("\t// terminating resource can report on it without following a reference that")
    out.append("\t// may since have moved.")
    out.append('\tWebKeyID string `json:"webKeyID,omitempty"`')
    out.append("}\n")

    out.append("// +kubebuilder:object:root=true")
    out.append("// +kubebuilder:subresource:status")
    out.append("// +kubebuilder:resource:scope=Namespaced,categories={zitadel}")
    out.append("// +kubebuilder:printcolumn:name=\"READY\",type=\"string\",JSONPath=\".status.conditions[?(@.type=='Ready')].status\"")
    out.append("// +kubebuilder:printcolumn:name=\"SYNCED\",type=\"string\",JSONPath=\".status.conditions[?(@.type=='Synced')].status\"")
    out.append("// +kubebuilder:printcolumn:name=\"AGE\",type=\"date\",JSONPath=\".metadata.creationTimestamp\"")
    out.append("")
    out.append("// %s is a managed resource that %s." % (kind, spec["purpose"]))
    out.append("//")
    out.append(spec["doc"].rstrip("\n"))
    out.append("// +kubebuilder:object:generate=true")
    out.append("type %s struct {" % kind)
    out.append('\tmetav1.TypeMeta   `json:",inline"`')
    out.append('\tmetav1.ObjectMeta `json:"metadata,omitempty"`\n')
    out.append('\tSpec   %sSpec   `json:"spec"`' % kind)
    out.append('\tStatus %sStatus `json:"status,omitempty"`' % kind)
    out.append("}\n")

    out.append("// +kubebuilder:object:root=true")
    out.append("")
    out.append("// %sList contains a list of %s." % (kind, kind))
    out.append("type %sList struct {" % kind)
    out.append('\tmetav1.TypeMeta `json:",inline"`')
    out.append('\tmetav1.ListMeta `json:"metadata,omitempty"`')
    out.append('\tItems           []%s `json:"items"`' % kind)
    out.append("}\n")

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
    return "\n".join(out)


def render_messaging(spec, first=False):
    """A messaging provider: one table, four shapes."""
    kind = spec["kind"]
    out = [LICENSE, ""]
    out.append("package v1alpha1\n")
    out.append("import (")
    out.append('\t"reflect"\n')
    out.append('\txpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"')
    out.append('\txpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"')
    out.append('\tmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"')
    out.append('\t"k8s.io/apimachinery/pkg/runtime/schema"\n)\n')

    if first:
        # The state is shared by all four providers, so it is written once with
        # the first of them rather than four times over.
        out.append(COMMON[0].strip("\n"))
        out.append("")

    out.append("// %sSpec defines the desired state of a %s." % (kind, kind))
    out.append("type %sSpec struct {" % kind)
    out.append('\txpv2.ManagedResourceSpec `json:",inline"`')
    out.append('\tForProvider %sParameters `json:"forProvider"`' % kind)
    out.append("}\n")

    out.append("// %sParameters is the desired configuration of a %s." % (kind, spec["title"]))
    out.append("type %sParameters struct {" % kind)
    for name, json_name, go_type, doc in spec["fields"]:
        out.append("\t// %s controls the following. %s" % (name, doc.rstrip(".")))
        out.append("\t//")
        if name in spec.get("required", []):
            out.append("\t// It is required: Zitadel cannot do without it.")
        else:
            out.append("\t//")
            out.append("\t// +optional")
        out.append('\t%s %s `json:"%s,omitempty"`' % (name, go_type, json_name))
    out.append("")
    out.append("\t// State is whether Zitadel sends through this provider. Empty leaves")
    out.append("\t// Zitadel's own choice, which is to send through it.")
    out.append("\t//")
    out.append("\t// +optional")
    out.append("\tState *MessageProviderState `json:\"state,omitempty\"`")
    out.append("}\n")

    out.append("// %sObservation is what Zitadel reports about a %s." % (kind, spec["title"]))
    out.append("type %sObservation struct {" % kind)
    out.append("\tID    string               `json:\"id,omitempty\"`")
    out.append("\tState string               `json:\"state,omitempty\"`")
    out.append("\tKind  string               `json:\"kind,omitempty\"`")
    for name, json_name, _, _ in spec["fields"]:
        # A credential is never reported: Zitadel does not return it, and a
        # status is read by anyone who can read the object.
        if "SecretKeySelector" in dict((n, t) for n, _, t, _ in spec["fields"])[name]:
            continue

        gotype = "bool" if json_name == "tls" else "string"
        out.append('\t%s %s `json:"%s,omitempty"`' % (name, gotype, json_name))
    out.append("}\n")

    out.append("// %sStatus reports the observed state of a %s." % (kind, spec["title"]))
    out.append("type %sStatus struct {" % kind)
    out.append('\txpv1.ResourceStatus `json:",inline"`\n')
    out.append("\t// AtProvider is the observed state.")
    out.append('\tAtProvider %sObservation `json:"atProvider,omitempty"`' % kind)
    out.append("}\n")

    out.append("// +kubebuilder:object:root=true")
    out.append("// +kubebuilder:subresource:status")
    out.append("// +kubebuilder:resource:scope=Namespaced,categories={zitadel}")
    out.append("// +kubebuilder:printcolumn:name=\"READY\",type=\"string\",JSONPath=\".status.conditions[?(@.type=='Ready')].status\"")
    out.append("// +kubebuilder:printcolumn:name=\"SYNCED\",type=\"string\",JSONPath=\".status.conditions[?(@.type=='Synced')].status\"")
    out.append("// +kubebuilder:printcolumn:name=\"STATE\",type=\"string\",JSONPath=\".status.atProvider.state\"")
    out.append("// +kubebuilder:printcolumn:name=\"AGE\",type=\"date\",JSONPath=\".metadata.creationTimestamp\"")
    out.append("")
    out.append("// %s is a managed resource that %s." % (kind, spec["purpose"]))
    out.append("//")
    out.append("// This is a namespaced object managing state that belongs to the whole")
    out.append("// Zitadel instance. Zitadel sends through one provider of each kind, so")
    out.append("// two of these of the same kind in different namespaces would be asking")
    out.append("// for the same thing and the second would win.")
    out.append("// +kubebuilder:object:generate=true")
    out.append("type %s struct {" % kind)
    out.append('\tmetav1.TypeMeta   `json:",inline"`')
    out.append('\tmetav1.ObjectMeta `json:"metadata,omitempty"`\n')
    out.append('\tSpec   %sSpec   `json:"spec"`' % kind)
    out.append('\tStatus %sStatus `json:"status,omitempty"`' % kind)
    out.append("}\n")

    out.append("// +kubebuilder:object:root=true")
    out.append("")
    out.append("// %sList contains a list of %s." % (kind, kind))
    out.append("type %sList struct {" % kind)
    out.append('\tmetav1.TypeMeta `json:",inline"`')
    out.append('\tmetav1.ListMeta `json:"metadata,omitempty"`')
    out.append('\tItems           []%s `json:"items"`' % kind)
    out.append("}\n")

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
    return "\n".join(out)


FILENAME = {
    "ApplicationSAML": "application_saml",
    "ActiveWebKey": "active_web_key",
    "EmailProviderSMTP": "email_provider_smtp",
    "EmailProviderHTTP": "email_provider_http",
    "SMSProviderTwilio": "sms_provider_twilio",
    "SMSProviderHTTP": "sms_provider_http",
}


if __name__ == "__main__":
    out_dir = sys.argv[1]

    for i, spec in enumerate(MESSAGING):
        path = "%s/%s_types.go" % (out_dir, FILENAME[spec["kind"]])
        with open(path, "w") as f:
            f.write(render_messaging(spec, first=(i == 0)))
        print("wrote", path)

    for spec in (SAML, ACTIVE_WEBKEY):
        path = "%s/%s_types.go" % (out_dir, FILENAME[spec["kind"]])
        with open(path, "w") as f:
            f.write(render_simple(spec))
        print("wrote", path)

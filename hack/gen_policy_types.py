#!/usr/bin/env python3
"""Generate the Zitadel policy managed resource API types.

Every Zitadel policy is a singleton of a scope with a fixed set of scalar
fields, so the Go types differ only in their kind name, their field list and the
prose that explains them. Writing seventeen near-identical files by hand is how
the field documentation drifts apart, so they are generated from one table
instead: the table below is the single place a policy's fields are declared.
"""

import textwrap

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


def field(go_name, json_name, go_type, doc, optional=True, default=None):
    """Describe one field of a policy."""
    return {
        "name": go_name,
        "json": json_name,
        "type": go_type,
        "doc": doc,
        "optional": optional,
        "default": default,
    }


def policy(kind, title, purpose, fields, extra_doc=""):
    return {
        "kind": kind,
        "title": title,
        "purpose": purpose,
        "fields": fields,
        "extra_doc": extra_doc,
    }


# The Go type a field maps to in the spec, the status, and the client.
BOOL = ("*bool", "bool", "*bool")
STR = ("*string", "string", "*string")
NUM = ("*int64", "uint32", "*int64")

ORGANIZATION_FIELDS = [
    field("OrganizationID", "organizationID", STR[0],
          "The ID of the organization the policy belongs to. Either `organizationID`, "
          "`organizationRef` or `organizationSelector` must be set."),
    field("OrganizationRef", "organizationRef", "*xpv1.Reference",
          "OrganizationRef references an Organization managed by this provider and "
          "uses its ID."),
    field("OrganizationSelector", "organizationSelector", "*xpv1.Selector",
          "OrganizationSelector selects an Organization managed by this provider and "
          "uses its ID."),
]

SCOPE_FIELDS = [
    field("Scope", "scope", STR[0],
          "The organization the policy was last read from."),
    field("Restore", "restore", "[]byte",
          "The policy this resource overwrote, kept so that deleting it can put the "
          "previous value back. Only used for a policy Zitadel cannot reset."),
]

POLICIES = [
    policy(
        "LockoutPolicy",
        "lockout policy",
        "controls how many failed attempts lock a user out",
        [
            field("MaxPasswordAttempts", "maxPasswordAttempts", NUM[0],
                  "Maximum password check attempts before the account is locked. "
                  "Attempts reset as soon as the password is checked correctly."),
            field("MaxOTPAttempts", "maxOTPAttempts", NUM[0],
                  "Maximum OTP check attempts before the account is locked. Attempts "
                  "reset as soon as an OTP is checked correctly."),
        ],
    ),
    policy(
        "NotificationPolicy",
        "notification policy",
        "controls which events make Zitadel send an email",
        [
            field("PasswordChange", "passwordChange", BOOL[0],
                  "Send a notification when a user changes their password."),
        ],
    ),
    policy(
        "PasswordAgePolicy",
        "password expiry policy",
        "controls when a password has to be changed",
        [
            field("MaxAgeDays", "maxAgeDays", NUM[0],
                  "Days after which a password expires. Zitadel treats 0 as \"never\"."),
            field("ExpireWarnDays", "expireWarnDays", NUM[0],
                  "Days before expiry at which the user is warned. Setting this "
                  "without `maxAgeDays` has no effect."),
        ],
    ),
    policy(
        "PasswordComplexityPolicy",
        "password complexity policy",
        "controls what a new password must contain",
        [
            field("MinLength", "minLength", NUM[0],
                  "Minimum password length."),
            field("HasUppercase", "hasUppercase", BOOL[0],
                  "Require an uppercase letter."),
            field("HasLowercase", "hasLowercase", BOOL[0],
                  "Require a lowercase letter."),
            field("HasNumber", "hasNumber", BOOL[0], "Require a number."),
            field("HasSymbol", "hasSymbol", BOOL[0],
                  "Require a symbol, such as `$`."),
        ],
    ),
    policy(
        "PrivacyPolicy",
        "privacy policy",
        "sets the legal and support links Zitadel shows",
        [
            field("TOSLink", "tosLink", STR[0], "Link to the Terms of Service."),
            field("PrivacyLink", "privacyLink", STR[0],
                  "Link to the Privacy Policy."),
            field("HelpLink", "helpLink", STR[0], "Link to the help or manual page."),
            field("SupportEmail", "supportEmail", STR[0],
                  "Support email address shown to users."),
            field("DocsLink", "docsLink", STR[0],
                  "Link to documentation, shown in the console."),
            field("CustomLink", "customLink", STR[0],
                  "Link to an external resource, shown as a button in the console."),
            field("CustomLinkText", "customLinkText", STR[0],
                  "Text of the custom link's button. Used only with `customLink`."),
        ],
    ),
    policy(
        "DomainPolicy",
        "domain policy",
        "controls how login names relate to an organization's domains",
        [
            field("UserLoginMustBeDomain", "userLoginMustBeDomain", BOOL[0],
                  "Require login names to be qualified with a domain, such as "
                  "`alice@example.com` rather than `alice`."),
            field("ValidateOrgDomains", "validateOrgDomains", BOOL[0],
                  "Verify that an organization's domains resolve before accepting "
                  "them. Zitadel rejects the change until they do."),
            field("SMTPSenderAddressMatchesInstanceDomain", "smtpSenderAddressMatchesInstanceDomain", BOOL[0],
                  "Require the SMTP sender address to match the instance domain."),
        ],
    ),
    policy(
        "LabelPolicy",
        "branding policy",
        "sets the colours, logo and watermark of an organization's login pages",
        [
            field("PrimaryColor", "primaryColor", STR[0],
                  "Primary colour, as a hex value such as `#5282C1`."),
            field("WarnColor", "warnColor", STR[0],
                  "Colour used for warnings, as a hex value."),
            field("BackgroundColor", "backgroundColor", STR[0],
                  "Page background colour, as a hex value."),
            field("FontColor", "fontColor", STR[0],
                  "Text colour, as a hex value."),
            field("PrimaryColorDark", "primaryColorDark", STR[0],
                  "Primary colour of the dark theme, as a hex value."),
            field("WarnColorDark", "warnColorDark", STR[0],
                  "Warning colour of the dark theme, as a hex value."),
            field("BackgroundColorDark", "backgroundColorDark", STR[0],
                  "Page background colour of the dark theme, as a hex value."),
            field("FontColorDark", "fontColorDark", STR[0],
                  "Text colour of the dark theme, as a hex value."),
            field("HideLoginNameSuffix", "hideLoginNameSuffix", BOOL[0],
                  "Hide the organization suffix on the login form. Takes effect when "
                  "the `urn:zitadel:iam:org:domain:primary:{domainname}` scope is "
                  "requested."),
            field("DisableWatermark", "disableWatermark", BOOL[0],
                  "Hide the Zitadel watermark."),
            field("ThemeMode", "themeMode", "*LabelThemeMode",
                  "Which theme the login pages use. One of `Auto`, `Light` or "
                  "`Dark`. Defaults to `Auto`."),
            field("LogoURL", "logoURL", STR[0],
                  "Where Zitadel serves the logo from. Reported only: assets are "
                  "uploaded outside the API, so this provider neither sets nor "
                  "removes them."),
            field("IconURL", "iconURL", STR[0],
                  "Where Zitadel serves the favicon from. Reported only, as with "
                  "`logoURL`."),
            field("LogoDarkURL", "logoDarkURL", STR[0],
                  "Where Zitadel serves the dark theme logo from. Reported only, as "
                  "with `logoURL`."),
            field("IconDarkURL", "iconDarkURL", STR[0],
                  "Where Zitadel serves the dark theme favicon from. Reported only, "
                  "as with `logoURL`."),
            field("FontURL", "fontURL", STR[0],
                  "Where Zitadel serves the custom font from. Reported only, as with "
                  "`logoURL`."),
        ],
    ),
]


THEME_MODE = """
// LabelThemeMode is which theme an organization's login pages use.
//
// +kubebuilder:validation:Enum=Auto;Light;Dark
type LabelThemeMode string

const (
	// LabelThemeModeAuto follows the visitor's own system preference.
	LabelThemeModeAuto LabelThemeMode = "Auto"

	// LabelThemeModeLight always uses the light theme.
	LabelThemeModeLight LabelThemeMode = "Light"

	// LabelThemeModeDark always uses the dark theme.
	LabelThemeModeDark LabelThemeMode = "Dark"
)
"""


def wrap_doc(doc, indent="\t"):
    lines = textwrap.wrap(" ".join(doc.split()), width=76 - len(indent))
    return "\n".join(indent + "// " + line for line in lines)


def render(p):
    kind = p["kind"]
    out = [LICENSE, "", "package v1alpha1", "", 'import (', '\t"reflect"', "",
           '\txpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"',
           '\txpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"',
           '\tmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"',
           '\t"k8s.io/apimachinery/pkg/runtime/schema"',
           ")", ""]

    # Spec fields: the organization reference plus the policy itself.
    out.append("// %sParameters are the configurable fields of a %s." % (kind, kind))
    out.append("type %sParameters struct {" % kind)
    for f in ORGANIZATION_FIELDS:
        out.append(wrap_doc(f["doc"]))
        out.append("\t// +optional")
        out.append("\t%s %s `json:\"%s,omitempty\"`" % (f["name"], f["type"], f["json"]))
        out.append("")
    for f in p["fields"]:
        out.append(wrap_doc(f["doc"]))
        out.append("\t// +optional")
        out.append("\t%s %s `json:\"%s,omitempty\"`" % (f["name"], f["type"], f["json"]))
        out.append("")
    out.append("}")
    out.append("")

    # Status fields: what Zitadel reported.
    out.append("// %sObservation is what the %s of an organization currently is." % (kind, p["title"]))
    out.append("type %sObservation struct {" % kind)
    out.append(wrap_doc("OrganizationID is the organization the policy belongs to."))
    out.append("\t// +optional")
    out.append("\tOrganizationID *string `json:\"organizationID,omitempty\"`")
    out.append("")
    out.append(wrap_doc("IsDefault is true while the organization still uses the instance default."))
    out.append("\t// +optional")
    out.append("\tIsDefault *bool `json:\"isDefault,omitempty\"`")
    out.append("")
    for f in p["fields"]:
        if f["type"].startswith("*xpv1."):
            continue
        # The observation mirrors the spec's types, pointers included.
        #
        # A policy value is often legitimately false or empty - a password may
        # not require a symbol, a privacy policy may have no custom link - and a
        # plain field would drop exactly those from the status, leaving an
        # operator unable to tell "Zitadel reports false" from "nothing was
        # observed".
        go_type = f["type"]
        out.append(wrap_doc(f["doc"]))
        out.append("\t// +optional")
        out.append("\t%s %s `json:\"%s,omitempty\"`" % (f["name"], go_type, f["json"]))
        out.append("")
    for f in SCOPE_FIELDS:
        out.append(wrap_doc(f["doc"]))
        out.append("\t// +optional")
        out.append("\t%s %s `json:\"%s,omitempty\"`" % (f["name"], f["type"], f["json"]))
        out.append("")
    out.append("}")
    out.append("")

    out.append("// A %s is an organization's %s." % (kind, p["title"]))
    out.append("//")
    out.append(textwrap.fill(p["purpose"] + ".", width=74, initial_indent="// ", subsequent_indent="// "))
    out.append("//")
    out.append(textwrap.fill(
        "It is a singleton: one resource describes the policy of one organization, "
        "and deleting it puts the organization back on the instance default rather "
        "than removing anything.", width=74,
        initial_indent="// ", subsequent_indent="// "))
    if p["extra_doc"]:
        out.append("//")
        out.append(textwrap.fill(p["extra_doc"], width=74, initial_indent="// ", subsequent_indent="// "))
    out.append("// +kubebuilder:object:root=true")
    out.append("// +kubebuilder:subresource:status")
    out.append("// +kubebuilder:storageversion")
    out.append("// +kubebuilder:printcolumn:name=\"Organization\",type=\"string\",JSONPath=\".status.atProvider.organizationID\"")
    out.append("// +kubebuilder:printcolumn:name=\"Default\",type=\"boolean\",JSONPath=\".status.atProvider.isDefault\"")
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
    out.append("// %sSpec defines the desired state of a %s." % (kind, kind))
    out.append("type %sSpec struct {" % kind)
    out.append("\txpv2.ManagedResourceSpec `json:\",inline\"`")
    out.append("\tForProvider              %sParameters `json:\"forProvider\"`" % kind)
    out.append("}")
    out.append("")
    out.append("// %sStatus is the observed state of a %s." % (kind, kind))
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
    out.append("// SetPolicyScope records the organization the policy belongs to.")
    out.append("func (mg *%s) SetPolicyScope(scope string) { mg.Status.AtProvider.Scope = StringPtr(scope) }" % kind)
    out.append("")
    out.append("// PolicyScope returns the organization the policy was last read from.")
    out.append("func (mg *%s) PolicyScope() string { return Deref(mg.Status.AtProvider.Scope) }" % kind)
    out.append("")
    out.append("// SetPolicyRestore records the policy to put back when this resource is deleted.")
    out.append("func (mg *%s) SetPolicyRestore(restore []byte) { mg.Status.AtProvider.Restore = restore }" % kind)
    out.append("")
    out.append("// PolicyRestore returns the recorded restore point.")
    out.append("func (mg *%s) PolicyRestore() []byte { return mg.Status.AtProvider.Restore }" % kind)
    out.append("")

    if kind == "LabelPolicy":
        out.append(THEME_MODE)

    return "\n".join(out)


if __name__ == "__main__":
    import sys
    out_dir = sys.argv[1]
    for p in POLICIES:
        name = p["kind"]
        snake = "".join("_" + c.lower() if c.isupper() else c for c in name).lstrip("_")
        path = f"{out_dir}/{snake}_types.go"
        with open(path, "w") as f:
            f.write(render(p))
        print("wrote", path)

#!/usr/bin/env python3
"""Generate the instance wide settings managed resource API types.

The feature flags, the registration restrictions and the secret generators are
the last instance wide things in Zitadel that are a single value rather than a
list or an object of their own. They differ in their fields and in which of the
four API calls they need, and agree on everything else: a scope, a fixed set of
scalar fields, and a read that answers before anything has been written.

They are generated from one table so that the field documentation cannot drift
apart the way four hand-written files would.
"""

import re
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


def snake_case(name):
    s1 = re.sub(r"(.)([A-Z][a-z]+)", r"\1_\2", name)
    return re.sub(r"([a-z0-9])([A-Z])", r"\1_\2", s1).lower()


BOOL = "*bool"
STR = "*string"
NUM = "*int64"
STRS = "[]string"
ENUMS = "[]string"

# (Go name, JSON name, Go type, documentation)
FEATURE_FIELDS = [
    ("LoginDefaultOrg", "loginDefaultOrg", BOOL,
     "Make the login screen use the default organization's settings when no "
     "organization context is set, rather than the organization's own."),
    ("UserSchema", "userSchema", BOOL,
     "Enable the user schema API, which manages a schema of extra data per user. "
     "The API is early and may change between Zitadel releases."),
    ("ImprovedPerformance", "improvedPerformance", ENUMS,
     "Execution paths Zitadel may shortcut in exchange for less work on write. "
     "Accepted values are `IMPROVED_PERFORMANCE_PROJECT_GRANT`, "
     "`IMPROVED_PERFORMANCE_PROJECT`, `IMPROVED_PERFORMANCE_USER_GRANT` and "
     "`IMPROVED_PERFORMANCE_ORG_DOMAIN_VERIFIED`."),
]

SYSTEM_FEATURE_FIELDS = [
    ("LoginDefaultOrg", "loginDefaultOrg", BOOL,
     "Make the login screen use the default organization's settings when no "
     "organization context is set, rather than the organization's own."),
    ("UserSchema", "userSchema", BOOL,
     "Enable the user schema API, which manages a schema of extra data per user."),
]

RESTRICTION_FIELDS = [
    ("DisallowPublicOrgRegistration", "disallowPublicOrgRegistration", BOOL,
     "Stop anyone registering their own organization. Off by default, so "
     "turning it off lets anyone sign up an organization again."),
    ("AllowedLanguages", "allowedLanguages", STRS,
     "The languages the login screen offers, such as `en` and `de`. Empty "
     "means every language Zitadel supports."),
]

SECRET_GENERATOR_FIELDS = [
    ("GeneratorType", "generatorType", STR,
     "Which of Zitadel's codes this shapes, such as "
     "`SECRET_GENERATOR_TYPE_INIT_CODE` or `SECRET_GENERATOR_TYPE_APP_SECRET`. "
     "Each type is a generator of its own, so two of these resources may manage "
     "two different types at once."),
    ("Length", "length", NUM,
     "How many characters the code has. Zitadel refuses a length below four."),
    ("Expiry", "expiry", STR,
     "How long the code stays usable, as a Go duration such as `5m` or `1h`. "
     "Zitadel refuses an expiry that is longer than the code's own lifetime."),
    ("IncludeLowerLetters", "includeLowerLetters", BOOL,
     "Let the code contain lower case letters."),
    ("IncludeUpperLetters", "includeUpperLetters", BOOL,
     "Let the code contain upper case letters."),
    ("IncludeDigits", "includeDigits", BOOL,
     "Let the code contain digits."),
    ("IncludeSymbols", "includeSymbols", BOOL,
     "Let the code contain symbols."),
]

SETTINGS = [
    {
        "kind": "InstanceFeatures",
        "title": "Zitadel instance feature flags",
        "purpose": "turn the instance wide feature flags on or off",
        "fields": FEATURE_FIELDS + [
            ("DebugOidcParentError", "debugOidcParentError", BOOL,
             "Return the underlying error to an OIDC client instead of a generic "
             "failure. It can reveal details about the system to anyone who asks, "
             "so it is meant for debugging rather than everyday use."),
        ],
        "example": [
            ("loginDefaultOrg", "true"),
            ("userSchema", "false"),
            ("debugOidcParentError", "false"),
            ("improvedPerformance", '["IMPROVED_PERFORMANCE_PROJECT", "IMPROVED_PERFORMANCE_USER_GRANT"]'),
        ],
        # The read tells us which scope set the flag, which is what a policy
        # inherited from an instance default also reports.
        "sources": True,
    },
    {
        "kind": "SystemFeatures",
        "title": "Zitadel system feature flags",
        "purpose": "turn the system wide feature flags on or off",
        "fields": SYSTEM_FEATURE_FIELDS,
        "example": [
            ("loginDefaultOrg", "true"),
            ("userSchema", "false"),
        ],
        "sources": True,
    },
    {
        "kind": "InstanceRestrictions",
        "title": "Zitadel instance registration restrictions",
        "purpose": "limit what anyone may register on this instance",
        "fields": RESTRICTION_FIELDS,
        "example": [
            ("disallowPublicOrgRegistration", "true"),
            ("allowedLanguages", '["en", "de"]'),
        ],
        "sources": False,
    },
    {
        "kind": "InstanceSecretGenerator",
        "title": "Zitadel instance secret generator",
        "purpose": "shape one of the codes Zitadel sends out",
        "fields": SECRET_GENERATOR_FIELDS,
        "required": ["GeneratorType"],
        "example": [
            ("generatorType", "SECRET_GENERATOR_TYPE_INIT_CODE"),
            ("length", "6"),
            ("expiry", '"5m"'),
            ("includeLowerLetters", "true"),
            ("includeUpperLetters", "false"),
            ("includeDigits", "true"),
            ("includeSymbols", "false"),
        ],
        "sources": False,
        # One per generator type, so the type is what tells two of them apart
        # and what the resource is identified by.
        "keyed": True,
    },
]


SETTINGS_ACCESSORS = """
// SetPolicyScope records the scope these settings were last read from.
func (mg *KIND) SetPolicyScope(scope string) { mg.Status.AtProvider.Scope = scope }

// PolicyScope returns the recorded scope.
func (mg *KIND) PolicyScope() string { return mg.Status.AtProvider.Scope }

// SetPolicyRestore records the value to put back when this resource is deleted.
func (mg *KIND) SetPolicyRestore(restore []byte) { mg.Status.Restore = restore }

// PolicyRestore returns the recorded restore point.
func (mg *KIND) PolicyRestore() []byte { return mg.Status.Restore }
"""


def render(s):
    kind = s["kind"]
    fields = s["fields"]
    obs = "".join(
        "\t%s %s `json:\"%s,omitempty\"`\n" % (
            f[0],
            f[2].replace("*", "").replace("[]string", "[]string"),
            f[1],
        )
        for f in fields
    )

    out = [LICENSE, ""]
    out.append("package v1alpha1\n")
    out.append("import (")
    out.append('\t"reflect"\n')
    out.append('\txpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"')
    out.append('\txpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"')
    out.append('\tmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"')
    out.append('\t"k8s.io/apimachinery/pkg/runtime/schema"\n)\n')

    # The spec is its own type embedding the Crossplane managed resource spec.
    # That embedded field is what the methodsets generator keys on: without it
    # a kind looks like an ordinary struct and no GetCondition is written for
    # it, which leaves the controller unable to compile against the harness.
    out.append("// %sSpec defines the desired state of a %s." % (kind, kind))
    out.append("type %sSpec struct {" % kind)
    out.append('\txpv2.ManagedResourceSpec `json:",inline"`')
    out.append('\tForProvider %sParameters `json:"forProvider"`' % kind)
    out.append("}\n")

    out.append("// %sParameters is the desired configuration of a %s." % (kind, s["title"]))
    out.append("type %sParameters struct {" % kind)
    for f in fields:
        req = "required" if f[0] in s.get("required", []) else "optional"
        out.append("\t// %s controls the following. %s" % (f[0], f[3].rstrip('.')))
        out.append("\t//")
        out.append("\t// It is %s." % req)
        out.append("\t%s %s `json:\"%s,omitempty\"`" % (f[0], f[2], f[1]))
    out.append("}\n")

    out.append("// %sObservation is what Zitadel reports about a %s." % (kind, s["title"]))
    out.append("type %sObservation struct {" % kind)
    # The scope is recorded for every one of them, singleton or not, because the
    # harness uses it on the delete path: a terminating resource acts on the
    # scope it last read rather than on one its reference has since moved to.
    out.append("\t// Scope is what identifies these settings. It is empty for a")
    out.append("\t// singleton and carries the generator type for the keyed kind.")
    out.append("\tScope string `json:\"scope,omitempty\"`\n")
    for f in fields:
        out.append("\t%s %s `json:\"%s,omitempty\"`" % (f[0], f[2].replace("*", ""), f[1]))
    if s.get("sources"):
        for f in fields:
            if f[2] == "*bool":
                out.append("\t%sSource string `json:\"%sSource,omitempty\"`" % (f[0], f[1][:1].lower() + f[1][1:]))
            elif f[2] == "[]string":
                out.append("\t%sSource string `json:\"%sSource,omitempty\"`" % (f[0], f[1][:1].lower() + f[1][1:]))
    out.append("}\n")

    out.append("// %sStatus reports the observed state of a %s." % (kind, s["title"]))
    out.append("type %sStatus struct {" % kind)
    out.append("\txpv1.ResourceStatus `json:\",inline\"`\n")
    out.append("\t// AtProvider is the observed state.")
    out.append("\tAtProvider %sObservation `json:\"atProvider,omitempty\"`\n" % kind)
    out.append("\t// Restore is the value this resource overwrote, kept so that deleting")
    out.append("\t// it can put it back.")
    out.append("\tRestore []byte `json:\"restore,omitempty\"`")
    out.append("}\n")

    out.append("// +kubebuilder:object:root=true")
    out.append("// +kubebuilder:subresource:status")
    out.append("// +kubebuilder:resource:scope=Namespaced,categories={zitadel}")
    out.append("// +kubebuilder:printcolumn:name=\"READY\",type=\"string\",JSONPath=\".status.conditions[?(@.type=='Ready')].status\"")
    out.append("// +kubebuilder:printcolumn:name=\"SYNCED\",type=\"string\",JSONPath=\".status.conditions[?(@.type=='Synced')].status\"")
    out.append("// +kubebuilder:printcolumn:name=\"AGE\",type=\"date\",JSONPath=\".metadata.creationTimestamp\"")
    out.append("")
    out.append("// %s is a managed resource that %s." % (kind, s["purpose"]))
    out.append("//")
    out.append("// This is a namespaced object managing state that belongs to the whole")
    out.append("// Zitadel instance. Two of them in different namespaces would be asking")
    out.append("// for the same settings, and the second would win.")
    out.append("// +kubebuilder:object:generate=true")
    out.append("type %s struct {" % kind)
    out.append("\tmetav1.TypeMeta   `json:\",inline\"`")
    out.append("\tmetav1.ObjectMeta `json:\"metadata,omitempty\"`\n")
    out.append("\tSpec   %sSpec   `json:\"spec\"`" % kind)
    out.append("\tStatus %sStatus `json:\"status,omitempty\"`" % kind)
    out.append("}\n")

    # Every kind implements the policy harness's accessors, which the generator
    # does not write for a settings kind.
    out.append(SETTINGS_ACCESSORS.replace("KIND", kind))

    out.append("// +kubebuilder:object:root=true")
    out.append("")
    out.append("// %sList contains a list of %s." % (kind, kind))
    out.append("type %sList struct {" % kind)
    out.append("\tmetav1.TypeMeta `json:\",inline\"`")
    out.append("\tmetav1.ListMeta `json:\"metadata,omitempty\"`")
    out.append("\tItems           []%s `json:\"items\"`" % kind)
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

    return "\n".join(out) + "\n"

    out.append("// GetCondition of this %s." % kind)
    out.append("func (mg *%s) GetCondition(ct xpv1.ConditionType) xpv1.Condition {" % kind)
    out.append("\treturn mg.Status.GetCondition(ct)")
    out.append("}\n")

    out.append("// GetConditions of this %s." % kind)
    out.append("func (mg *%s) GetConditions() []xpv1.Condition {" % kind)
    out.append("\treturn mg.Status.GetConditions()")
    out.append("}\n")

    out.append("// SetConditions of this %s." % kind)
    out.append("func (mg *%s) SetConditions(c ...xpv1.Condition) { mg.Status.SetConditions(c...) }\n" % kind)

    out.append("// GetObservation returns the observed state.")
    out.append("func (mg *%s) GetObservation() *%sObservation {" % (kind, kind))
    out.append("\treturn &mg.Status.AtProvider")
    out.append("}")
    out.append("")
    return "\n".join(out)


if __name__ == "__main__":
    out_dir = sys.argv[1]
    for s in SETTINGS:
        path = "%s/%s_types.go" % (out_dir, snake_case(s["kind"]))
        with open(path, "w") as f:
            f.write(render(s))
        print("wrote", path)

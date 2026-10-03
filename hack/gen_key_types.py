#!/usr/bin/env python3
"""Generate the key and membership managed resource API types.

These are the last four resources that are not settings or list members: the
signing keys an application verifies a token with, the one active key, the key
an application authenticates itself with, and a user's membership of a project.

They differ in what identifies them and in which API carries them, and agree on
everything else, so they are generated from one table.
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
STRS = "[]string"
NUMS = "[]string"

ORG_FIELDS = '''	// OrganizationID is the ID of the organization the resource belongs to.
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

PROJECT_FIELDS = ORG_FIELDS + '''
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

USER_FIELDS = PROJECT_FIELDS + '''
	// UserID is the ID of the user the grant applies to. Either `userID`,
	// `userRef` or `userSelector` must be set.
	//
	// +optional
	UserID *string `json:"userID,omitempty"`

	// UserRef references a HumanUser or ServiceAccount managed by this provider
	// and uses its ID.
	//
	// +optional
	UserRef *xpv1.Reference `json:"userRef,omitempty"`

	// UserSelector selects a HumanUser or ServiceAccount managed by this provider
	// and uses its ID.
	//
	// +optional
	UserSelector *xpv1.Selector `json:"userSelector,omitempty"`
'''

# Appended to a field's documentation where the schema has to enforce something
# Zitadel insists on, so an operator learns it at apply time rather than from a
# ZITADEL error later.
REQUIRED = "+kubebuilder:validation:MinItems=1"

KIND = {
    "WebKey": [
        ("Algorithm", "algorithm", STR,
         "`rsa`, `ecdsa` or `ed25519`. Zitadel generates the key; the private half "
         "never leaves it, so there is no material to supply or read back."),
        ("RSABits", "rsaBits", STR,
         "The key size for an RSA key: `RSA_BITS_2048`, `RSA_BITS_3072` or "
         "`RSA_BITS_4096`. Zitadel's default is 2048. Ignored for the other "
         "algorithms."),
        ("RSAHasher", "rsaHasher", STR,
         "The signing algorithm for an RSA key: `RSA_HASHER_SHA256`, "
         "`RSA_HASHER_SHA384` or `RSA_HASHER_SHA512`. Zitadel's default is "
         "SHA256. Ignored for the other algorithms."),
        ("ECDSACurve", "ecdsaCurve", STR,
         "The curve for an ECDSA key: `ECDSA_CURVE_P256`, `ECDSA_CURVE_P384` or "
         "`ECDSA_CURVE_P512`. Zitadel documents P-256 as the default but rejects "
         "an unset curve rather than choosing one, so an empty field means "
         "P-256. Ignored for the other algorithms."),
    ],
    "ApplicationKey": [
        ("ExpirationDate", "expirationDate", STR,
         "When the key stops working, as a duration from now such as `8760h`. "
         "Empty means it does not expire. Zitadel returns the private half once, "
         "at creation, and never again."),
    ],
    "ProjectMember": [
        ("Roles", "roles", STRS,
         "The roles the user holds on the project, such as `PROJECT_OWNER`. A "
         "project may define roles of its own, so the keys are not fixed. "
         "Zitadel refuses a member with no role at all, so at least one is "
         "required: to take every role away, delete this resource instead.",
         REQUIRED),
    ],
}

OBS = {
    "WebKey": [
        ("ID", "string"),
        ("Algorithm", "string"),
        ("State", "string"),
        ("CreationDate", "string"),
        ("ChangeDate", "string"),
    ],
    "ApplicationKey": [
        ("ID", "string"),
        ("CreationDate", "string"),
        ("ExpirationDate", "string"),
    ],
    "ProjectMember": [
        ("UserID", "string"),
        ("Roles", "[]string"),
        ("UserName", "string"),
        ("DisplayName", "string"),
        ("ProjectName", "string"),
    ],
}

SPEC = {
    "WebKey": dict(
        purpose="hold one of Zitadel's own signing keys",
        fields="",
        observation=(
            "// WebKeyObservation is what Zitadel reports about a signing key.\n"
            "type WebKeyObservation struct {\n"
            "\tID           string `json:\"id,omitempty\"`\n"
            "\tAlgorithm    string `json:\"algorithm,omitempty\"`\n"
            "\tState        string `json:\"state,omitempty\"`\n"
            "\tCreationDate string `json:\"creationDate,omitempty\"`\n"
            "\tChangeDate   string `json:\"changeDate,omitempty\"`\n"
            "}\n\n"
        ),
        doc=(
            "// Zitadel signs its tokens with one of these, and an application verifies\n"
            "// them with the active one. The private half never leaves Zitadel, so a key\n"
            "// this provider manages is an identity and a state and nothing more:\n"
            "//\n"
            "//   * Zitadel creates it inactive, so it verifies nothing until activated.\n"
            "//     ActivatedWebKey is what does that.\n"
            "//   * The key type cannot be changed, only replaced, so changing the algorithm\n"
            "//     is reported as needing a new key rather than silently ignored.\n"
            "//   * Zitadel refuses to remove the active key, so it has to be activated\n"
            "//     away first.\n"
        ),
    ),
    "ApplicationKey": dict(
        purpose="hold the key an application authenticates itself with",
        fields=PROJECT_FIELDS,
        observation=(
            "// ApplicationKeyObservation is what Zitadel reports about an application key.\n"
            "type ApplicationKeyObservation struct {\n"
            "\tID             string `json:\"id,omitempty\"`\n"
            "\tProjectID      string `json:\"projectID,omitempty\"`\n"
            "\tApplicationID  string `json:\"applicationID,omitempty\"`\n"
            "\tCreationDate   string `json:\"creationDate,omitempty\"`\n"
            "\tExpirationDate string `json:\"expirationDate,omitempty\"`\n"
            "}\n\n"
        ),
        doc=(
            "// The key is written to the connection secret as `key.json`, the same shape as\n"
            "// a machine key, because Zitadel returns the private half exactly once: at\n"
            "// creation. There is no way to read it again, so a key whose secret was lost\n"
            "// can only be replaced.\n"
        ),
        extra_application=True,
    ),
    "ProjectMember": dict(
        purpose="grant a user roles on a project",
        fields=USER_FIELDS,
        observation=(
            "// ProjectMemberObservation is what Zitadel reports about a project member.\n"
            "type ProjectMemberObservation struct {\n"
            "\tUserID      string   `json:\"userID,omitempty\"`\n"
            "\tRoles       []string `json:\"roles,omitempty\"`\n"
            "\tUserName    string   `json:\"userName,omitempty\"`\n"
            "\tDisplayName string   `json:\"displayName,omitempty\"`\n"
            "\tProjectName string   `json:\"projectName,omitempty\"`\n"
            "}\n\n"
        ),
        doc=(
            "// Deleting the resource takes the user off the project. An empty role list is\n"
            "// not the same as deleting: it leaves the user a member who can see the\n"
            "// project and hold nothing on it.\n"
        ),
    ),
}

APP_REF = '''
	// ApplicationID is the ID of the application the key belongs to. Either
	// `applicationID`, `applicationRef` or `applicationSelector` must be set.
	//
	// +optional
	ApplicationID *string `json:"applicationID,omitempty"`

	// ApplicationRef references an OIDCApplication or ApplicationAPI managed by
	// this provider and uses its ID.
	//
	// +optional
	ApplicationRef *xpv1.Reference `json:"applicationRef,omitempty"`

	// ApplicationSelector selects an OIDCApplication or ApplicationAPI managed by
	// this provider and uses its ID.
	//
	// +optional
	ApplicationSelector *xpv1.Selector `json:"applicationSelector,omitempty"`
'''

# The connection secret a generated key is written to.
KEY_SECRET = '''
	// Publisher is where the generated key is written, so that a workload can
	// mount it. The private half is returned by Zitadel exactly once, at
	// creation, so this is the only chance to keep it.
	Publisher config.TypedLocalSecretPublisher `json:"publisherRef"`
'''


def render(kind):
    spec = SPEC[kind]
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

    out.append("// %sParameters is the desired configuration of a %s." % (kind, spec["purpose"]))
    out.append("type %sParameters struct {" % kind)
    out.append(spec["fields"])
    for entry in KIND[kind]:
        name, json_name, go_type, doc = entry[:4]
        # A fifth element is a kubebuilder marker the schema needs.
        marker = entry[4] if len(entry) > 4 else None

        out.append("\t// %s controls the following. %s" % (name, doc.rstrip(".")))
        out.append("\t//")
        out.append("\t// +optional")
        if marker:
            out.append("\t// " + marker)
        out.append('\t%s %s `json:"%s,omitempty"`' % (name, go_type, json_name))
    if kind == "ApplicationKey":
        out.append(APP_REF)
    out.append("}\n")

    out.append(spec["observation"])

    out.append("// %sStatus reports the observed state of a %s." % (kind, spec["purpose"]))
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
    if kind == "ProjectMember":
        out.append("// +kubebuilder:printcolumn:name=\"ROLES\",type=\"string\",JSONPath=\".spec.forProvider.roles\"")
    out.append("// +kubebuilder:printcolumn:name=\"AGE\",type=\"date\",JSONPath=\".metadata.creationTimestamp\"")
    out.append("")
    out.append("// %s is a managed resource that %s." % (kind, spec["purpose"]))
    out.append("//")
    # A blank line between the comment lines would split the doc comment into
    # separate blocks, which detaches the markers above it from the type and
    # leaves the generator writing no methods for it.
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
    out.append("}\n")

    return "\n".join(out)


if __name__ == "__main__":
    out_dir = sys.argv[1]
    for kind in KIND:
        path = "%s/%s_types.go" % (out_dir, {
            "WebKey": "web_key",
            "ApplicationKey": "application_key",
            "ProjectMember": "project_member",
        }[kind])
        with open(path, "w") as f:
            f.write(render(kind))
        print("wrote", path)

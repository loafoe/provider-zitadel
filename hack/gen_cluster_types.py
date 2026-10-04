#!/usr/bin/env python3
"""Generate the cluster scoped variants of the instance wide managed resources.

Nineteen of the eighty kinds manage state that belongs to a whole Zitadel
instance: the feature flags, the ten instance wide policy defaults, the
registration restrictions, the secret generators, the active signing key, and the
four messaging providers.

Namespaced, each of those is a singleton that any namespace can claim, so two of
them in different namespaces fight over the same settings and the second one
wins. That is exactly what the cluster scoped variant removes: a resource in the
cluster scoped group exists once, so the API server makes the conflict
unrepresentable rather than leaving it to be discovered.

The variant shares the parameters, the observation and the status with the kind it
mirrors - the same Go types, not copies - so there is one definition of what a
field means. Only the spec, the list and the object itself are new, plus two
small methods that let the controller that already reconciles the namespaced kind
reconcile this one unchanged.

Every namespaced kind is kept. This adds to them.
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

# The namespaced kind each one mirrors, and why it is worth mirroring.
# Everything else is left namespaced on purpose: see README.md.
KINDS = [
    ("InstanceFeatures", "the instance wide feature flags",
     "It is the singleton of the instance, so two namespaced copies would be\n// asking for the same flags and the second would win."),
    ("SystemFeatures", "the system wide feature flags",
     "It is the singleton of the instance, so two namespaced copies would be\n// asking for the same flags and the second would win."),
    ("InstanceRestrictions", "what anyone may register on the instance",
     "It is the singleton of the instance, so two namespaced copies would be\n// asking for the same restrictions and the second would win."),
    ("InstanceSecretGenerator", "the shape of one of Zitadel's generated codes",
     "A secret generator belongs to no organization and to no namespace: a\n// namespace here implies a separation that does not exist."),
    ("ActiveWebKey", "which of Zitadel's signing keys is the active one",
     "There is one active key on an instance, so a namespaced copy in two\n// namespaces would be activating different keys at once."),
    ("DefaultLockoutPolicy", "when a user of any organization is locked out",
     "It is the instance default, so it exists once however it is scoped."),
    ("DefaultNotificationPolicy", "which events send an email for any organization",
     "It is the instance default, so it exists once however it is scoped."),
    ("DefaultPasswordAgePolicy", "when passwords expire across the instance",
     "It is the instance default, so it exists once however it is scoped."),
    ("DefaultPasswordComplexityPolicy", "what a password must contain, instance wide",
     "It is the instance default, so it exists once however it is scoped."),
    ("DefaultPrivacyPolicy", "the legal and support links an organization inherits",
     "It is the instance default, so it exists once however it is scoped."),
    ("DefaultDomainPolicy", "how login names relate to domains, instance wide",
     "It is the instance default, so it exists once however it is scoped."),
    ("DefaultLabelPolicy", "the theme an organization inherits for its login pages",
     "It is the instance default, so it exists once however it is scoped."),
    ("DefaultLoginPolicy", "what an organization allows until it customises its own",
     "It is the instance default, so it exists once however it is scoped."),
    ("DefaultOIDCSettings", "the instance wide OIDC defaults",
     "It is the instance default, so it exists once however it is scoped."),
    ("DefaultSecuritySettings", "the instance wide security settings",
     "It is the instance default, so it exists once however it is scoped."),
    ("EmailProviderSMTP", "the SMTP server Zitadel sends email through",
     "Zitadel sends through one of these, so two namespaced copies would be\n// fighting over the same provider."),
    ("EmailProviderHTTP", "the endpoint Zitadel posts email to",
     "Zitadel sends through one of these, so two namespaced copies would be\n// fighting over the same provider."),
    ("SMSProviderTwilio", "the Twilio account Zitadel sends SMS through",
     "Zitadel sends through one of these, so two namespaced copies would be\n// fighting over the same provider."),
    ("SMSProviderHTTP", "the endpoint Zitadel posts SMS to",
     "Zitadel sends through one of these, so two namespaced copies would be\n// fighting over the same provider."),
]


def cluster(name):
    return "Cluster" + name


def render(base):
    """One cluster scoped kind, mirroring the namespaced one of the same name."""
    kind = cluster(base)

    out = [LICENSE, ""]
    out.append("package v1alpha1\n")
    out.append("import (")
    out.append('\t"reflect"\n')
    # xpv1 is not imported: the status is the namespaced one under another name,
    # so nothing here mentions its fields.
    out.append('\txpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"')
    out.append('\tmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"')
    out.append('\t"k8s.io/apimachinery/pkg/runtime/schema"\n')
    out.append('\n\tmanaged "github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"')
    out.append(")\n")

    out.append("// %sSpec is the desired state of a %s." % (kind, kind))
    out.append("//")
    out.append("// It carries the same parameters as the namespaced %s, so the two kinds" % base)
    out.append("// cannot describe a different thing from one another.")
    out.append("type %sSpec struct {" % kind)
    out.append('\txpv2.ManagedResourceSpec `json:",inline"`')
    out.append('\tForProvider managed.%sParameters `json:"forProvider"`' % base)
    out.append("}\n")

    out.append("// %sStatus is the observed state of a %s." % (kind, kind))
    out.append("//")
    out.append("// It is the namespaced status under another name, for the same reason: one")
    out.append("// definition of what a field means.")
    out.append("type %sStatus = managed.%sStatus" % (kind, base))
    out.append("")

    out.append("// +kubebuilder:object:root=true")
    out.append("// +kubebuilder:subresource:status")
    out.append("// +kubebuilder:resource:scope=Cluster,categories={zitadel}")
    out.append("// +kubebuilder:printcolumn:name=\"READY\",type=\"string\",JSONPath=\".status.conditions[?(@.type=='Ready')].status\"")
    out.append("// +kubebuilder:printcolumn:name=\"SYNCED\",type=\"string\",JSONPath=\".status.conditions[?(@.type=='Synced')].status\"")
    out.append("// +kubebuilder:printcolumn:name=\"AGE\",type=\"date\",JSONPath=\".metadata.creationTimestamp\"")
    out.append("")
    out.append("// %s is the cluster scoped form of %s." % (kind, base))
    out.append("//")
    out.append("// It manages %s." % purpose(base))
    out.append("//")
    out.append("// %s" % reason(base))
    out.append("//")
    out.append("// The namespaced kind is kept and behaves identically. Use whichever suits the")
    out.append("// cluster: a namespaced one to keep instance configuration beside the team or")
    out.append("// environment that owns it, this one when it should be unambiguous.")
    out.append("//")
    out.append("// It references a ClusterProviderConfig, because a cluster scoped object cannot")
    out.append("// name a secret in a namespace.")
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

    out.append(REUSE.replace("@@KIND@@", kind).replace("@@BASE@@", base))
    return "\n".join(out)


REUSE = '''// AsNamespaced returns this resource as the namespaced @@BASE@@ it mirrors.
//
// The controller that already reconciles @@BASE@@ is reused unchanged: this is
// the same object, the same fields and the same external resource, so it is
// handed over as the kind it knows. Nothing is copied that could disagree -
// there is only one definition of the spec, the observation and the status.
func (mg *@@KIND@@) AsNamespaced() *managed.@@BASE@@ {
	return &managed.@@BASE@@{
		TypeMeta:   mg.TypeMeta,
		ObjectMeta: *mg.ObjectMeta.DeepCopy(),
		Spec: managed.@@BASE@@Spec{
			ManagedResourceSpec: *mg.Spec.ManagedResourceSpec.DeepCopy(),
			ForProvider:         mg.Spec.ForProvider,
		},
		Status: mg.Status,
	}
}

// AdoptNamespaced copies back what the namespaced controller recorded.
//
// The external name and the status are what a reconcile produces, so they are
// taken from the resource it was given. The spec is left alone: it is what the
// operator wrote, and it is the same value either way.
func (mg *@@KIND@@) AdoptNamespaced(src *managed.@@BASE@@) {
	mg.ObjectMeta = *src.ObjectMeta.DeepCopy()
	mg.Status = src.Status
}
'''

# What each kind manages, and why the cluster scoped form is worth having.
REASONS = {
    "InstanceFeatures": "It is the singleton of the instance, so two namespaced copies\n// would be asking for the same flags and the second would win.",
    "SystemFeatures": "It is the singleton of the instance, so two namespaced copies\n// would be asking for the same flags and the second would win.",
    "InstanceRestrictions": "It is the singleton of the instance, so two namespaced copies\n// would be asking for the same restrictions and the second would win.",
    "InstanceSecretGenerator": "A secret generator belongs to no organization and to no namespace: a\n// namespace here implies a separation that does not exist.",
    "ActiveWebKey": "There is one active key on an instance, so a namespaced copy in two\n// namespaces would be activating different keys at once.",
    "EmailProviderSMTP": "Zitadel sends through one of these, so two namespaced copies would\n// be fighting over the same provider.",
    "EmailProviderHTTP": "Zitadel sends through one of these, so two namespaced copies would\n// be fighting over the same provider.",
    "SMSProviderTwilio": "Zitadel sends through one of these, so two namespaced copies would\n// be fighting over the same provider.",
    "SMSProviderHTTP": "Zitadel sends through one of these, so two namespaced copies would\n// be fighting over the same provider.",
}

DEFAULT = "It is the instance default, so it exists once however it is scoped."


def purpose(base):
    if base.startswith("Default"):
        rest = base[len("Default"):]
        return "the instance wide default for the %s of any organization" % (rest[0].lower() + rest[1:])

    return {
        "InstanceFeatures": "the instance wide feature flags",
        "SystemFeatures": "the system wide feature flags",
        "InstanceRestrictions": "what anyone may register on the instance",
        "InstanceSecretGenerator": "the shape of one of Zitadel's generated codes",
        "ActiveWebKey": "which of Zitadel's signing keys is the active one",
        "EmailProviderSMTP": "the SMTP server Zitadel sends email through",
        "EmailProviderHTTP": "the endpoint Zitadel posts email to",
        "SMSProviderTwilio": "the Twilio account Zitadel sends SMS through",
        "SMSProviderHTTP": "the endpoint Zitadel posts SMS to",
    }[base]


def reason(base):
    return REASONS.get(base, DEFAULT)


def snake(name):
    import re
    s = re.sub(r"(.)([A-Z][a-z]+)", r"\1_\2", name)
    return re.sub(r"([a-z0-9])([A-Z])", r"\1_\2", s).lower()


if __name__ == "__main__":
    out_dir = sys.argv[1]
    for base, _purpose, _reason in KINDS:
        path = "%s/%s_types.go" % (out_dir, snake(cluster(base)))
        with open(path, "w") as f:
            f.write(render(base))
        print("wrote", path)

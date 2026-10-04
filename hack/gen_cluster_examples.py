#!/usr/bin/env python3
"""Generate one example per cluster scoped managed resource.

Each example is the namespaced manifest with the namespace dropped, the provider
configuration reference pointed at a ClusterProviderConfig, and a note on why the
cluster scoped form exists at all.
"""

import os
import re
import sys

NOTE = """# An example of a Cluster%s.
#
# This is the cluster scoped form of %s. The namespaced kind is kept and
# behaves identically; use whichever suits the cluster.
#
# It manages %s.
#
# %s
#
# A cluster scoped resource cannot name a secret in a namespace, so it
# references a ClusterProviderConfig, whose credential references say which
# namespace their secret is in.
"""


def snake(name):
    s = re.sub(r"(.)([A-Z][a-z]+)", r"\1_\2", name)
    return re.sub(r"([a-z0-9])([A-Z])", r"\1_\2", s).lower()


def transform(text, kind, manages, why):
    out = text
    # The namespaced example's own header is replaced wholesale: it describes the
    # namespaced kind, and this one is about the cluster scoped form of it.
    out = re.sub(r"^(?:#.*\n)+", "", out, count=1)

    out = out.replace(
        "apiVersion: zitadel.m.crossplane.io/v1alpha1",
        "apiVersion: zitadel.crossplane.io/v1alpha1")
    out = out.replace("kind: " + kind + "\n", "kind: Cluster" + kind + "\n", 1)
    # A cluster scoped object has no metadata namespace.
    # A Kubernetes name may not contain an underscore, so the cluster examples
    # are named with hyphens even where the file name uses them.
    out = re.sub(r"^metadata:\n(?:  \w[^\n]*\n)+",
                 "metadata:\n  name: %s\n" % snake("Cluster" + kind).replace("_", "-"),
                 out, count=1, flags=re.M)

    out = out.replace("    name: zitadel\n    kind: ProviderConfig",
                      "    name: zitadel\n    kind: ClusterProviderConfig")

    # The header goes in front, once the API version is known.
    out = out.replace("apiVersion: zitadel.crossplane.io/v1alpha1",
                      (NOTE % (kind, kind, manages, why))
                      + "apiVersion: zitadel.crossplane.io/v1alpha1", 1)
    return out


# Managed state of each kind, in the cluster scoped form's own words.
MANAGES = {
    "InstanceFeatures": "the feature flags of the whole instance",
    "SystemFeatures": "the system wide feature flags",
    "InstanceRestrictions": "what anyone may register on the instance",
    "InstanceSecretGenerator": "the shape of one of Zitadel's generated codes",
    "ActiveWebKey": "which of Zitadel's signing keys is the active one",
    "DefaultLockoutPolicy": "the lockout policy of any organization",
    "DefaultNotificationPolicy": "the notification policy of any organization",
    "DefaultPasswordAgePolicy": "when passwords expire across the instance",
    "DefaultPasswordComplexityPolicy": "what a password must contain, instance wide",
    "DefaultPrivacyPolicy": "the legal and support links an organization inherits",
    "DefaultDomainPolicy": "how login names relate to domains, instance wide",
    "DefaultLabelPolicy": "the theme an organization inherits for its login pages",
    "DefaultLoginPolicy": "what an organization allows until it customises its own",
    "DefaultOIDCSettings": "the instance wide OIDC defaults",
    "DefaultSecuritySettings": "the instance wide security settings",
    "EmailProviderSMTP": "the SMTP server Zitadel sends email through",
    "EmailProviderHTTP": "the endpoint Zitadel posts email to",
    "SMSProviderTwilio": "the Twilio account Zitadel sends SMS through",
    "SMSProviderHTTP": "the endpoint Zitadel posts SMS to",
}

WHY = {
    "InstanceFeatures": "It is the singleton of the instance, so two namespaced copies\n# would be asking for the same flags and the second would win.",
    "SystemFeatures": "It is the singleton of the instance, so two namespaced copies\n# would be asking for the same flags and the second would win.",
    "InstanceRestrictions": "It is the singleton of the instance, so two namespaced copies\n# would be asking for the same restrictions and the second would win.",
    "InstanceSecretGenerator": "A secret generator belongs to no organization and to no namespace:\n# a namespace here implies a separation that does not exist.",
    "ActiveWebKey": "There is one active key on an instance, so a namespaced copy in two\n# namespaces would be activating different keys at once.",
    "EmailProviderSMTP": "Zitadel sends through one of these, so two namespaced copies would\n# be fighting over the same provider.",
    "EmailProviderHTTP": "Zitadel sends through one of these, so two namespaced copies would\n# be fighting over the same provider.",
    "SMSProviderTwilio": "Zitadel sends through one of these, so two namespaced copies would\n# be fighting over the same provider.",
    "SMSProviderHTTP": "Zitadel sends through one of these, so two namespaced copies would\n# be fighting over the same provider.",
}

DEFAULT_WHY = "It is the instance default, so it exists once however it is scoped."


def find_example(examples, kind):
    """The namespaced example for a kind.

    The files are not all named the same way - some are snake_cased and some keep
    the kind's own capitals - so the kind is read out of each file rather than
    guessed at from its name.
    """
    for f in sorted(os.listdir(examples)):
        if not f.endswith(".yaml"):
            continue
        text = open(os.path.join(examples, f)).read()
        if re.search(r"^kind: %s$" % re.escape(kind), text, re.M):
            return os.path.join(examples, f)

    return None


if __name__ == "__main__":
    examples, out_dir = sys.argv[1], sys.argv[2]
    made = []
    for kind in MANAGES:
        src = find_example(examples, kind)
        if src is None:
            print("no example for", kind)
            continue
        dst = os.path.join(out_dir, "cluster_" + snake(kind) + ".yaml")
        with open(dst, "w") as f:
            f.write(transform(open(src).read(), kind,
                              MANAGES[kind], WHY.get(kind, DEFAULT_WHY)))
        made.append(dst)
    print("wrote %d examples" % len(made))

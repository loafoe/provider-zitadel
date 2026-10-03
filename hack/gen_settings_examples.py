#!/usr/bin/env python3
"""Generate one example per instance wide settings managed resource.

Each example is a manifest that could be applied as it stands, so the values in
it are the ones Zitadel actually accepts rather than placeholders that only
look right.
"""

import sys

LICENSE_NOTE = """# An example of a %s.
#
# Apply it with:
#
#     kubectl apply -f %s
#
# This manages state that belongs to the whole Zitadel instance, so two of these
# in different namespaces would be asking for the same settings and the second
# would win. Only one at a time.
"""

EXAMPLES = {
    "InstanceFeatures": (
        "Zitadel instance feature flags",
        "instance_features.yaml",
        """apiVersion: zitadel.m.crossplane.io/v1alpha1
kind: InstanceFeatures
metadata:
  name: zitadel-features
  namespace: zitadel
spec:
  providerConfigRef:
    name: zitadel
    kind: ProviderConfig
  forProvider:
    # The login screen falls back to the default organization rather than to the
    # organization's own settings when no organization context is set.
    loginDefaultOrg: true
    # The user schema API is early and may change between Zitadel releases.
    userSchema: false
    # Returns the underlying error to an OIDC client. It can reveal details about
    # the system, so it is off unless it is being used to debug something.
    debugOidcParentError: false
    # Execution paths Zitadel may shortcut in exchange for less work on write.
    improvedPerformance:
      - IMPROVED_PERFORMANCE_PROJECT
      - IMPROVED_PERFORMANCE_USER_GRANT
""",
    ),
    "SystemFeatures": (
        "Zitadel system feature flags",
        "system_features.yaml",
        """apiVersion: zitadel.m.crossplane.io/v1alpha1
kind: SystemFeatures
metadata:
  name: zitadel-system-features
  namespace: zitadel
spec:
  providerConfigRef:
    name: zitadel
    kind: ProviderConfig
  forProvider:
    loginDefaultOrg: true
    userSchema: false
""",
    ),
    "InstanceRestrictions": (
        "Zitadel instance registration restrictions",
        "instance_restrictions.yaml",
        """apiVersion: zitadel.m.crossplane.io/v1alpha1
kind: InstanceRestrictions
metadata:
  name: zitadel-restrictions
  namespace: zitadel
spec:
  providerConfigRef:
    name: zitadel
    kind: ProviderConfig
  forProvider:
    # Off by default. Turning it off lets anyone sign up their own organization
    # again, which is not usually what a manifest should say out loud.
    disallowPublicOrgRegistration: true
    # The languages the login screen offers. Empty means every language Zitadel
    # supports.
    allowedLanguages:
      - en
      - de
""",
    ),
    "InstanceSecretGenerator": (
        "Zitadel instance secret generator",
        "instance_secret_generator.yaml",
        """apiVersion: zitadel.m.crossplane.io/v1alpha1
kind: InstanceSecretGenerator
metadata:
  name: zitadel-init-code
  namespace: zitadel
spec:
  providerConfigRef:
    name: zitadel
    kind: ProviderConfig
  forProvider:
    # Which of Zitadel's codes this shapes. Each type is a generator of its own,
    # so a second one of these may manage a different type at the same time.
    generatorType: SECRET_GENERATOR_TYPE_INIT_CODE
    # Zitadel refuses a length below four.
    length: 6
    # How long the code stays usable, as a Go duration.
    expiry: 5m
    # The four flags together have to spell at least one character class.
    includeLowerLetters: true
    includeUpperLetters: false
    includeDigits: true
    includeSymbols: false
""",
    ),
}


# The three domain lists.
EXAMPLES["InstanceCustomDomain"] = (
    "Zitadel instance custom domain",
    "instance_custom_domain.yaml",
    """apiVersion: zitadel.m.crossplane.io/v1alpha1
kind: InstanceCustomDomain
metadata:
  name: zitadel-login-domain
  namespace: zitadel
spec:
  providerConfigRef:
    name: zitadel
    kind: ProviderConfig
  forProvider:
    # The domain Zitadel answers on. Zitadel also answers on one domain it
    # generated for itself, which is not managed here: it made that one, and it
    # will not remove it.
    domain: login.example.com
""",
)

EXAMPLES["InstanceTrustedDomain"] = (
    "Zitadel instance trusted domain",
    "instance_trusted_domain.yaml",
    """apiVersion: zitadel.m.crossplane.io/v1alpha1
kind: InstanceTrustedDomain
metadata:
  name: zitadel-trusted-domain
  namespace: zitadel
spec:
  providerConfigRef:
    name: zitadel
    kind: ProviderConfig
  forProvider:
    # The domain allowed to ask this Zitadel instance for one of its tokens,
    # which is how an application outside the instance authenticates a user
    # against it.
    domain: apps.example.com
""",
)

EXAMPLES["OrganizationDomain"] = (
    "Zitadel organization domain",
    "organization_domain.yaml",
    """apiVersion: zitadel.m.crossplane.io/v1alpha1
kind: OrganizationDomain
metadata:
  name: platform-domain
  namespace: zitadel
spec:
  providerConfigRef:
    name: zitadel
    kind: ProviderConfig
  forProvider:
    organizationRef:
      name: platform
    # The domain the organization owns. Zitadel adds it unverified.
    domain: example.com
    # DNS asks for a TXT record, HTTP for a file at a URL Zitadel gives.
    validationType: DOMAIN_VALIDATION_TYPE_DNS
    # Asking Zitadel to check is only half of it: the token and URL it reports in
    # the status have to be published first, and the domain becomes verified once
    # Zitadel sees them.
    verify: true
""",
)


if __name__ == "__main__":
    out_dir = sys.argv[1]
    for kind, (title, filename, body) in EXAMPLES.items():
        path = "%s/%s" % (out_dir, filename)
        with open(path, "w") as f:
            f.write(LICENSE_NOTE % (title, filename))
            f.write("#\n")
            f.write(body)
        print("wrote", path)

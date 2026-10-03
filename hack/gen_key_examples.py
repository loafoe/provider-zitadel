#!/usr/bin/env python3
"""Generate one example per key and membership managed resource.

Each example is a manifest that could be applied as it stands, so the values are
the ones Zitadel accepts rather than placeholders that only look right.
"""

import sys

HEADER = """# An example of a %s.
#
# Apply it with:
#
#     kubectl apply -f %s
"""

EXAMPLES = {
    "WebKey": (
        "Zitadel signing key",
        "web_key.yaml",
        """apiVersion: zitadel.m.crossplane.io/v1alpha1
kind: WebKey
metadata:
  name: zitadel-web-key
  namespace: zitadel
spec:
  providerConfigRef:
    name: zitadel
    kind: ProviderConfig
  forProvider:
    # rsa, ecdsa or ed25519. Zitadel generates the key; the private half never
    # leaves it, so there is no material to supply.
    algorithm: rsa
    # Zitadel's default is 2048. Ignored for the other algorithms.
    rsaBits: RSA_BITS_2048
    # Zitadel's default is SHA256. Ignored for the other algorithms.
    rsaHasher: RSA_HASHER_SHA256
    #
    # Zitadel creates a key inactive, so it verifies nothing until something
    # activates it. An ActivatedWebKey is what does that.
""",
    ),
    "ApplicationKey": (
        "Zitadel application key",
        "application_key.yaml",
        """apiVersion: zitadel.m.crossplane.io/v1alpha1
kind: ApplicationKey
metadata:
  name: platform-app-key
  namespace: zitadel
spec:
  providerConfigRef:
    name: zitadel
    kind: ProviderConfig
  writeConnectionSecretToRef:
    name: platform-app-key
  forProvider:
    organizationRef:
      name: platform
    projectRef:
      name: platform
    applicationRef:
      name: platform-oidc
    # When the key stops working, as a duration from now. Empty means it does
    # not expire.
    expirationDate: 8760h
    #
    # The private half is written to the connection secret as key.json, because
    # Zitadel returns it exactly once: here. There is no way to read it again,
    # so a key whose secret was lost can only be replaced.
""",
    ),
    "ProjectMember": (
        "Zitadel project member",
        "project_member.yaml",
        """apiVersion: zitadel.m.crossplane.io/v1alpha1
kind: ProjectMember
metadata:
  name: platform-owner
  namespace: zitadel
spec:
  providerConfigRef:
    name: zitadel
    kind: ProviderConfig
  forProvider:
    organizationRef:
      name: platform
    projectRef:
      name: platform
    userRef:
      name: platform-owner
    # The role keys are not fixed: a project may define roles of its own, so
    # they are read from the project rather than assumed.
    roles:
      - PROJECT_OWNER
    #
    # Zitadel refuses a member with no role at all, so at least one is required.
    # To take every role away, delete this resource rather than emptying the list.
""",
    ),
}

if __name__ == "__main__":
    out_dir = sys.argv[1]
    for title, filename, body in EXAMPLES.values():
        path = "%s/%s" % (out_dir, filename)
        with open(path, "w") as f:
            f.write(HEADER % (title, filename))
            f.write("#\n")
            f.write(body)
        print("wrote", path)

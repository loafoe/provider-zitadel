#!/usr/bin/env python3
"""Generate one example per SAML application, active key and messaging provider.

Each example is a manifest that could be applied as it stands.
"""

import sys

HEADER = """# An example of a %s.
#
# Apply it with:
#
#     kubectl apply -f %s
"""

EXAMPLES = {
    "ApplicationSAML": (
        "Zitadel SAML application",
        "application_saml.yaml",
        """apiVersion: zitadel.m.crossplane.io/v1alpha1
kind: ApplicationSAML
metadata:
  name: billing-saml
  namespace: zitadel
spec:
  providerConfigRef:
    name: zitadel
    kind: ProviderConfig
  forProvider:
    organizationRef:
      name: platform
    projectRef:
      name: billing
    name: Billing SAML
    # Exactly one of metadata and metadataURL: Zitadel requires one or the other.
    #
    # The document itself always works, because Zitadel has to fetch nothing. A
    # URL is only worth using when Zitadel can reach it.
    metadata: |
      <?xml version="1.0"?>
      <EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata"
                        entityID="https://idp.example.com"/>
    # Which login screen the user is sent to. Empty leaves the instance default.
    loginVersion: "LoginV1"
""",
    ),
    "ActiveWebKey": (
        "Zitadel active signing key",
        "active_web_key.yaml",
        """apiVersion: zitadel.m.crossplane.io/v1alpha1
kind: ActiveWebKey
metadata:
  name: zitadel-active-key
  namespace: zitadel
spec:
  providerConfigRef:
    name: zitadel
    kind: ProviderConfig
  forProvider:
    webKeyRef:
      name: zitadel-web-key
    #
    # There is one active key on an instance, so two of these in different
    # namespaces would be fighting over the same setting. Activating another key
    # makes this one stale and the former inactive.
    #
    # Zitadel will not remove a key that is active, so a key this points at has
    # to be activated away before it can be deleted.
""",
    ),
    "EmailProviderSMTP": (
        "Zitadel SMTP email provider",
        "email_provider_smtp.yaml",
        """apiVersion: zitadel.m.crossplane.io/v1alpha1
kind: EmailProviderSMTP
metadata:
  name: zitadel-smtp
  namespace: zitadel
spec:
  providerConfigRef:
    name: zitadel
    kind: ProviderConfig
  forProvider:
    # host:port. Zitadel does not connect when the provider is added, only when
    # something is sent, so this can be set up ahead of a server that is not
    # reachable yet.
    host: "smtp.example.com:587"
    user: "zitadel@example.com"
    # The password is a secret rather than a field, so it is not readable by
    # anyone who can read this object. Zitadel never returns it, so it is applied
    # and then left alone.
    passwordSecretRef:
      name: zitadel-smtp
      key: password
    senderAddress: "noreply@example.com"
    senderName: "Example"
    # Zitadel calls this tls rather than starttls: it means the connection is
    # wrapped from the start, which is what port 465 expects.
    tls: true
    description: "The instance wide email provider"
""",
    ),
    "EmailProviderHTTP": (
        "Zitadel HTTP email provider",
        "email_provider_http.yaml",
        """apiVersion: zitadel.m.crossplane.io/v1alpha1
kind: EmailProviderHTTP
metadata:
  name: zitadel-email-http
  namespace: zitadel
spec:
  providerConfigRef:
    name: zitadel
    kind: ProviderConfig
  forProvider:
    # Receives the same payload Zitadel would hand an SMTP server, so the
    # endpoint has to understand it.
    endpoint: "https://email.example.com/send"
    description: "The instance wide email provider"
""",
    ),
    "SMSProviderTwilio": (
        "Zitadel Twilio SMS provider",
        "sms_provider_twilio.yaml",
        """apiVersion: zitadel.m.crossplane.io/v1alpha1
kind: SMSProviderTwilio
metadata:
  name: zitadel-twilio
  namespace: zitadel
spec:
  providerConfigRef:
    name: zitadel
    kind: ProviderConfig
  forProvider:
    sid: "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
    # The token is a secret, and Zitadel sets it through a call of its own and
    # never returns it.
    tokenSecretRef:
      name: zitadel-twilio
      key: token
    senderNumber: "+441234567890"
    description: "The instance wide SMS provider"
""",
    ),
    "SMSProviderHTTP": (
        "Zitadel HTTP SMS provider",
        "sms_provider_http.yaml",
        """apiVersion: zitadel.m.crossplane.io/v1alpha1
kind: SMSProviderHTTP
metadata:
  name: zitadel-sms-http
  namespace: zitadel
spec:
  providerConfigRef:
    name: zitadel
    kind: ProviderConfig
  forProvider:
    # Receives the same payload Zitadel would hand an SMS gateway, so the
    # endpoint has to understand it.
    endpoint: "https://sms.example.com/send"
    description: "The instance wide SMS provider"
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

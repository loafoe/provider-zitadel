#!/usr/bin/env python3
"""Generate one example manifest per Zitadel policy managed resource.

The examples are the first thing a reader copies, so they are generated from the
same table as the API types and the controllers: a policy's example cannot then
drift away from the fields its kind actually accepts.
"""

import sys

sys.path.insert(0, __file__.rsplit("/", 1)[0])
from gen_policy_types import INSTANCE_POLICIES, POLICIES  # noqa: E402

ORG_HEADER = """# {purpose}.
#
# This is a singleton: one resource describes the policy of one organization,
# and deleting it puts the organization back on the instance default rather than
# removing anything.
#
# Every field is optional. A field left unset is not compared against Zitadel,
# so its default is never fought over.
apiVersion: zitadel.m.crossplane.io/v1alpha1
kind: {kind}
metadata:
  name: {name}
  namespace: zitadel
spec:
  providerConfigRef: {{name: zitadel, kind: ProviderConfig}}
  forProvider:
    organizationRef: {{name: platform}}"""

INSTANCE_HEADER = """# {purpose}.
#
# This is an instance wide setting: there is one of each for the whole instance,
# and every organization inherits it until it sets its own.
#
# Zitadel has no way to reset an instance wide policy - there is no default
# beneath it - so deleting this resource does not remove anything. It writes back
# whatever the policy was before this resource first changed it, which is
# recorded in status.atProvider.restore.
#
# Every field is optional. A field left unset is not compared against Zitadel,
# so its default is never fought over.
apiVersion: zitadel.m.crossplane.io/v1alpha1
kind: {kind}
metadata:
  name: {name}
  namespace: zitadel
spec:
  providerConfigRef: {{name: zitadel, kind: ProviderConfig}}
  forProvider:"""

# kind -> (example name, the fields worth showing, purpose)
# The instance wide policies show the settings worth setting once for the whole
# instance. Their names drop the "Default" prefix, because that is the only
# instance wide policy of that kind.
INSTANCE_SHOWCASE = {
    "DefaultLockoutPolicy": ("instance-lockout", [("maxPasswordAttempts", "5"), ("maxOTPAttempts", "5")],
                             "Lock out a user of any organization after too many failed attempts"),
    "DefaultNotificationPolicy": ("instance-notifications", [("passwordChange", "true")],
                                  "Choose which events make Zitadel send an email, anywhere in the instance"),
    "DefaultPasswordAgePolicy": ("instance-password-expiry", [("maxAgeDays", "365"), ("expireWarnDays", "14")],
                                 "Make passwords expire everywhere in the instance"),
    "DefaultPasswordComplexityPolicy": ("instance-password-complexity",
                                        [("minLength", "12"), ("hasUppercase", "true"), ("hasLowercase", "true"),
                                         ("hasNumber", "true"), ("hasSymbol", "true")],
                                        "Set what a new password must contain, instance wide"),
    "DefaultPrivacyPolicy": ("instance-privacy",
                             [("tosLink", "https://example.com/tos"), ("privacyLink", "https://example.com/privacy"),
                              ("helpLink", "https://example.com/help"), ("supportEmail", "support@example.com")],
                             "Set the legal and support links an organization inherits"),
    "DefaultDomainPolicy": ("instance-domains",
                            [("userLoginMustBeDomain", "false"), ("validateOrgDomains", "false")],
                            "Control how login names relate to domains, instance wide"),
    "DefaultLabelPolicy": ("instance-branding",
                           [("primaryColor", "#5282C1"), ("warnColor", "#F4A300"),
                            ("backgroundColor", "#FFFFFF"), ("fontColor", "#1A1A1A"),
                            ("hideLoginNameSuffix", "false"), ("disableWatermark", "true"),
                            ("themeMode", "Auto")],
                           "Set the colours and theme an organization inherits for its login pages"),
    "DefaultLoginPolicy": ("instance-login",
                           [("allowUsernamePassword", "true"), ("allowRegister", "false"),
                            ("ignoreUnknownUsernames", "true"), ("passwordlessType", "NotAllowed"),
                            ("passwordCheckLifetime", "24h"), ("secondFactors", ["OTP", "OTPEmail"])],
                           "Set what an organization allows until it sets its own login policy"),
    "DefaultOIDCSettings": ("instance-oidc-settings",
                            [("accessTokenLifetime", "1h"), ("idTokenLifetime", "1h"),
                             ("refreshTokenExpiration", "720h"), ("refreshTokenIdleExpiration", "56h")],
                            "Set how long the tokens Zitadel issues stay valid"),
    "DefaultSecuritySettings": ("instance-security-settings",
                                [("enableImpersonation", "false"), ("embeddedIframe", "false")],
                                "Control impersonation and whether Zitadel may be embedded in a frame"),
}

SHOWCASE = {
    "LockoutPolicy": (
        "platform-lockout",
        [("maxPasswordAttempts", "5"), ("maxOTPAttempts", "5")],
        "Lock out a user after too many failed attempts",
    ),
    "NotificationPolicy": (
        "platform-notifications",
        [("passwordChange", "true")],
        "Choose which events make Zitadel send an email",
    ),
    "PasswordAgePolicy": (
        "platform-password-expiry",
        [("maxAgeDays", "365"), ("expireWarnDays", "14")],
        "Make passwords expire, and warn users before they do",
    ),
    "PasswordComplexityPolicy": (
        "platform-password-complexity",
        [("minLength", "12"), ("hasUppercase", "true"), ("hasLowercase", "true"),
         ("hasNumber", "true"), ("hasSymbol", "true")],
        "Set what a new password has to contain",
    ),
    "PrivacyPolicy": (
        "platform-privacy",
        [("tosLink", "https://example.com/tos"), ("privacyLink", "https://example.com/privacy"),
         ("helpLink", "https://example.com/help"), ("supportEmail", "support@example.com"),
         ("docsLink", "https://example.com/docs"), ("customLink", "https://example.com/status"),
         ("customLinkText", "Status")],
        "Set the legal and support links Zitadel shows",
    ),
    "DomainPolicy": (
        "platform-domains",
        [("userLoginMustBeDomain", "false"), ("validateOrgDomains", "false"),
         ("smtpSenderAddressMatchesInstanceDomain", "true")],
        "Control how login names relate to an organization's domains",
    ),
    "LabelPolicy": (
        "platform-branding",
        [("primaryColor", "#5282C1"), ("warnColor", "#F4A300"),
         ("backgroundColor", "#FFFFFF"), ("fontColor", "#1A1A1A"),
         ("primaryColorDark", "#3C6EA5"), ("warnColorDark", "#D18F00"),
         ("backgroundColorDark", "#121212"), ("fontColorDark", "#F5F5F5"),
         ("hideLoginNameSuffix", "false"), ("disableWatermark", "true"),
         ("themeMode", "Auto")],
        "Set the colours and theme of an organization's login pages",
    ),
}


def render(p):
    kind = p["kind"]
    if kind.startswith("Default"):
        name, fields, purpose = INSTANCE_SHOWCASE[kind]
    else:
        name, fields, purpose = SHOWCASE[kind]

    lines = [(ORG_HEADER if not kind.startswith("Default") else INSTANCE_HEADER)
             .format(kind=kind, name=name, purpose=purpose)]

    def literal(value):
        if isinstance(value, list):
            return "[%s]" % ", ".join(literal(v) for v in value)

        # Quote anything YAML would otherwise read as something else. The trap
        # here is a hex colour: "#5282C1" unquoted starts a comment, so the
        # field arrives empty and the example looks fine until it is applied.
        if value in ("true", "false", "null", "~"):
            return value

        if value.replace(".", "", 1).isdigit():
            return value

        return '"%s"' % value

    for json_name, value in fields:
        lines.append("    %s: %s" % (json_name, literal(value)))

    if kind.endswith("LabelPolicy"):
        lines.append("")
        lines.append("# The logo, favicon and custom font are uploaded to Zitadel outside its")
        lines.append("# API, so this provider reports where they are served from but never")
        lines.append("# sets or removes them. Their URLs appear in status.atProvider.")

    lines.append("")
    return "\n".join(lines)


if __name__ == "__main__":
    out_dir = sys.argv[1]
    for p in POLICIES + INSTANCE_POLICIES:
        path = "%s/%s.yaml" % (out_dir, p["kind"].lower())
        with open(path, "w") as f:
            f.write(render(p))
        print("wrote", path)

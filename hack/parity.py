#!/usr/bin/env python3
"""Compare this provider's managed resources against the Terraform provider.

Parity is claimed capability by capability: every resource the Terraform
provider registers is either modelled here, by naming the kind or kinds that
cover it, or deliberately not, with the reason. Nothing is estimated, nothing is
counted twice, and a resource that appears in neither is an error rather than a
number quietly drifting.

The Terraform provider is read from its own source rather than from a list kept
here, so the figures cannot drift away from it:

    git clone --depth 1 https://github.com/zitadel/terraform-provider-zitadel
    ./hack/parity.py /path/to/terraform-provider-zitadel

Exits non-zero while anything worth modelling is still missing, so it can be used
as a gate rather than a report.
"""

import os
import re
import sys

# A resource is covered by one or more of our kinds. Where two kinds cover one
# resource, it is because Zitadel's v2 application API is split by application
# type where the Terraform provider takes it in one resource.
MODELLED = {
    "action": "Action",
    "action_execution_event": "ActionExecutionEvent",
    "action_execution_function": "ActionExecutionFunction",
    "action_execution_request": "ActionExecutionRequest",
    "action_execution_response": "ActionExecutionResponse",
    "action_target": "ActionTarget",
    "action_target_public_key": "ActionTargetPublicKey",
    "application_v2": "ApplicationAPI, OIDCApplication",
    "default_domain_policy": "DefaultDomainPolicy",
    "default_label_policy": "DefaultLabelPolicy",
    "default_lockout_policy": "DefaultLockoutPolicy",
    "default_login_policy": "DefaultLoginPolicy",
    "default_notification_policy": "DefaultNotificationPolicy",
    "default_oidc_settings": "DefaultOIDCSettings",
    "default_password_age_policy": "DefaultPasswordAgePolicy",
    "default_password_complexity_policy": "DefaultPasswordComplexityPolicy",
    "default_privacy_policy": "DefaultPrivacyPolicy",
    "default_security_settings": "DefaultSecuritySettings",
    "domain_policy": "DomainPolicy",
    "human_user": "HumanUser",
    "idp_apple": "IDPApple",
    "idp_azure_ad": "IDPAzureAD",
    "idp_github": "IDPGitHub",
    "idp_github_es": "IDPGitHubEnterpriseServer",
    "idp_gitlab": "IDPGitLab",
    "idp_gitlab_self_hosted": "IDPGitLabSelfHosted",
    "idp_google": "IDPGoogle",
    "idp_ldap": "IDPLDAP",
    "idp_oauth": "IDPOAuth",
    "idp_oidc": "IDPOIDC",
    "idp_saml": "IDPSAML",
    "instance_custom_domain": "InstanceCustomDomain",
    "instance_features": "InstanceFeatures",
    "instance_member": "InstanceMember",
    "instance_restrictions": "InstanceRestrictions",
    "instance_secret_generator": "InstanceSecretGenerator",
    "instance_trusted_domain": "InstanceTrustedDomain",
    "label_policy": "LabelPolicy",
    "lockout_policy": "LockoutPolicy",
    "login_policy": "LoginPolicy",
    "machine_key": "MachineKey",
    "machine_user": "ServiceAccount",
    "notification_policy": "NotificationPolicy",
    "org_idp_apple": "OrgIDPApple",
    "org_idp_azure_ad": "OrgIDPAzureAD",
    "org_idp_github": "OrgIDPGitHub",
    "org_idp_github_es": "OrgIDPGitHubEnterpriseServer",
    "org_idp_gitlab": "OrgIDPGitLab",
    "org_idp_gitlab_self_hosted": "OrgIDPGitLabSelfHosted",
    "org_idp_google": "OrgIDPGoogle",
    "org_idp_jwt": "OrgIDPJWT",
    "org_idp_ldap": "OrgIDPLDAP",
    "org_idp_oauth": "OrgIDPOAuth",
    "org_idp_oidc": "OrgIDPOIDC",
    "org_idp_saml": "OrgIDPSAML",
    "org_member": "OrgMember",
    "organization": "Organization",
    "organization_domain": "OrganizationDomain",
    "organization_metadata": "OrganizationMetadata",
    "password_age_policy": "PasswordAgePolicy",
    "password_complexity_policy": "PasswordComplexityPolicy",
    "personal_access_token": "PersonalAccessToken",
    "privacy_policy": "PrivacyPolicy",
    "project_grant": "ProjectGrant",
    "project_grant_member": "ProjectGrantMember",
    "project_role": "ProjectRole",
    "project_v2": "Project",
    "system_features": "SystemFeatures",
    "trigger_actions": "TriggerActions",
    "user_grant": "UserGrant",
    "user_metadata": "UserMetadata",
}

# Resources deliberately not modelled, each with the reason.
EXCLUDED = {
    "application_api": "The v1 application API. Superseded by the v2 application API, which is what this provider uses.",
    "application_oidc": "The v1 application API. Superseded by the v2 application API, which is what this provider uses.",
    "default_hosted_login_translation": "A set of static translation strings rather than anything an operator manages in a cluster.",
    "domain": "Deprecated by the Terraform provider in favour of organization_domain.",
    "hosted_login_translation": "A set of static translation strings rather than anything an operator manages in a cluster.",
    "org": "The v1 organization API. Superseded by organization, the v2 API this provider uses.",
    "org_metadata": "Deprecated by the Terraform provider in favour of organization_metadata, which is modelled.",
    "project": "The v1 project API. Superseded by project_v2, which is what this provider uses.",
    "smtp_config": "Deprecated by the Terraform provider in favour of email_provider_smtp.",
}

# The provider registers each resource in a map pointing at the package that
# implements it.
RESOURCE_MAP = re.compile(r'"(zitadel_[a-z0-9_]+)"\s*:\s*([a-z0-9_]+)\.GetResource\(\)')
IMPORT = re.compile(r'^\s*([a-z0-9_]+)\s+"github.com/zitadel/terraform-provider-zitadel/zitadel/([a-z0-9_/]+)"', re.M)


def our_kinds(crds):
    """Every managed kind, read from the CRDs rather than from a list."""
    out = []
    for f in sorted(os.listdir(crds)):
        if not f.startswith("zitadel.m.crossplane.io_"):
            continue
        m = re.search(r"^\s+kind:\s*(\S+)\s*$", open(os.path.join(crds, f)).read(), re.M)
        if m:
            out.append(m.group(1))
    return out


def their_resources(root):
    """Every resource the Terraform provider registers, read from its source."""
    provider = open(os.path.join(root, "zitadel", "provider.go")).read()
    return {n.replace("zitadel_", "") for n, _ in RESOURCE_MAP.findall(provider)}


def main():
    if len(sys.argv) < 2:
        print(__doc__)
        return 2

    theirs = their_resources(sys.argv[1])
    ours = our_kinds(os.path.join(os.path.dirname(__file__), "..", "package", "crds"))

    # Every kind this table claims must exist, or the table is describing a
    # provider that no longer has it.
    claimed = {k.strip() for v in MODELLED.values() for k in v.split(",")}
    missing_kinds = sorted(claimed - set(ours))
    # And every kind that exists must be in the table, or a new resource would
    # silently not count.
    unclaimed = sorted(set(ours) - claimed)

    accounted = set(MODELLED) | set(EXCLUDED)
    unknown = sorted(accounted - theirs)
    wanted = sorted(theirs - accounted)

    print("Terraform provider resources  : %d" % len(theirs))
    print("Managed resource kinds here   : %d" % len(ours))
    print("Capabilities modelled         : %d" % len(MODELLED))
    print("Deliberately not modelled     : %d" % len(EXCLUDED))
    print()
    print("Raw parity      : %d/%d = %d%%" % (len(MODELLED), len(theirs), round(100 * len(MODELLED) / len(theirs))))
    print("Adjusted parity : %d/%d = %d%%" % (len(MODELLED), len(theirs) - len(EXCLUDED),
                                              round(100 * len(MODELLED) / (len(theirs) - len(EXCLUDED)))))

    if wanted:
        print()
        print("NOT YET MODELLED (%d):" % len(wanted))
        for n in wanted:
            print("   ", n)
    if unknown:
        print()
        print("In the table but not in the Terraform provider (%d):" % len(unknown))
        for n in unknown:
            print("   ", n)
    if missing_kinds:
        print()
        print("Claimed but no longer a kind here (%d): %s" % (len(missing_kinds), ", ".join(missing_kinds)))
    if unclaimed:
        print()
        print("A kind here that the table does not mention (%d): %s" % (len(unclaimed), ", ".join(unclaimed)))

    if wanted or unknown or missing_kinds or unclaimed:
        return 1

    print()
    print("Every resource worth modelling is modelled.")
    return 0


if __name__ == "__main__":
    sys.exit(main())

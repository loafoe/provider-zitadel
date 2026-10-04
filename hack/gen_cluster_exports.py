#!/usr/bin/env python3
"""Generate the exported client factories the cluster scoped setups need.

A cluster scoped kind is reconciled by the controller that already reconciles the
namespaced one, so it needs two things that package does not otherwise expose:
the driver's type parameters, which are known only inside it, and a name for the
factory the namespaced Setup builds.

Rather than exporting the driver itself, each package exposes one function that
returns the client factory. The cluster scoped setup then names a function rather
than repeating the type parameters, which is the part that could drift.
"""

import os
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

# The packages that own a driver the cluster scoped setups need, and the kind
# each driver reconciles.
OWNERS = {
    "default_lockout_policy": "DefaultLockoutPolicy",
    "default_notification_policy": "DefaultNotificationPolicy",
    "default_password_age_policy": "DefaultPasswordAgePolicy",
    "default_password_complexity_policy": "DefaultPasswordComplexityPolicy",
    "default_privacy_policy": "DefaultPrivacyPolicy",
    "default_domain_policy": "DefaultDomainPolicy",
    "default_label_policy": "DefaultLabelPolicy",
    "default_login_policy": "DefaultLoginPolicy",
    "default_oidc_settings": "DefaultOIDCSettings",
    "default_security_settings": "DefaultSecuritySettings",
}

# instancesettings owns four drivers, so its kinds are named rather than derived.
INSTANCESETTINGS = [
    ("InstanceFeatures", "featuresDriver{instance: true}"),
    ("SystemFeatures", "featuresDriver{}"),
    ("InstanceRestrictions", "restrictionsDriver{}"),
    ("InstanceSecretGenerator", "secretGeneratorDriver{}"),
]

DRIVER_ASSERTION = re.compile(
    r"common\.PolicyDriver\[([^,\]]+),\s*([^\]]+)\]\s*=\s*([^\s]+)")


def driver_types(root, pkg, literal):
    """The driver's own type parameters, read from the package that declares it.

    They are read rather than written here because a guess that compiles is still
    a guess, and the client it builds would be for a slightly different policy.
    """
    src = ""
    for f in sorted(os.listdir(os.path.join(root, pkg))):
        if f.endswith(".go"):
            src += open(os.path.join(root, pkg, f)).read()

    # The driver expression is compared up to its brace, because the literal it
    # is written as carries fields and the assertion does not.
    want = literal.split("{")[0]

    for p, o, drv in DRIVER_ASSERTION.findall(src):
        if drv.split("{")[0] == want:
            return p.strip(), o.strip()

    raise SystemExit("no driver assertion found in %s for %s" % (pkg, literal))


def client_name(kind):
    return kind + "Client"


POLICY_FACTORY = '''
// %(fn)s returns the external client factory for this package's driver, which is
// how %(kind)s is reconciled.
//
// The cluster scoped %(cluster)s is set up from this, so both are reconciled by
// one client rather than by two that could drift apart.
func %(fn)s() common.NewExternalClientFn {
	return common.PolicyExternalFactory[%(p)s, %(o)s, *v1alpha1.%(cr)s](%(d)s)
}
'''


def render(pkg, entries):  # noqa: PLR0913 - the four are one row of a table
    imports = [
        '\tv1alpha1 "github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"\n',
        '\t"github.com/loafoe/provider-zitadel/internal/clients/zitadel"\n',
        '\t"github.com/loafoe/provider-zitadel/internal/controller/common"\n',
    ]

    body = LICENSE + "\npackage %s\n\nimport (\n" % pkg + "".join(imports) + ")\n"

    for cr, p, o, d in entries:
        body += POLICY_FACTORY % {
            "fn": client_name(cr),
            "p": p,
            "o": o,
            "cr": cr,
            "d": d,
            "kind": cr,
            "cluster": "Cluster" + cr,
        }

    return body


if __name__ == "__main__":
    root = sys.argv[1]

    work = {}
    for pkg, kind in OWNERS.items():
        src = "".join(open(os.path.join(root, pkg, f)).read()
                      for f in sorted(os.listdir(os.path.join(root, pkg))) if f.endswith(".go"))
        m = DRIVER_ASSERTION.search(src)
        if not m:
            raise SystemExit("no driver assertion found in " + pkg)
        work[pkg] = [(kind, m.group(1).strip(), m.group(2).strip(), "driver{}")]

    work["instancesettings"] = []
    for kind, literal in INSTANCESETTINGS:
        p, o = driver_types(root, "instancesettings", literal)
        work["instancesettings"].append((kind, p, o, literal))

    for pkg, entries in work.items():
        path = os.path.join(root, pkg, "client.go")
        with open(path, "w") as f:
            f.write(render(pkg, entries))
        print("wrote", path)

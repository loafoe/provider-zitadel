#!/usr/bin/env python3
"""Generate the cluster scoped controllers.

Nineteen of the kinds manage state belonging to a whole Zitadel instance, and each
is a singleton. The cluster scoped form of each is reconciled by the very
controller that already reconciles the namespaced one, so what is generated here
is a setup call and nothing else: no driver, no field list, and no copy of the
logic that already works.
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

PACKAGE_DOC = """// Package cluster holds the cluster scoped forms of the instance wide managed
// resources.
//
// Each kind here manages state belonging to a whole Zitadel instance and is a
// singleton: one set of instance feature flags, one login policy default, one
// SMTP provider. Namespaced, two namespaces could each hold one and fight over
// the same settings, which the second writer wins. Cluster scoped, that conflict
// cannot be expressed.
//
// None of this is a second implementation. Every one of these kinds already has a
// controller that reconciles it, and that controller is reused unchanged: the
// cluster scoped resource is handed to it as the namespaced kind it mirrors, and
// the external name, finalizers and status it produces are copied back. The
// namespaced kinds are all kept, and all still work.
package cluster
"""

# (namespaced kind, package that reconciles it, that package's client factory).
KINDS = [
    ("InstanceFeatures", "instancesettings", "InstanceFeaturesClient"),
    ("SystemFeatures", "instancesettings", "SystemFeaturesClient"),
    ("InstanceRestrictions", "instancesettings", "InstanceRestrictionsClient"),
    ("InstanceSecretGenerator", "instancesettings", "InstanceSecretGeneratorClient"),
    ("ActiveWebKey", "messaging", "NewActiveWebKeyExternal"),
    ("EmailProviderSMTP", "messaging", "SMTPProviderExternal"),
    ("EmailProviderHTTP", "messaging", "EmailHTTPProviderExternal"),
    ("SMSProviderTwilio", "messaging", "TwilioProviderExternal"),
    ("SMSProviderHTTP", "messaging", "SMSHTTPProviderExternal"),
]

# The ten instance wide policy defaults, each in its own package.
POLICY_KINDS = [
    "DefaultLockoutPolicy",
    "DefaultNotificationPolicy",
    "DefaultPasswordAgePolicy",
    "DefaultPasswordComplexityPolicy",
    "DefaultPrivacyPolicy",
    "DefaultDomainPolicy",
    "DefaultLabelPolicy",
    "DefaultLoginPolicy",
    "DefaultOIDCSettings",
    "DefaultSecuritySettings",
]


def snake(name):
    s = re.sub(r"(.)([A-Z][a-z]+)", r"\1_\2", name)
    return re.sub(r"([a-z0-9])([A-Z])", r"\1_\2", s).lower()


def factory(pkg, driver):
    """The client factory the owning package exports for this kind.

    Every package that owns a driver the cluster scoped kinds need exposes one
    factory, so the setup names a function rather than repeating the driver's
    type parameters - which are known only inside the package that declares them,
    and are the part that could drift.
    """
    # The active key already has a factory of the client factory shape; the rest
    # are functions that return one.
    if driver == "NewActiveWebKeyExternal":
        return "%s.%s" % (pkg, driver)

    return "%s.%s()" % (pkg, driver)


def setup(base, pkg, driver):
    kind = "Cluster" + base
    return f'''
// Setup{kind} registers the {kind} controller.
//
// It is the same controller that reconciles {base}, wrapped so that it is handed
// this kind instead. Nothing about how {base} is reconciled is repeated here.
func Setup{kind}(mgr ctrl.Manager, o controller.Options) error {{
	return common.SetupManagedResourceController(
		mgr, o,
		apisv1alpha1.{kind}GroupKind,
		apisv1alpha1.{kind}GroupVersionKind,
		&apisv1alpha1.{kind}{{}},
		&apisv1alpha1.{kind}List{{}},
		common.ReusingForCluster[*zitadelv1alpha1.{base}]({factory(pkg, driver)}),
	)
}}
'''


def render():
    entries = [(b, p, d) for b, p, d in KINDS]
    entries += [(b, snake(b), b + "Client") for b in POLICY_KINDS]

    pkgs = sorted({p for _, p, _ in entries})
    imports = [
        '\t"github.com/crossplane/crossplane-runtime/v2/pkg/controller"\n',
        '\n\tctrl "sigs.k8s.io/controller-runtime"\n',
        '\n\tapisv1alpha1 "github.com/loafoe/provider-zitadel/apis/v1alpha1"\n',
        '\tzitadelv1alpha1 "github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"\n',
        '\t"github.com/loafoe/provider-zitadel/internal/controller/common"\n',
    ]
    imports += ['\t"github.com/loafoe/provider-zitadel/internal/controller/%s"\n' % p for p in pkgs]

    out = [LICENSE, PACKAGE_DOC, "\nimport (\n", "".join(imports), ")\n"]

    out.append("\n// Setup registers the cluster scoped controllers.")
    out.append("func Setup(mgr ctrl.Manager, o controller.Options) error {")
    out.append("\tfor _, s := range []func(ctrl.Manager, controller.Options) error{")
    for base, _, _ in entries:
        out.append("\t\tSetupCluster%s," % base)
    out.append("\t} {")
    out.append("\t\tif err := s(mgr, o); err != nil {")
    out.append("\t\t\treturn err")
    out.append("\t\t}")
    out.append("\t}\n")
    out.append("\treturn nil")
    out.append("}")

    for base, pkg, driver in entries:
        out.append(setup(base, pkg, driver))

    return "\n".join(out)


if __name__ == "__main__":
    path = "%s/cluster.go" % sys.argv[1]
    with open(path, "w") as f:
        f.write(render())
    print("wrote", path)

#!/usr/bin/env python3
"""Generate the controllers of the Zitadel policy managed resources.

The lifecycle of every Zitadel policy is shared and lives in
internal/controller/common/policy.go. What is left for each kind is a driver:
where the policy applies, how it is read, how it is written, how it is reset,
and how the desired and observed shapes are projected onto the spec and the
status.

Those drivers are the same shape for all of them and differ only in field names,
so they are generated from the same table the API types are generated from. That
keeps a policy's spec, its status and its controller from drifting apart.
"""

import re
import sys

sys.path.insert(0, __file__.rsplit("/", 1)[0])
from gen_policy_types import POLICIES  # noqa: E402

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

# kind -> (client type, input type, getter, setter, resetter, reset docs)
CLIENT = {
    "LockoutPolicy": ("LockoutPolicy", "LockoutPolicyInput",
                      "GetLockoutPolicy", "SetLockoutPolicy", "ResetLockoutPolicy"),
    "NotificationPolicy": ("NotificationPolicy", "NotificationPolicyInput",
                           "GetNotificationPolicy", "SetNotificationPolicy", "ResetNotificationPolicy"),
    "PasswordAgePolicy": ("PasswordAgePolicy", "PasswordAgePolicyInput",
                          "GetPasswordAgePolicy", "SetPasswordAgePolicy", "ResetPasswordAgePolicy"),
    "PasswordComplexityPolicy": ("PasswordComplexityPolicy", "PasswordComplexityPolicyInput",
                                 "GetPasswordComplexityPolicy", "SetPasswordComplexityPolicy",
                                 "ResetPasswordComplexityPolicy"),
    "PrivacyPolicy": ("PrivacyPolicy", "PrivacyPolicyInput",
                      "GetPrivacyPolicy", "SetPrivacyPolicy", "ResetPrivacyPolicy"),
    "DomainPolicy": ("DomainPolicy", "DomainPolicyInput",
                     "GetDomainPolicy", "SetDomainPolicy", "ResetDomainPolicy"),
    "LabelPolicy": ("LabelPolicy", "LabelPolicyInput",
                    "GetLabelPolicy", "SetLabelPolicy", "ResetLabelPolicy"),
}

# Field names whose spec type is not the client field's own type, and so need an
# explicit conversion in each direction.
NUMERIC = {"MaxPasswordAttempts", "MaxOTPAttempts", "MaxAgeDays",
           "ExpireWarnDays", "MinLength"}


def snake(name):
    return re.sub(r"(?<!^)(?=[A-Z])", "_", name).lower()


def render(p):
    kind = p["kind"]
    observed, inp, get, set_, reset = CLIENT[kind]

    # Only the policy's own fields appear in both the spec and the status; the
    # organization reference does not.
    fields = [f for f in p["fields"]]

    def spec_expr(f):
        n = f["name"]
        if n.endswith("URL"):
            return '""'
        if n in NUMERIC:
            return "common.ToUint32(common.DerefInt64(fp.%s))" % n
        if f["type"] == "*int64":
            return "common.DerefInt64(fp.%s)" % n
        if f["type"] == "*bool":
            return "common.DerefBool(fp.%s)" % n
        if f["type"] == "*LabelThemeMode":
            return "common.Value(fp.%s)" % n
        return "common.Deref(fp.%s)" % n

    def observed_expr(f):
        n = f["name"]
        if n in NUMERIC:
            return "int64(observed.%s)" % n
        return "observed.%s" % n

    def status_expr(f):
        n = f["name"]
        if n in NUMERIC:
            return "cr.Status.AtProvider.%s = common.Int64Ptr(int64(observed.%s))" % (n, n)

        if f["type"] == "*int64":
            return "cr.Status.AtProvider.%s = common.Int64Ptr(int64(observed.%s))" % (n, n)

        if f["type"] == "*bool":
            return "cr.Status.AtProvider.%s = common.BoolPtr(observed.%s)" % (n, n)

        if f["type"] == "*LabelThemeMode":
            mode = "LabelThemeMode(observed.%s)" % n
            return ("if observed.%s != \"\" {\n\t\tmode := v1alpha1.%s\n"
                    "\t\tcr.Status.AtProvider.%s = &mode\n\t}") % (n, mode, n)

        return "cr.Status.AtProvider.%s = common.StringPtr(observed.%s)" % (n, n)

    def compare(f):
        n = f["name"]
        if f["type"] == "*LabelThemeMode":
            return "want.%s != observed.%s" % (n, n)
        return "want.%s != observed.%s" % (n, n)

    out = [LICENSE, "", "package " + snake(kind), "", "import (", '\t"context"', "",
           '\t"github.com/crossplane/crossplane-runtime/v2/pkg/controller"',
           '\t"github.com/crossplane/crossplane-runtime/v2/pkg/meta"',
           '\t"github.com/pkg/errors"', '\tctrl "sigs.k8s.io/controller-runtime"',
           '\t"sigs.k8s.io/controller-runtime/pkg/client"', "",
           '\t"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"',
           '\t"github.com/loafoe/provider-zitadel/internal/clients/zitadel"',
           '\t"github.com/loafoe/provider-zitadel/internal/controller/common"',
           ")", "",
           "// Setup adds a controller that reconciles %s managed resources." % kind,
           "func Setup(mgr ctrl.Manager, o controller.Options) error {",
           "\treturn common.SetupPolicyController(",
           "\t\tmgr, o,",
           "\t\tv1alpha1.%sGroupKind," % kind,
           "\t\tv1alpha1.%sGroupVersionKind," % kind,
           "\t\t&v1alpha1.%s{}," % kind,
           "\t\t&v1alpha1.%sList{}," % kind,
           "\t\tdriver{},",
           "\t)",
           "}", "",
           "// driver is everything specific to the %s. Everything around it is" % p["title"],
           "// shared, because every Zitadel policy behaves the same way.",
           "type driver struct{}", "",
           "var _ common.PolicyDriver[zitadel.%s, zitadel.%s] = driver{}" % (inp, observed),
           "",
           "// Kind names the policy, for errors and events.",
           "func (driver) Kind() string { return \"%s\" }" % kind,
           "",
           "// Scope resolves the organization the policy belongs to. A policy is always",
           "// an organization's own, so an unresolved reference is an error rather than a",
           "// request for the instance default.",
           "func (driver) Scope(ctx context.Context, kube client.Client, cr common.ManagedPolicy) (string, error) {",
           "\tpolicy, ok := cr.(*v1alpha1.%s)" % kind,
           "\tif !ok {",
           "\t\treturn \"\", errors.New(\"managed resource is not a %s custom resource\")" % kind,
           "\t}",
           "",
           "\tfp := policy.Spec.ForProvider",
           "",
           "\torgDefault, err := common.ProviderConfigOrganizationID(ctx, kube, policy)",
           "\tif err != nil {",
           "\t\treturn \"\", common.Join(common.ErrNoOrganizationID, err)",
           "\t}",
           "",
           "\t// A terminating object acts on the organization it recorded: it exists to",
           "\t// reset the policy it set, not one belonging to a replacement.",
           "\torgID, err := common.ResolveOrganizationID(ctx, kube, policy, fp.OrganizationRef,",
           "\t\tfp.OrganizationSelector, fp.OrganizationID, orgDefault,",
           "\t\tcommon.CurrentIfDeleting(meta.WasDeleted(policy), policy.Status.AtProvider.OrganizationID))",
           "\tif err != nil {",
           "\t\treturn \"\", common.Join(common.ErrNoOrganizationID, err)",
           "\t}",
           "",
           "\tif orgID == \"\" {",
           "\t\treturn \"\", errors.New(common.ErrNoOrganizationID.Error())",
           "\t}",
           "",
           "\treturn orgID, nil",
           "}",
           "",
           "// Get reads the organization's %s." % p["title"],
           "func (driver) Get(ctx context.Context, c *zitadel.Client, scope string) (zitadel.%s, bool, error) {" % observed,
           "\tp, err := c.%s(ctx, scope)" % get,
           "\tif err != nil {",
           "\t\treturn zitadel.%s{}, false, err" % observed,
           "\t}",
           "",
           "\treturn *p, p.IsDefault, nil",
           "}",
           "",
           "// Apply writes the policy, adding or updating it as Zitadel requires.",
           "//",
           "// Zitadel separates adding a custom policy from updating one, and which to use",
           "// depends on whether the organization has one. The client works that out, so a",
           "// driver does not have to.",
           "func (driver) Apply(ctx context.Context, c *zitadel.Client, scope string, want zitadel.%s) error {" % inp,
           "\treturn c.%s(ctx, scope, want)" % set_,
           "}",
           "",
           "// Reset puts the organization back on the instance default %s." % p["title"],
           "// Resettable reports that deleting this resource can put the organization",
           "// back on the instance default, which is how Zitadel removes a custom policy.",
           "func (driver) Resettable() bool { return true }",
           "",
           "func (driver) Reset(ctx context.Context, c *zitadel.Client, scope string) error {",
           "\treturn c.%s(ctx, scope)" % reset,
           "}",
           "",
           "// Desired reads the desired policy out of the managed resource.",
           "func (driver) Desired(cr common.ManagedPolicy) zitadel.%s {" % inp,
           "\tfp := cr.(*v1alpha1.%s).Spec.ForProvider" % kind,
           "",
           "\treturn zitadel.%s{" % inp]
    for f in fields:
        if f["name"].endswith("URL"):
            continue
        out.append("\t\t%s: %s," % (f["name"], spec_expr(f)))
    out += ["\t}", "}", "",
            "// Report writes the observed policy into the managed resource's status.",
            "func (driver) Report(mg common.ManagedPolicy, observed zitadel.%s) {" % observed,
            "\tcr := mg.(*v1alpha1.%s)" % kind,
            "",
            "\tcr.Status.AtProvider.OrganizationID = common.StringPtr(observed.OrgID)",
            "\tcr.Status.AtProvider.IsDefault = common.BoolPtr(observed.IsDefault)",
            ""]
    for f in fields:
        out.append("\t" + status_expr(f))
    out += ["}", "",
            "// Equal reports whether the observed policy already matches the desired one.",
            "//",
            "// A field the operator left unset is not compared, so Zitadel's own defaults are",
            "// never fought over.",
            "func (driver) Equal(want zitadel.%s, observed zitadel.%s) bool {" % (inp, observed)]
    compared = [f for f in fields if not f["name"].endswith("URL")]

    # Compared as tables rather than as a chain of conditions: a policy has as
    # many fields as it has settings, and a chain of them is unreadable at the
    # point where a field is added.
    compared = [f for f in fields if not f["name"].endswith("URL")]

    def table(items, go_type):
        if not items:
            return []
        rows = ["\tfor _, f := range []struct {",
                "\t\tname string",
                "\t\twant %s" % go_type,
                "\t\tgot  %s" % go_type,
                "\t}{"]
        for f in items:
            rows.append('\t\t{"%s", want.%s, observed.%s},' % (f["name"], f["name"], f["name"]))
        rows += ["\t} {", "\t\tif f.want != f.got {", "\t\t\treturn false", "\t\t}", "\t}", ""]
        return rows

    def is_bool(f):
        return f["type"] == "*bool"

    def is_num(f):
        return f["name"] in NUMERIC

    out += table([f for f in compared if is_bool(f)], "bool")
    out += table([f for f in compared if is_num(f)], "uint32")
    out += table([f for f in compared if not is_bool(f) and not is_num(f)], "string")
    out += ["\treturn true", "}", ""]

    text = "\n".join(out)

    # Desired() needs the policy fields named fp, and Report needs the concrete
    # kind: name the receiver in Desired.
    text = text.replace(
        "func (driver) Desired(cr common.ManagedPolicy) zitadel.%s {\n\tfp := cr.(*v1alpha1.%s).Spec.ForProvider" % (inp, kind),
        "func (driver) Desired(cr common.ManagedPolicy) zitadel.%s {\n\tfp := cr.(*v1alpha1.%s).Spec.ForProvider" % (inp, kind))
    return text


if __name__ == "__main__":
    import os
    out_dir = sys.argv[1]
    for p in POLICIES:
        d = os.path.join(out_dir, snake(p["kind"]))
        os.makedirs(d, exist_ok=True)
        path = os.path.join(d, snake(p["kind"]) + ".go")
        with open(path, "w") as f:
            f.write(render(p))
        print("wrote", path)

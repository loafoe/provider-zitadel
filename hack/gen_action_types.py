#!/usr/bin/env python3
"""Generate the API types of the action and action execution managed resources.

The five action execution kinds differ only in which condition they bind on,
so they are generated from one template with a condition per kind. The action,
the target and the target's public keys are ordinary resources and are written
out, because what they describe is not a variation on one another.
"""

import textwrap

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

IMPORTS = '''import (
	"reflect"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)'''


def wrap(doc, indent="\t"):
    return "\n".join(indent + "// " + l for l in textwrap.wrap(" ".join(doc.split()), width=76 - len(indent)))


def struct(name, summary, prose, spec, obs=None):
    """Render one managed resource kind."""
    out = ["// %s are the configurable fields of a %s." % (name, name),
           "type %sParameters struct {" % name]
    for f in spec:
        out.append(wrap(f["doc"]))
        out.append("\t// +optional")
        out.append("\t%s %s `json:\"%s,omitempty\"`" % (f["name"], f["type"], f["json"]))
        out.append("")
    out.append("}")
    out.append("")

    out.append("// %sObservation is what Zitadel currently has." % name)
    out.append("type %sObservation struct {" % name)
    for f in obs:
        out.append(wrap(f["doc"]))
        out.append("\t// +optional")
        out.append("\t%s %s `json:\"%s,omitempty\"`" % (f["name"], f["type"], f["json"]))
        out.append("")
    out.append("}")
    out.append("")

    out.append("// %s is %s." % (name, summary.rstrip(".")))

    if prose.strip(" .") != summary.strip(" ."):
        out.append("//")
        out.append(textwrap.fill(" ".join(prose.split()), width=74,
                                 initial_indent="// ", subsequent_indent="// "))

    out.append("// +kubebuilder:object:root=true")
    out.append("// +kubebuilder:subresource:status")
    out.append("// +kubebuilder:storageversion")
    for col in columns.get(name, []):
        out.append(col)
    out.append("// +kubebuilder:printcolumn:name=\"Age\",type=\"date\",JSONPath=\".metadata.creationTimestamp\"")
    out.append("// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}")
    out.append("type %s struct {" % name)
    out.append("\tmetav1.TypeMeta   `json:\",inline\"`")
    out.append("\tmetav1.ObjectMeta `json:\"metadata,omitempty\"`")
    out.append("")
    out.append("\tSpec   %sSpec   `json:\"spec\"`" % name)
    out.append("\tStatus %sStatus `json:\"status,omitempty\"`" % name)
    out.append("}")
    out.append("")
    out.append("// %sSpec defines the desired state of a %s." % (name, name))
    out.append("type %sSpec struct {" % name)
    out.append("\txpv2.ManagedResourceSpec `json:\",inline\"`")
    out.append("\tForProvider              %sParameters `json:\"forProvider\"`" % name)
    out.append("}")
    out.append("")
    out.append("// %sStatus is the observed state of a %s." % (name, name))
    out.append("type %sStatus struct {" % name)
    out.append("\txpv1.ResourceStatus `json:\",inline\"`")
    out.append("\tAtProvider               %sObservation `json:\"atProvider,omitempty\"`" % name)
    out.append("}")
    out.append("")
    out.append("// %sList contains a list of %s." % (name, name))
    out.append("//")
    out.append("// +kubebuilder:object:root=true")
    out.append("type %sList struct {" % name)
    out.append("\tmetav1.TypeMeta `json:\",inline\"`")
    out.append("\tmetav1.ListMeta `json:\"metadata,omitempty\"`")
    out.append("\tItems           []%s `json:\"items\"`" % name)
    out.append("}")
    out.append("")
    out.append("// %s type metadata." % name)
    out.append("var (")
    out.append("\t%sKind             = reflect.TypeOf(%s{}).Name()" % (name, name))
    out.append("\t%sGroupKind        = schema.GroupKind{Group: Group, Kind: %sKind}.String()" % (name, name))
    out.append("\t%sKindAPIVersion   = %sKind + \".\" + SchemeGroupVersion.String()" % (name, name))
    out.append("\t%sGroupVersionKind = SchemeGroupVersion.WithKind(%sKind)" % (name, name))
    out.append(")")
    out.append("")
    out.append("func init() {")
    out.append("\tSchemeBuilder.Register(&%s{}, &%sList{})" % (name, name))
    out.append("}")
    out.append("")
    return "\n".join(out)


def summary_of(kind):
    """The one line that says what the kind is, for the type's own doc comment."""
    return {
        "Action": "a JavaScript snippet Zitadel runs during a login",
        "ActionTarget": "somewhere an action's payload is sent",
        "ActionTargetPublicKey": "a key a target's payloads are encrypted with",
    }[kind]


def f(name, json_name, type_, doc):
    return {"name": name, "json": json_name, "type": type_, "doc": doc}


ORG = [
    f("OrganizationID", "organizationID", "*string",
      "The ID of the organization the action belongs to. Either `organizationID`, "
      "`organizationRef` or `organizationSelector` must be set."),
    f("OrganizationRef", "organizationRef", "*xpv1.Reference",
      "OrganizationRef references an Organization managed by this provider and uses its ID."),
    f("OrganizationSelector", "organizationSelector", "*xpv1.Selector",
      "OrganizationSelector selects an Organization managed by this provider and uses its ID."),
]

extra_doc = {
    "Action": "A JavaScript snippet Zitadel runs at a point in a login flow. The "
              "script itself is managed through Zitadel's v1 management API, "
              "because the v2 action service manages where actions are sent rather "
              "than what they do.",
    "ActionTarget": "Somewhere an action's payload is sent: a webhook, a call back "
                    "into Zitadel, or an asynchronous request. Zitadel generates a "
                    "signing key for each target, which is published to the connection "
                    "secret so that the receiving side can verify what it gets.",
    "ActionTargetPublicKey": "A key that a target's payloads are encrypted with. It "
                             "belongs to an ActionTarget, and is what makes the "
                             "`Jwe` payload type possible.",
}

columns = {
    "Action": ['// +kubebuilder:printcolumn:name="Organization",type="string",JSONPath=".status.atProvider.organizationID"',
               '// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.atProvider.state"'],
    "ActionTarget": ['// +kubebuilder:printcolumn:name="Type",type="string",JSONPath=".status.atProvider.type"',
                     '// +kubebuilder:printcolumn:name="Endpoint",type="string",JSONPath=".status.atProvider.endpoint"'],
    "ActionTargetPublicKey": ['// +kubebuilder:printcolumn:name="Active",type="boolean",JSONPath=".status.atProvider.active"'],
}

KINDS = [
    ("Action", ORG + [
        f("Name", "name", "*string", "Name of the action, shown in the console."),
        f("Script", "script", "*string",
          "The JavaScript the action runs. It is sent to Zitadel as written and is "
          "never read back, so a change here is applied but never drift detected."),
        f("Timeout", "timeout", "*string",
          "How long the action may run before Zitadel gives up, such as `10s`."),
        f("AllowedToFail", "allowedToFail", "*bool",
          "Carry on with the next action when this one fails."),
        f("State", "state", "*ActionState",
          "Whether the action runs. One of `Active` or `Inactive`. An inactive "
          "action keeps its script and its bindings. Defaults to `Active`."),
    ], [
        f("OrganizationID", "organizationID", "*string",
          "The organization the action belongs to."),
        f("ID", "id", "string", "Zitadel's identifier of the action."),
        f("Name", "name", "*string", "Name of the action."),
        f("Timeout", "timeout", "*string", "How long the action may run."),
        f("AllowedToFail", "allowedToFail", "*bool",
          "Whether the next action runs when this one fails."),
        f("State", "state", "*string",
          "Whether the action runs. One of `Active` or `Inactive`."),
    ]),
    ("ActionTarget", [
        f("Name", "name", "*string", "Name of the target, shown in the console."),
        f("Type", "type", "*ActionTargetType",
          "How the target is called. One of `Webhook`, `Call` or `Async`."),
        f("Endpoint", "endpoint", "*string",
          "Where the payload is sent, as an absolute URL."),
        f("Timeout", "timeout", "*string",
          "How long Zitadel waits for the target, such as `10s`."),
        f("PayloadType", "payloadType", "*ActionPayloadType",
          "How the payload is encoded. One of `Json`, `Jwt` or `Jwe`. `Jwe` needs an "
          "active ActionTargetPublicKey on the target."),
    ], [
        f("ID", "id", "string", "Zitadel's identifier of the target."),
        f("Name", "name", "*string", "Name of the target."),
        f("Type", "type", "*string", "How the target is called."),
        f("Endpoint", "endpoint", "*string", "Where the payload is sent."),
        f("Timeout", "timeout", "*string", "How long Zitadel waits for the target."),
        f("PayloadType", "payloadType", "*string", "How the payload is encoded."),
        f("SigningKey", "signingKey", "string",
          "The key Zitadel signs this target's payloads with. Generated once, at "
          "creation, and never changed."),
    ]),
    ("ActionTargetPublicKey", [
        f("TargetID", "targetID", "*string",
          "The ID of the ActionTarget the key belongs to. Either `targetID`, "
          "`targetRef` or `targetSelector` must be set."),
        f("TargetRef", "targetRef", "*xpv1.Reference",
          "TargetRef references an ActionTarget managed by this provider and uses its ID."),
        f("TargetSelector", "targetSelector", "*xpv1.Selector",
          "TargetSelector selects an ActionTarget managed by this provider and uses its ID."),
        f("PublicKey", "publicKey", "*string",
          "The public key in PEM form, RSA or EC. It cannot be changed in place: "
          "Zitadel ties a key to the payload encryption of the target, so rotating "
          "one means adding a new key and removing the old."),
        f("ExpirationDate", "expirationDate", "*metav1.Time",
          "When the key stops being accepted, as an RFC3339 timestamp."),
        f("Active", "active", "*bool",
          "Whether the key is used to encrypt. Only one key on a target encrypts at "
          "a time, so adding a key and activating it is how a rotation is done. "
          "Defaults to true."),
    ], [
        f("TargetID", "targetID", "*string", "The ActionTarget the key belongs to."),
        f("KeyID", "keyID", "string",
          "Zitadel's identifier of the key, used as `kid` in the payload header."),
        f("PublicKey", "publicKey", "*string", "The public key in PEM form."),
        f("Active", "active", "*bool", "Whether the key is used to encrypt."),
        f("Fingerprint", "fingerprint", "string", "Zitadel's fingerprint of the key."),
        f("CreationDate", "creationDate", "string",
          "When the key was added, as an RFC3339 timestamp."),
    ]),
]

# The five execution kinds, and the condition each one binds on.
EXECUTIONS = [
    ("ActionExecutionRequest",
     "an incoming gRPC request",
     [("Method", "method", "*string",
       "The gRPC method to run on, such as `/zitadel.session.v2.SessionService/GetSession`."),
      ("Service", "service", "*string",
       "The gRPC service to run on, such as `zitadel.session.v2.SessionService`. "
       "Every method of it triggers. Exactly one of `method`, `service` or `all` "
       "must be set."),
      ("All", "all", "*bool",
       "Run on every request Zitadel handles. Exactly one of `method`, `service` or "
       "`all` must be set.")],
     "a request Zitadel handles"),
    ("ActionExecutionResponse",
     "a gRPC response Zitadel is about to return",
     [("Method", "method", "*string",
       "The gRPC method whose response triggers the action."),
      ("Service", "service", "*string",
       "The gRPC service whose responses trigger the action. Exactly one of "
       "`method`, `service` or `all` must be set."),
      ("All", "all", "*bool",
       "Run on every response Zitadel is about to return. Exactly one of `method`, "
       "`service` or `all` must be set.")],
     "a response Zitadel is about to return"),
    ("ActionExecutionEvent",
     "a Zitadel event",
     [("Event", "event", "*string",
       "The event to run on, such as `user.human.created`."),
      ("Group", "group", "*string",
       "The group of events to run on, such as `user`. Exactly one of `event`, "
       "`group` or `all` must be set."),
      ("All", "all", "*bool",
       "Run on every event Zitadel emits. Exactly one of `event`, `group` or `all` "
       "must be set.")],
     "an event Zitadel emits"),
    ("ActionExecutionFunction",
     "a function another action calls",
     [("Name", "name", "*string", "The name of the function to run.")],
     "a function another action calls"),
]

if __name__ == "__main__":
    import os
    import sys

    out_dir = sys.argv[1]

    def snake(name):
        import re
        parts = re.findall(r"[A-Z]+(?![a-z])|[A-Z][a-z0-9]*|[a-z0-9]+", name)
        return "_".join(p.lower() for p in parts)

    for name, spec, obs in KINDS:
        body = LICENSE + "\npackage v1alpha1\n\n" + IMPORTS + "\n\n"
        body += struct(name, summary_of(name), extra_doc[name], spec, obs)
        path = os.path.join(out_dir, snake(name) + "_types.go")
        with open(path, "w") as fh:
            fh.write(body)
        print("wrote", path)

    # Every execution kind is a binding of targets to one condition, so they are
    # rendered from one template with the condition differing.
    for name, when, cond, cond_doc in EXECUTIONS:
        cond = [f(*c) for c in cond]
        spec = [
            f("TargetIDs", "targetIDs", "[]string",
              "The ActionTargets to call. Zitadel validates that each one exists "
              "before the binding is written, so a typo is reported here rather "
              "than at the moment the action would have run."),
        ] + cond

        obs = [f("TargetIDs", "targetIDs", "[]string",
                 "The ActionTargets this binding currently calls."),
               f("CreationDate", "creationDate", "string",
                 "When the binding was created, as an RFC3339 timestamp."),
               f("ChangeDate", "changeDate", "string",
                 "When the binding last changed, as an RFC3339 timestamp.")]

        summary = "ActionTargets to call when Zitadel handles %s" % when
        doc = ("Bind ActionTargets to %s.\n"
               "Zitadel gives a binding no identifier of its own: the condition is\n"
               "the identity, and reading one back means listing every binding and\n"
               "matching. Deleting the resource clears the binding rather than\n"
               "removing it, because that is the only thing Zitadel can do." % when)

        body = LICENSE + "\npackage v1alpha1\n\n" + IMPORTS + "\n\n"
        body += struct(name, cond_doc, doc, spec, obs)
        extra_doc[name] = doc
        columns[name] = ['// +kubebuilder:printcolumn:name="Targets",type="string",JSONPath=".status.atProvider.targetIDs"']
        path = os.path.join(out_dir, snake(name) + "_types.go")
        with open(path, "w") as fh:
            fh.write(body)
        print("wrote", path)

    # The trigger actions resource: the same binding, at a login flow's trigger.
    name = "TriggerActions"
    spec = [
        f("OrganizationID", "organizationID", "*string",
          "The ID of the organization whose login flow is customised. Either "
          "`organizationID`, `organizationRef` or `organizationSelector` must be set."),
        f("OrganizationRef", "organizationRef", "*xpv1.Reference",
          "OrganizationRef references an Organization managed by this provider and uses its ID."),
        f("OrganizationSelector", "organizationSelector", "*xpv1.Selector",
          "OrganizationSelector selects an Organization managed by this provider and uses its ID."),
        f("FlowType", "flowType", "*TriggerFlowType",
          "Which login flow the trigger belongs to."),
        f("TriggerType", "triggerType", "*TriggerType",
          "Where in that flow the actions run."),
        f("ActionIDs", "actionIDs", "[]string",
          "The Actions to run at that point."),
    ]
    obs = [
        f("OrganizationID", "organizationID", "*string",
          "The organization whose login flow is customised."),
        f("ActionIDs", "actionIDs", "[]string",
          "The Actions the trigger currently runs."),
    ]
    summary = "the actions to run at a point in a login flow"
    doc = ("Bind Actions to a point in a login flow, such as after a user has "
           "authenticated. Zitadel has no call for listing these, so they are "
           "read back out of the flow and the wanted trigger picked from it. "
           "Deleting the resource clears the trigger rather than removing it.")
    body = LICENSE + "\npackage v1alpha1\n\n" + IMPORTS + "\n\n"
    body += struct(name, summary, doc, spec, obs)
    extra_doc[name] = doc
    columns[name] = ['// +kubebuilder:printcolumn:name="Organization",type="string",JSONPath=".status.atProvider.organizationID"',
                     '// +kubebuilder:printcolumn:name="Flow",type="string",JSONPath=".spec.forProvider.flowType"',
                     '// +kubebuilder:printcolumn:name="Trigger",type="string",JSONPath=".spec.forProvider.triggerType"']
    path = os.path.join(out_dir, snake(name) + "_types.go")
    with open(path, "w") as fh:
        fh.write(body)
    print("wrote", path)

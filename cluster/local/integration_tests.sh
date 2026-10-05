#!/usr/bin/env bash
# Copyright 2025 The Crossplane Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Integration tests: the provider against a real Zitadel.
#
# `make test` proves the provider's own logic. It cannot prove that the requests
# it sends are ones Zitadel accepts, that the shapes it reads back are the ones
# Zitadel returns, or that a finalizer is released when Zitadel behaves the way
# its documentation says. That needs a real instance, so this script stands one
# up: Zitadel in a kind cluster, the provider built from this tree and deployed
# against it, and then a set of resources applied and watched until they settle.
#
# STATUS: this script has not been run end to end, and it is deliberately not a
# CI gate until it has been. Everything below is the design it was written
# against; treat the details as unproven. It is a developer tool, not a check.

set -euo pipefail

# Settings. Every one of them can be overridden from the environment, which is
# what lets this be run against something other than a throwaway cluster.
KIND_CLUSTER="${KIND_CLUSTER:-provider-zitadel-integration}"
KIND_NODE_IMAGE_TAG="${KIND_NODE_IMAGE_TAG:-v1.34.0}"
CROSSPLANE_NAMESPACE="${CROSSPLANE_NAMESPACE:-crossplane-system}"
CROSSPLANE_CHART_VERSION="${CROSSPLANE_CHART_VERSION:-2.4.2}"
ZITADEL_NAMESPACE="${ZITADEL_NAMESPACE:-zitadel}"
ZITADEL_CHART_VERSION="${ZITADEL_CHART_VERSION:-10.3.0}"
ZITADEL_PORT="${ZITADEL_PORT:-8080}"
TEST_NAMESPACE="${TEST_NAMESPACE:-zitadel-system}"
TEST_TIMEOUT="${TEST_TIMEOUT:-10m}"

# Zitadel requires PostgreSQL. The chart can bundle one, but its setup job is a
# pre-install hook and so is not retried once it has failed against a database
# that does not exist yet. The database therefore gets its own release, and
# Zitadel is installed against it.
ZITADEL_DB_RELEASE="${ZITADEL_DB_RELEASE:-zitadel-db}"
ZITADEL_DB_PASSWORD="${ZITADEL_DB_PASSWORD:-zitadel}"

# The instance's master key. Generated rather than hardcoded so a rerun never
# starts from a predictable instance.
ZITADEL_MASTERKEY="${ZITADEL_MASTERKEY:-$(head -c 32 /dev/urandom | base64)}"

# Keeps the cluster when a run fails, so the failure can be investigated rather
# than re-diagnosed.
KEEP_CLUSTER="${KEEP_CLUSTER:-false}"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CRD_DIR="${ROOT_DIR}/package/crds"
CONTROLLER_IMAGE="${CONTROLLER_IMAGE:-provider-zitadel:integration}"

# The Zitadel chart seeds a first instance and writes the credentials it made to
# a secret. Using that is the whole reason the ProviderConfig can be a service
# account key here: a Personal Access Token would work too, but it would not
# exercise the key path, which is the one an operator is most likely to get
# wrong.
ZITADEL_MACHINE_KEY_SECRET="${ZITADEL_MACHINE_KEY_SECRET:-iam-admin}"
ZITADEL_MACHINE_KEY_FIELD="${ZITADEL_MACHINE_KEY_FIELD:-iam-admin.json}"

log() {
  echo "==> $*"
}

fail() {
  echo "FAIL: $*" >&2
  diagnostics
  exit 1
}

# diagnostics prints what is going on in the cluster. A failure here is almost
# always "the resource never became ready", and the reason is nearly always in a
# controller or Zitadel log rather than in the test.
diagnostics() {
  echo "" >&2
  echo "---- diagnostics ----" >&2

  kubectl get pods -A >&2 || true
  echo "" >&2
  kubectl -n "${CROSSPLANE_NAMESPACE}" logs deploy/crossplane --tail=200 >&2 || true
  echo "" >&2
  kubectl -n "${ZITADEL_NAMESPACE}" logs -l app.kubernetes.io/name=zitadel --tail=200 >&2 || true

  if [ "${KEEP_CLUSTER}" = "true" ]; then
    echo "" >&2
    echo "the cluster ${KIND_CLUSTER} was kept for inspection" >&2
  else
    kind delete cluster --name "${KIND_CLUSTER}" >&2 || true
  fi
}

# wait_for polls until the command succeeds or the timeout runs out.
wait_for() {
  local description="$1"
  shift

  log "waiting for ${description}"

  if ! timeout "${TEST_TIMEOUT}" bash -c "until $* >/dev/null 2>&1; do sleep 2; done"; then
    fail "timed out waiting for ${description}"
  fi

  log "${description} is ready"
}

# need checks for the tools this script drives, by name, so a missing one is a
# clear message rather than a failure somewhere in the middle.
need() {
  for tool in "$@"; do
    command -v "${tool}" >/dev/null 2>&1 || fail "${tool} is required but not installed"
  done
}

create_cluster() {
  log "creating the kind cluster"

  if kind get clusters | grep -qx "${KIND_CLUSTER}"; then
    kind delete cluster --name "${KIND_CLUSTER}"
  fi

  KIND_NODE_IMAGE_TAG="${KIND_NODE_IMAGE_TAG}" kind create cluster \
    --name "${KIND_CLUSTER}" \
    --wait 120s

  kubectl cluster-info --context "kind-${KIND_CLUSTER}"
}

# install_crossplane installs Crossplane from its chart rather than from an
# install manifest. There is no manifest to fetch any more - the paths a
# generated install script would use are gone from the upstream repository - and
# the chart is the supported route.
install_crossplane() {
  log "installing Crossplane"

  helm repo add crossplane-stable https://charts.crossplane.io/stable >/dev/null
  helm repo update crossplane-stable >/dev/null

  helm upgrade --install crossplane crossplane-stable/crossplane \
    --namespace "${CROSSPLANE_NAMESPACE}" \
    --create-namespace \
    --version "${CROSSPLANE_CHART_VERSION}" \
    --wait --timeout "${TEST_TIMEOUT}"

  wait_for "Crossplane to be ready" \
    "kubectl -n ${CROSSPLANE_NAMESPACE} rollout status deployment/crossplane --timeout=300s"
}

# install_zitadel stands up a database and then the instance against it. The
# instance is deliberately left without TLS, because everything here talks to it
# over the cluster network.
install_zitadel() {
  log "installing the Zitadel database"

  helm repo add bitnami https://charts.bitnami.com/bitnami >/dev/null
  helm repo update bitnami >/dev/null

  helm upgrade --install "${ZITADEL_DB_RELEASE}" bitnami/postgresql \
    --namespace "${ZITADEL_NAMESPACE}" \
    --create-namespace \
    --set auth.database=zitadel \
    --set auth.username=zitadel \
    --set "auth.password=${ZITADEL_DB_PASSWORD}" \
    --wait --timeout "${TEST_TIMEOUT}"

  log "installing Zitadel"

  helm repo add zitadel https://charts.zitadel.io >/dev/null
  helm repo update zitadel >/dev/null

  helm upgrade --install zitadel zitadel/zitadel \
    --namespace "${ZITADEL_NAMESPACE}" \
    --version "${ZITADEL_CHART_VERSION}" \
    --set "zitadel.masterkey=${ZITADEL_MASTERKEY}" \
    --set zitadel.configmapConfig.Database.Host="${ZITADEL_DB_RELEASE}-postgresql" \
    --set zitadel.configmapConfig.Database.Port=5432 \
    --set zitadel.configmapConfig.Database.Database=zitadel \
    --set zitadel.configmapConfig.Database.UserUsername=zitadel \
    --set "zitadel.configmapConfig.Database.UserPassword=${ZITADEL_DB_PASSWORD}" \
    --set zitadel.configmapConfig.Database.UserSSLMode=disable \
    --wait --timeout "${TEST_TIMEOUT}"

  wait_for "the Zitadel API to answer" \
    "kubectl -n ${ZITADEL_NAMESPACE} exec deploy/zitadel -- wget -q -O- http://localhost:${ZITADEL_PORT}/.well-known/openid-configuration"
}

# build_controller_image builds the provider from this tree and wraps it in the
# same image the release builds, so the tests run the artefact that ships.
build_controller_image() {
  log "building the provider image"

  local platform
  platform="$(go env GOOS)_$(go env GOARCH)"

  mkdir -p "${ROOT_DIR}/_output/bin/${platform}"

  # The Dockerfile consumes bin/${TARGETOS}_${TARGETARCH}/provider, which is the
  # layout the build submodule produces. Building it here keeps this script
  # independent of that submodule's targets.
  (cd "${ROOT_DIR}" && CGO_ENABLED=0 GOOS="$(go env GOOS)" GOARCH="$(go env GOARCH)" \
    go build -o "${ROOT_DIR}/_output/bin/${platform}/provider" ./cmd/provider)

  (cd "${ROOT_DIR}/cluster/images/provider-zitadel" && \
    docker build \
      --build-arg "TARGETOS=$(go env GOOS)" \
      --build-arg "TARGETARCH=$(go env GOARCH)" \
      -t "${CONTROLLER_IMAGE}" \
      .)

  kind load docker-image "${CONTROLLER_IMAGE}" --name "${KIND_CLUSTER}"
}

# deploy_provider runs the provider as a plain Deployment, which is the simplest
# thing that exercises the controllers without going through the xpkg machinery.
deploy_provider() {
  log "deploying the provider"

  kubectl create namespace "${PROVIDER_NAMESPACE:-${CROSSPLANE_NAMESPACE}}" --dry-run=client -o yaml | kubectl apply -f -

  local ns="${PROVIDER_NAMESPACE:-${CROSSPLANE_NAMESPACE}}"

  kubectl -n "${ns}" apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: provider-zitadel
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: provider-zitadel-integration
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
  - kind: ServiceAccount
    name: provider-zitadel
    namespace: ${ns}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: provider-zitadel
spec:
  replicas: 1
  selector:
    matchLabels:
      app: provider-zitadel
  template:
    metadata:
      labels:
        app: provider-zitadel
    spec:
      serviceAccountName: provider-zitadel
      containers:
        - name: provider
          image: ${CONTROLLER_IMAGE}
          imagePullPolicy: IfNotPresent
          args:
            - --debug
          env:
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
EOF

  wait_for "the provider to be ready" \
    "kubectl -n ${ns} rollout status deployment/provider-zitadel --timeout=300s"

  log "installing the provider CRDs"

  kubectl apply -f "${CRD_DIR}"

  wait_for "the provider CRDs to be established" \
    "kubectl wait --for=condition=Established --timeout=180s crd --all -l provider=zitadel"
}

# configure_provider creates the ProviderConfig the tests authenticate with,
# from the machine key the Zitadel chart generated for its first instance.
configure_provider() {
  log "creating the ProviderConfig"

  kubectl create namespace "${TEST_NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

  wait_for "the Zitadel chart to publish its machine key" \
    "kubectl -n ${ZITADEL_NAMESPACE} get secret ${ZITADEL_MACHINE_KEY_SECRET}"

  kubectl -n "${ZITADEL_NAMESPACE}" get secret "${ZITADEL_MACHINE_KEY_SECRET}" \
    -o "jsonpath={.data.${ZITADEL_MACHINE_KEY_FIELD}}" | base64 --decode > /tmp/zitadel-key.json

  if [ ! -s /tmp/zitadel-key.json ]; then
    fail "the machine key the Zitadel chart generated is empty"
  fi

  kubectl -n "${TEST_NAMESPACE}" apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: zitadel-sa-key
stringData:
  key.json: |
$(sed 's/^/    /' /tmp/zitadel-key.json)
EOF

  kubectl -n "${TEST_NAMESPACE}" apply -f - <<EOF
apiVersion: zitadel.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: zitadel
  namespace: ${TEST_NAMESPACE}
spec:
  url: http://zitadel.${ZITADEL_NAMESPACE}.svc.cluster.local:${ZITADEL_PORT}
  insecure: true
  credentials:
    source: Secret
    authType: ServiceAccount
    serviceAccount:
      keySecretRef:
        name: zitadel-sa-key
        key: key.json
EOF

  wait_for "the ProviderConfig to be accepted" \
    "kubectl -n ${TEST_NAMESPACE} get providerconfig zitadel"
}

# apply_and_wait applies a manifest and waits for every managed resource in it to
# report ready. That wait is the assertion: a resource that never becomes ready
# is a provider bug, and `kubectl wait` turns it into a failure rather than a
# hung pipeline.
apply_and_wait() {
  local manifest="$1"

  log "applying $(basename "${manifest}")"

  kubectl -n "${TEST_NAMESPACE}" apply -f "${manifest}"

  local resources
  resources="$(kubectl -n "${TEST_NAMESPACE}" get -f "${manifest}" \
    -o jsonpath='{range .items[*]}{.kind}{"/"}{.metadata.name}{"\n"}{end}')"

  for resource in ${resources}; do
    local kind="${resource%%/*}"
    local name="${resource##*/}"

    log "waiting for ${resource}"

    if ! kubectl -n "${TEST_NAMESPACE}" wait "${kind}/${name}" \
      --for=condition=Ready --timeout="${TEST_TIMEOUT}"; then
      echo "---- ${resource} ----" >&2
      kubectl -n "${TEST_NAMESPACE}" get "${kind}/${name}" -o yaml >&2
      kubectl -n "${TEST_NAMESPACE}" describe "${kind}/${name}" >&2
      fail "${resource} never became ready"
    fi

    log "${resource} is ready"
  done
}

# run_tests drives a set of resources through their whole life.
#
# They are chosen for what they cover rather than for how many there are: an
# organization and a project give the reference resolution something to resolve,
# a user and a service account cover both kinds of user, and the policies cover
# the add-or-update lifecycle that is the easiest thing in this provider to get
# wrong. The delete at the end matters just as much: a resource that becomes
# ready but never releases its finalizer is the failure this catches.
run_tests() {
  local create_order=(
    organization.yaml
    project.yaml
    humanuser.yaml
    serviceaccount.yaml
    lockoutpolicy.yaml
    loginpolicy.yaml
  )

  local delete_order=(
    loginpolicy.yaml
    lockoutpolicy.yaml
    serviceaccount.yaml
    humanuser.yaml
    project.yaml
    organization.yaml
  )

  for manifest in "${create_order[@]}"; do
    local path="${ROOT_DIR}/examples/${manifest}"
    if [ ! -f "${path}" ]; then
      echo "SKIP: ${manifest} is not in examples/" >&2
      continue
    fi

    apply_and_wait "${path}"
  done

  log "deleting the resources, to prove the finalizers are released"

  for manifest in "${delete_order[@]}"; do
    local path="${ROOT_DIR}/examples/${manifest}"
    [ -f "${path}" ] || continue

    kubectl -n "${TEST_NAMESPACE}" delete -f "${path}" --ignore-not-found --timeout="${TEST_TIMEOUT}"
  done

  wait_for "the managed resources to drain" \
    "test -z \"\$(kubectl -n ${TEST_NAMESPACE} get managed.zitadel.m.crossplane.io,clusters.zitadel.crossplane.io -o name 2>/dev/null)\""
}

main() {
  need kind kubectl helm docker go

  create_cluster
  install_crossplane
  install_zitadel
  build_controller_image
  deploy_provider
  configure_provider
  run_tests

  log "integration tests passed"

  [ "${KEEP_CLUSTER}" = "true" ] || kind delete cluster --name "${KIND_CLUSTER}"
}

main "$@"
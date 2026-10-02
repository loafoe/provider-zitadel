#!/usr/bin/env bash
# Generate the managed resource API types, their deepcopy methods, their CRDs
# and their Crossplane methodsets.
#
# Adding a new managed resource kind needs its deepcopy methods, and the
# deepcopy generator only runs on a package that compiles. A freshly generated
# kind does not compile, because registering it with the scheme requires the
# DeepCopyObject method that the generator has not written yet. So the
# registration calls are parked for the deepcopy pass and put back afterwards.
#
# The Crossplane methodsets are generated in a second pass, once the package
# compiles again: that generator works from the kinds the scheme knows about, so
# it sees nothing that is still parked.
set -euo pipefail

cd "$(dirname "$0")/.."

TYPES_DIR=apis/zitadel/v1alpha1
STUB="$TYPES_DIR/zz_parked_registration.go"
BACKUP=$(mktemp -d)

restore() {
  rm -f "$STUB"
  if [ -d "$BACKUP" ]; then
    cp -a "$BACKUP/." "$TYPES_DIR/" 2>/dev/null || true
    rm -rf "$BACKUP"
  fi
}
trap restore EXIT

unpark() {
  for f in "$TYPES_DIR"/*.go; do
    [ "$(basename "$f")" = "$(basename "$STUB")" ] && continue
    perl -pi -e 's/\bparkSchemeRegister\(/SchemeBuilder.Register(/g' "$f"
  done
  rm -f "$STUB"
}

# A previous run that was interrupted between parking and restoring would have
# left parked calls on disk, and those would be copied into the backup. Undo
# them first so that what gets restored is the real code.
unpark

# Pass one: deepcopy methods and CRDs.
mkdir -p "$BACKUP"
for f in "$TYPES_DIR"/*.go; do
  if grep -q 'SchemeBuilder.Register(' "$f"; then
    cp "$f" "$BACKUP/$(basename "$f")"
    perl -pi -e 's/^(\s*)SchemeBuilder\.Register\(/${1}parkSchemeRegister(/' "$f"
  fi
done

# The parked call has to exist and accept anything: the real Register insists on
# types implementing runtime.Object, which is exactly the method the first pass
# has not written yet.
cat > "$STUB" <<'EOF'
package v1alpha1

// parkSchemeRegister stands in for SchemeBuilder.Register while the deepcopy
// methods are generated. See hack/generate.sh.
func parkSchemeRegister(...any) {}
EOF

if ! ( cd apis && go run -tags generate sigs.k8s.io/controller-tools/cmd/controller-gen \
  "object:headerFile=../hack/boilerplate.go.txt" paths=./... \
  "crd:crdVersions=v1" "output:artifacts:config=../package/crds" ); then
  exit 1
fi

restore
trap - EXIT
unpark

# Pass two: Crossplane methodsets, now that every kind is registered and the
# package compiles.
( cd apis && go run -tags generate github.com/crossplane/crossplane-tools/cmd/angryjet \
  generate-methodsets --header-file=../hack/boilerplate.go.txt ./... )

( cd apis && go run -tags generate sigs.k8s.io/controller-tools/cmd/controller-gen \
  "object:headerFile=../hack/boilerplate.go.txt" paths=./... \
  "crd:crdVersions=v1" "output:artifacts:config=../package/crds" )

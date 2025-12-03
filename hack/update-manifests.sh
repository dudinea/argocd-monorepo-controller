#! /usr/bin/env bash
set -x
set -o errexit
set -o nounset
set -o pipefail
set -e

SRCROOT="$( CDPATH='' cd -- "$(dirname "$0")/.." && pwd -P )"
AUTOGENMSG="# This is an auto-generated file. DO NOT EDIT"

KUSTOMIZE=kustomize
[ -f "$SRCROOT/dist/kustomize" ] && KUSTOMIZE="$SRCROOT/dist/kustomize"

IMAGE_NAMESPACE="${IMAGE_NAMESPACE:-quay.io/argoprojlabs}"
IMAGE_TAG="${IMAGE_TAG:-}"

# if the tag has not been declared, and we are on a release branch, use the VERSION file.
if [ "$IMAGE_TAG" = "" ]; then
  branch=$(git rev-parse --abbrev-ref HEAD)
  if [[ $branch = release-* ]]; then
    pwd
    IMAGE_TAG=v$(cat $SRCROOT/VERSION)
  fi
fi
# otherwise, use latest
if [ "$IMAGE_TAG" = "" ]; then
  IMAGE_TAG=latest
fi

$KUSTOMIZE version
which $KUSTOMIZE

cd ${SRCROOT}/manifests/base && $KUSTOMIZE edit set image quay.io/argoprojlabs/argocd-monorepo-controller=${IMAGE_NAMESPACE}/argocd-monorepo-controller:${IMAGE_TAG}

echo "${AUTOGENMSG}" > "${SRCROOT}/manifests/install.yaml"
$KUSTOMIZE build "${SRCROOT}/manifests/cluster-install" >> "${SRCROOT}/manifests/install.yaml"

echo "${AUTOGENMSG}" > "${SRCROOT}/manifests/namespace-install.yaml"
$KUSTOMIZE build "${SRCROOT}/manifests/namespace-install" >> "${SRCROOT}/manifests/namespace-install.yaml"



#!/bin/bash
#
# Development helper that installs the Monorepo Controller UI extension into a
# running Argo CD installation.
#
# It stores ui/extension-monorepo-controller.js in a ConfigMap and patches the
# argocd-server Deployment to mount that ConfigMap under /tmp/extensions, where
# argocd-server picks up UI extensions from.
#
# NOTE: this modifies the argocd-server Deployment, which is part of the Argo CD
# installation and is not managed by this project. Use -u to revert the change.
#
# This is not how the extension is meant to be installed in production: there it
# is delivered with the argocd-extension-installer init container.

set -eu

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
EXTENSION_JS="${SCRIPT_DIR}/../dist/extension-monorepo-controller.js"
PATCH_FILE="${SCRIPT_DIR}/argocd-server-patch.yaml"

CONFIGMAP_NAME="argocd-monorepo-ui-extension"
DEPLOYMENT_NAME="argocd-server"
CONTAINER_NAME="argocd-server"
VOLUME_NAME="monorepo-ui-extension"

NAMESPACE="${ARGOCD_NAMESPACE:-argocd}"
ACTION="install"

usage() {
    cat <<EOF
Usage: $(basename "$0") [-n NAMESPACE] [-u] [-h]

  -n NAMESPACE  namespace of the Argo CD installation
                (default: \$ARGOCD_NAMESPACE or "argocd")
  -u            uninstall the extension
  -h            show this help
EOF
}

# index_of NAMES NAME
# Prints the zero based index of NAME in the newline separated list NAMES.
# Returns 1 if NAME is not in the list.
index_of() {
    local index
    index="$(printf '%s\n' "$1" | grep -n -x -F -- "$2" | head -n 1 | cut -d: -f1 || true)"
    if [ -z "${index}" ]; then
        return 1
    fi
    echo $((index - 1))
}

deployment_jsonpath() {
    kubectl -n "${NAMESPACE}" get deployment "${DEPLOYMENT_NAME}" -o jsonpath="$1"
}

container_index() {
    local names
    names="$(deployment_jsonpath '{range .spec.template.spec.containers[*]}{.name}{"\n"}{end}')"
    index_of "${names}" "${CONTAINER_NAME}"
}

volume_index() {
    local names
    names="$(deployment_jsonpath '{range .spec.template.spec.volumes[*]}{.name}{"\n"}{end}')"
    index_of "${names}" "${VOLUME_NAME}"
}

volume_mount_index() {
    local names
    names="$(deployment_jsonpath "{range .spec.template.spec.containers[$1].volumeMounts[*]}{.name}{\"\\n\"}{end}")"
    index_of "${names}" "${VOLUME_NAME}"
}

check_deployment() {
    if ! kubectl -n "${NAMESPACE}" get deployment "${DEPLOYMENT_NAME}" >/dev/null 2>&1; then
        echo "error: deployment ${DEPLOYMENT_NAME} not found in namespace ${NAMESPACE}" >&2
        echo "       is Argo CD installed there? use -n to select another namespace" >&2
        exit 1
    fi
}

install_extension() {
    if [ ! -f "${EXTENSION_JS}" ]; then
        echo "error: ${EXTENSION_JS} not found" >&2
        echo "       build it first: make build-ui-local" >&2
        exit 1
    fi
    check_deployment

    echo "==> creating ConfigMap ${CONFIGMAP_NAME} in namespace ${NAMESPACE}"
    kubectl -n "${NAMESPACE}" create configmap "${CONFIGMAP_NAME}" \
        --from-file="$(basename "${EXTENSION_JS}")=${EXTENSION_JS}" \
        --dry-run=client -o yaml | kubectl -n "${NAMESPACE}" apply -f -

    echo "==> patching deployment ${DEPLOYMENT_NAME} in namespace ${NAMESPACE}"
    kubectl -n "${NAMESPACE}" patch deployment "${DEPLOYMENT_NAME}" \
        --patch-file "${PATCH_FILE}"

    # Restart unconditionally. On a reinstall the patch is a no-op, and the
    # kubelet only refreshes an already mounted ConfigMap after its sync period,
    # so without this argocd-server would keep serving the previous version of
    # the extension for up to a minute.
    echo "==> restarting ${DEPLOYMENT_NAME} to pick up the extension"
    kubectl -n "${NAMESPACE}" rollout restart "deployment/${DEPLOYMENT_NAME}"

    echo "==> waiting for ${DEPLOYMENT_NAME} to roll out"
    kubectl -n "${NAMESPACE}" rollout status "deployment/${DEPLOYMENT_NAME}"

    echo
    echo "The extension is installed. Open an application in the Argo CD UI, the"
    echo "Change Revision item should appear in the status panel. A hard reload"
    echo "may be needed, /extensions.js is loaded when the page is rendered."
}

uninstall_extension() {
    check_deployment

    local container volume mount patch
    patch=""

    if container="$(container_index)" && mount="$(volume_mount_index "${container}")"; then
        patch="{\"op\":\"remove\",\"path\":\"/spec/template/spec/containers/${container}/volumeMounts/${mount}\"}"
    fi

    if volume="$(volume_index)"; then
        if [ -n "${patch}" ]; then
            patch="${patch},"
        fi
        patch="${patch}{\"op\":\"remove\",\"path\":\"/spec/template/spec/volumes/${volume}\"}"
    fi

    if [ -n "${patch}" ]; then
        echo "==> removing the extension volume from deployment ${DEPLOYMENT_NAME}"
        kubectl -n "${NAMESPACE}" patch deployment "${DEPLOYMENT_NAME}" \
            --type=json -p "[${patch}]"
        echo "==> waiting for ${DEPLOYMENT_NAME} to roll out"
        kubectl -n "${NAMESPACE}" rollout status "deployment/${DEPLOYMENT_NAME}"
    else
        echo "==> deployment ${DEPLOYMENT_NAME} does not mount the extension, nothing to do"
    fi

    echo "==> deleting ConfigMap ${CONFIGMAP_NAME}"
    kubectl -n "${NAMESPACE}" delete configmap "${CONFIGMAP_NAME}" --ignore-not-found
}

while getopts ":n:uh" opt; do
    case "${opt}" in
        n) NAMESPACE="${OPTARG}" ;;
        u) ACTION="uninstall" ;;
        h) usage; exit 0 ;;
        *) usage >&2; exit 1 ;;
    esac
done

case "${ACTION}" in
    install) install_extension ;;
    uninstall) uninstall_extension ;;
    *) usage >&2; exit 1 ;;
esac

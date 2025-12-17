## Installation

The controller should be installed in the same Kubernetes namespace
cluster that Argo CD is running in (the `argocd` namespace in most
cases).  The provided installation files plug the Monorepo Controller
components in the existing Argo CD installation, reusing its
configuration in the `argocd-cmd-params-cm` and `argocd-cm` configmaps.

Potentially it is possible to run the controller from another
namespace, but it will require extra configuration and currently this
is not supported.


## Method 1: Installing using provided plain manifests

We provide two installation manifests: 
* `install.yaml' - with cluster-wide permissions to watch Application
  manifests in any namespece
* `install-namespaced.yaml` - for a namespaced ArgoCD instance that
  will watch Application only in the ArgoVD namespace.

### Cluster-wide Installation

Apply the manifest in the ArgoCD namespace:

```
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj-labs/argocd-monorepo-controller/refs/heads/stable/manifests/install.yaml
```

!!! warning The installation manifests include `ClusterRoleBinding`
    resources that reference `argocd` namespace. If you are installing
    Argo CD into a different namespace then make sure to update the
    namespace reference in the `install.yaml` file.

### Namespaced Installation

Apply the manifest in the ArgoCD namespace:

```
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj-labs/argocd-monorepo-controller/refs/heads/stable/manifests/install-namespaced.yaml
```

## Method 2: Installing using kustomize

The Monorepo Controller manifests can also be installed using
Kustomize.  You may include the above manifests as a remote resource and
apply additional customizations using Kustomize patches.


For example

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: argocd
resources:
- https://raw.githubusercontent.com/argoproj-labs/argocd-monorepo-controller/refs/heads/stable/manifests/install.yaml
```

## Method 3: Installing using Helm

The Monorepo Controller manifests can also be installed using Helm. 

The helm chart is maintained in the same Git repository as the Monorepo Controller itself
and released together with the Monorepo Controller.

In simple cases it can be installed with the command:

```shell
helm install <RELEASE-NAME> --namespace argocd "quay.io/eugened/argocd-monorepo-controller:<VERSION>"
```

* <RELEASE-NAME> - user selected release name
* <VERSION> - chart version, which is same as version of the application release

In more complex cases one would need to customize the `values.yaml` file. 
See Helm Chart [Documentation](helm.md) for available options.


### Tune Configuration Parameters

Most configuration parameters are are configured using the
`monorepo-cmd-params-cm` configmap. 

The following parameters reuse ArgoCD configuration from ConfigMaps:

From `argocd-cm`:

* timeout.reconciliation

From `argocd-cmd-params-cm`:

* application.namespaces 
* otlp.address
* otlp.insecure
* otlp.headers
* otlp.attrs
* redis.server
* redis.compression
* redis.db
* reposerver.disable.tls 
* reposerver.tls.minversion 
* reposerver.tls.maxversion
* reposerver.tls.ciphers
* reposerver.repo.cache.expiration
* reposerver.default.cache.expiration
* reposerver.max.combined.directory.manifests.size
* reposerver.revision.cache.lock.timeout
* reposerver.enable.git.submodule
* reposerver.git.request.timeout
* reposerver.grpc.max.size
* reposerver.include.hidden.directories

From `argocd-redis`:

* auth

#### Configure the desired log level of Monorepo Controller components.

While this step is optional, we recommend to set the log level
explicitly.  During your first steps with the Argo CD Monorepo
Controller, a more verbose logging may help greatly in troubleshooting
things.

Edit the value of the `controller.log.level` parameter in the
ConfigMap `monorepo-cmd-params-cm`. 

#### Configure namespaces for Application Manifests

In its default configuration Monorepo Controller will look for
Application Manifests in the same namespaces that ArgoCD does. If you
want to use a different list of namespaces (for example to limit load
on the Monorepo Controller, or for troubleshooting purposes) you need
to change the value of the `ARGOCD_APPLICATION_NAMESPACES` environment
variable of the Monoripo Controller.

## Configuring notifications

See sample triggers and templates in samples/notifications.






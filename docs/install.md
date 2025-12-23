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

!!! warning 
    The installation manifests include `ClusterRoleBinding`
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

The Helm chart is maintained in the same Git repository as the Monorepo Controller itself
and released together with the Monorepo Controller.

In simple cases it can be installed with the command:

```shell
helm install <RELEASE-NAME> --namespace argocd "quay.io/eugened/argocd-monorepo-controller:<VERSION>"
```

* `<RELEASE-NAME>` - user selected release name
* `<VERSION>` - chart version, which is same as the version of the application release

In more complex cases one would need to customize the `values.yaml` file. 
See Helm Chart [Documentation](helm.md) for available options.


## Updating ApplicationSet Controller Configuration

By default, the ApplicationSet controller will remove any annotations
added by the Monorepo Controller. This triggers the Monorepo
Controller to immediately attempt to restore them, leading to a
"conflict loop" between the two controllers.

To prevent this, you must configure the ApplicationSet controller to
ignore specific annotations. It is recommended to do this at the
global level using the
`applicationsetcontroller.global.preserved.annotations` parameter in
the argocd-cmd-params-cm ConfigMap:

```yaml
data:
  applicationsetcontroller.global.preserved.annotations: "mrp-controller.argoproj.io/change-revision,mrp-controller.argoproj.io/change-revisions,mrp-controller.argoproj.io/git-revision,mrp-controller.argoproj.io/git-revisions"
```

If you use Helm to install ArgoCD, you should configure this in your `values.yaml` file under the `configs.params` section:

```yaml
configs:
  params:
    applicationsetcontroller.global.preserved.annotations: "mrp-controller.argoproj.io/change-revision,mrp-controller.argoproj.io/change-revisions,mrp-controller.argoproj.io/git-revision,mrp-controller.argoproj.io/git-revisions"
```

While you can also configure this for specific ApplicationSets using
the `preservedFields` property, a global configuration is generally
better. Without a global setting, any ApplicationSet that lacks these
`preservedFields` property,but utilizes the manifest-generate-path
annotation, will cause the controllers to enter a continuous add/delete
loop.


For more details, see the ArgoCD documentation: [Preserving changes made to an Applications annotations and labels](https://argo-cd.readthedocs.io/en/stable/operator-manual/applicationset/Controlling-Resource-Modification/#preserving-changes-made-to-an-applications-annotations-and-labels).

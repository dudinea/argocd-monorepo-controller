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
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj-labs/argocd-monorepo-controller/refs/heads/main/manifests/install.yaml
```

!!! warning The installation manifests include `ClusterRoleBinding`
    resources that reference `argocd` namespace. If you are installing
    Argo CD into a different namespace then make sure to update the
    namespace reference in the `install.yaml` file.

### Namespaced Installation

Apply the manifest in the ArgoCD namespace:

```
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj-labs/argocd-monorepo-controller/refs/heads/main/manifests/install-namespaced.yaml
```

### Tune Configuration Parameters

Most configuration parameters are are configured using the
`monorepo-cmd-params-cm` configmap. The following parameters
reuse ArgoCD configuration from the `argocd-cmd-params-cm`:

* application.namespaces - list of namespaces to watch ApplicationResources
* 

#### Configure the desired log level of Monorepo Controller components.

While this step is optional, we recommend to set the log level
explicitly.  During your first steps with the Argo CD Monorepo
Controller, a more verbose logging may help greatly in troubleshooting
things.

Edit the value of the `controller.log.level` parameter in the
ConfigMap `monorepo-cmd-params-cm`. 

#### Configure namespaces for Application Manifests

In its default configuration Monorepo Controller will look for
Application Manifests in the same namespaces, as ArgoCD is configured
to look. If you want to use a different list of namespaces (for
example to limit load on the Monorepo Controller, or for
troubleshooting purposes) you need to change the value of the
`ARGOCD_APPLICATION_NAMESPACES` environment variable of the Monoripo
Controller.

#### Configure Other Parameters



### Set up Prometheus Metrics Collection

If you have Prometheus installed on your cluster you may install 
ServiceMonitor manifests for the Monorepo Controller manifests:

```
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj-labs/argocd-monorepo-controller/refs/heads/main/manifests/install-metrics-collection.yaml
```



### Disable 


Or use `kustomize` to install kustomization from
https://github.com/argoproj-labs/argocd-monorepo-controller/tree/main/manifests

## Configuring notifications

See sample triggers and templates in samples/notifications.


## Development 

The project is based on essencially the same Makefile and other 
Argocd infrastructure, so Argocd Developer Documentation 
can be currently used.

One quick way to build and run it locally is:

```
kubectl config set-context --current --namespace=argocd   # set current context to the argocd namespace
make cli-local                                            # build the program binary
make run                                                  # uses goreman to both monorepo controller and its repo-server
```


## Community

 You can reach the developers via the following channels:

* Q & A : [Github Discussions](https://github.com/argoproj-labs/argocd-monorepo-controller/discussions)  [TBD]
* Chat : [The monorepo-controller Slack channel](https://argoproj.github.io/community/join-slack)  [TBD]
* [Github Issues](https://github.com/argoproj-labs/argocd-monorepo-controller/issues)


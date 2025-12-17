# Monorepo Controller Configuration

The configuration parameters, which are specific to the Monorepo Controler
as well as those, that have to be configured separately from the rest of ArgoCD
components, are configured using the `argocd-monorepo-cmd-params-cm` configmap. 

| Key                                    | Default                           | Description                                                        |
|----------------------------------------|-----------------------------------|--------------------------------------------------------------------|
| controller.log.level                   | "info"                            | Monorepo Controller log level                                      |
| controller.log.format                  | "text"                            | Monorepo Controller log format (text or json)                      |
| controller.metrics.cache.expiration    | disabled by default               | Prometheus metrics cache expiration                                |
| controller.metrics.address             | "0.0.0.0"                         | Controller's Metrics server will listen on given address           |
| controller.metrics.port                | "8090"                            | Controller's Metrics server will listen on given port              |
| controller.repo.server                 | argocd-monorepo-repo-server:8091" | Monorepo Repo server address                                       |
| controller.repo.server.plaintext       | "false"                           | Use a non-TLS client to connect to repository  server              |
| controller.repo.server.strict.tls      | "false"                           | Perform strict validation of monorepo repo server TLS certificates |
| controller.repo.server.timeout.seconds | "60"                              | Repo server RPC call timeout seconds.                              |
| reposerver.log.level                   | "info"                            | Monorepo Controller log level                                      |
| reposerver.log.format                  | "text"                            | Monorepo Repo Server log format (text or json)                     |
| reposerver.parallelism.limit           | "0" - no limit                    | Limit on number of concurrent manifests generate requests.         |
| reposerver.listen.address              | "0.0.0.0"                         | Repo Server will listen on given address for incoming connections  |
|                                        |                                   |                                                                    |


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

## Configure the desired log level of Monorepo Controller components.

While this step is optional, we recommend to set the log level
explicitly.  During your first steps with the Argo CD Monorepo
Controller, a more verbose logging may help greatly in troubleshooting
things.

Edit the value of the `controller.log.level` parameter in the
ConfigMap `argocd-monorepo-cmd-params-cm`. 

## Configure namespaces for Application Manifests

In its default configuration Monorepo Controller will look for
Application Manifests in the same namespaces that ArgoCD does. If you
want to use a different list of namespaces (for example to limit load
on the Monorepo Controller, or for troubleshooting purposes) you need
to change the value of the `ARGOCD_APPLICATION_NAMESPACES` environment
variable of the Monoripo Controller.

## Configuring network policies

The Monorepo Controller installation using kustomize/manifests 
contains Metwork Policies manifest for it's components. 

In the case when your Kubernetes cluster supports Network Policies, but 
your ArgoCD installation does not use network policies, 
you must disable them in the manifest or in the kustomize 
`kustomization.yaml` files.

The Helm Chart, however, has the Network Policies disabled by default 
(in line with the ArgoCD Helm Chart default). If your ArgoCD installation 
uses Network Policies, you should enable Network Policies installation
using the `` key in your values.yaml.


## Configuring notifications

See sample triggers and templates in samples/notifications.

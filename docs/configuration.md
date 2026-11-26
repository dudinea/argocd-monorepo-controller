# Monorepo Controller Configuration

The configuration parameters, which are specific to the Monorepo Controler
as well as those, that have to be configured separately from the rest of ArgoCD
components, are configured using the `argocd-monorepo-cmd-params-cm` configmap. 

| Key                                       | Default                           | Description                                                        |
|-------------------------------------------|-----------------------------------|--------------------------------------------------------------------|
| controller.app.resync.period              | disabled by default               | Interval of handling every application again without an event      |
| controller.event.handling.timeout.seconds | "120"                             | Context deadline for processing a single application event         |
| controller.log.level                      | "info"                            | Monorepo Controller log level                                      |
| controller.log.format                     | "text"                            | Monorepo Controller log format (text or json)                      |
| controller.metrics.cache.expiration       | disabled by default               | Prometheus metrics cache expiration                                |
| controller.metrics.address                | "0.0.0.0"                         | Controller's Metrics server will listen on given address           |
| controller.metrics.port                   | "8090"                            | Controller's Metrics server will listen on given port              |
| controller.repo.server                    | argocd-monorepo-repo-server:8091" | Monorepo Repo server address                                       |
| controller.repo.server.plaintext          | "false"                           | Use a non-TLS client to connect to repository  server              |
| controller.repo.server.strict.tls         | "false"                           | Perform strict validation of monorepo repo server TLS certificates |
| controller.repo.server.timeout.seconds    | "120"                             | Repo server RPC call timeout seconds.                              |
| controller.retry.backoff.factor           | "2.0"                             | Multiplier applied to the retry delay after every attempt          |
| controller.retry.base.delay               | "20s"                             | Delay before the first retry of a failed application               |
| controller.retry.jitter                   | "0.4"                             | Fraction by which the retry delay is randomized                    |
| controller.retry.max.attempts             | "3"                               | Retries of a failed application, "0" disables the retries          |
| controller.retry.max.delay                | "15m"                             | Upper bound of the retry delay, "0" leaves it uncapped             |
| controller.workers                        | "3"                               | Number of workers processing application events                    |
| reposerver.log.level                      | "info"                            | Monorepo Controller log level                                      |
| reposerver.log.format                     | "text"                            | Monorepo Repo Server log format (text or json)                     |
| reposerver.parallelism.limit              | "0" - no limit                    | Limit on number of concurrent manifests generate requests.         |
| reposerver.listen.address                 | "0.0.0.0"                         | Repo Server will listen on given address for incoming connections  |
| reposerver.use.list.rev.cache             | "true"                            | Enable List Revisions requests cache                               | 
| reposerver.use.diff.tree.cache            | "true"                            | Enable Diff Tree requests cache                                    |
| reposerver.incremental.fetch              | "false"                           | Fetch the history of a repository in portions                      |
| reposerver.incremental.fetch.initial.depth | "50"                             | Commits the first fetched portion of the history holds             |
| reposerver.incremental.fetch.batch.depth  | "200"                             | Commits every further fetched portion adds                         |
| reposerver.incremental.fetch.max.depth    | "5000"                            | Commits of an application history the search may examine, "0" is no limit |
| reposerver.incremental.fetch.budget       | "1m"                              | Time one incremental search may spend fetching and walking         |
| reposerver.incremental.fetch.on.exhaustion | "full-fetch"                     | What an incremental search that ran out of budget does             |
|-------------------------------------------|-----------------------------------|--------------------------------------------------------------------|

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

## Configure event handling timeout

The event handling timeout controls how long the controller will wait for a single
application event to be processed before timing out. The default is 2 minutes.

If your applications have complex manifest generation or large Git repositories,
you may need to increase this timeout. Conversely, you can decrease it if you
want faster timeout on problematic events.

Edit the value of the `controller.event.handling.timeout` parameter in the
ConfigMap `argocd-monorepo-cmd-params-cm`. The value should be a valid Go duration
string (e.g., `5m`, `30s`, `1h`).

**Example:**
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-monorepo-cmd-params-cm
data:
  controller.event.handling.timeout: "5m"
```

## Configure the application resync period

A failing application is retried a bounded number of times. Once those
attempts are used up the application is left alone until its next event, so
its [Change Revision](terminology.md#change-revision) annotations stay stale
for as long as nothing happens to it. On a quiet installation that can be a
long time.

The resync period closes that gap: the controller asks the Kubernetes
informer to redeliver every application it watches at that interval, and
handles each of them again. An application whose annotations are up to date
costs almost nothing to handle again, because the controller compares the
recorded Git revision with the current one and stops there without calling
the monorepo repo server. The applications which never got their revisions
calculated are the ones that do real work.

The resync is disabled by default. Set `controller.app.resync.period` to a
Go duration string of at least one minute to enable it; something well above
the time a sweep of all your applications takes, such as `1h`, is a
reasonable starting point. A redelivered application which is already
waiting for a retry, or which was just handled, is recognized as unchanged:
it neither restarts the retry backoff nor hands the application a new budget
of attempts.

**Example:**
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-monorepo-cmd-params-cm
data:
  controller.app.resync.period: "1h"
```

## Configure the incremental fetch

Calculating a [Change Revision](terminology.md#change-revision) needs the
commits between the revision an application was synchronized to before and the
one it is synchronized to now. The monorepo repo server fetches the repository
to get them, and for a large monorepo that fetch of the complete history does
not finish within the timeout of a git command:

```
git fetch origin --tags --force --prune failed fatal timeout after 1m40s
```

The incremental fetch replaces the one fetch of everything with a history that
grows in portions. The repo server fetches the newest
`reposerver.incremental.fetch.initial.depth` commits, looks for the commit that
changed the application in them, and fetches
`reposerver.incremental.fetch.batch.depth` commits more only if the answer is
not there yet. Most of the time the first portion is enough, because the commit
that changed an application is usually close to the tip of the branch, and an
application whose paths nothing touches keeps the revisions it was annotated
with earlier instead of being calculated again.

The history a search fetches stays in the local repository, so the next request
for the same repository, including a retry of a request that ran out of its
budget, starts from the depth the previous one reached. The changed files of
every commit are cached, so examining a grown portion of the history only pays
for the commits it added.

The depth applies to every branch and tag of the repository, counted from its
own tip, because the repo server fetches all of them, the way a fetch of the
complete history does. A portion of N commits therefore holds about N commits
of every branch, and a branch or tag that was created since the last fetch is
bounded by the same depth instead of arriving with its complete history.

The feature is disabled by default. Enable it for installations whose
repositories are too large to fetch at once:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-monorepo-cmd-params-cm
data:
  reposerver.incremental.fetch: "true"
```

A search gives up when it has examined `reposerver.incremental.fetch.max.depth`
commits of the history of the application, or when it has spent
`reposerver.incremental.fetch.budget` on fetching and walking. Keep the budget
below `controller.repo.server.timeout.seconds`, which bounds the whole request.
`reposerver.incremental.fetch.on.exhaustion` then selects what happens:

| Value              | Behavior                                                                                               |
|--------------------|--------------------------------------------------------------------------------------------------------|
| `full-fetch`       | Fetch the complete history and calculate the change revision as a disabled incremental fetch would. The default. |
| `error`            | Fail the request. The controller retries it, and the retry resumes from the depth this search reached, which is the value for a repository whose complete history cannot be fetched at all. |
| `current-revision` | Report the current revision as the change revision, which is what the calculation does anyway when it finds no path between the two revisions. |

Watch `monorepo_incremental_fetch_batches`: a value above one means that
`reposerver.incremental.fetch.initial.depth` is too small for your
applications. `monorepo_incremental_fetch_exhausted_total` counts the searches
that ran out of their budget.

Two limits are worth knowing about. While the previous revision is not among
the commits that have been fetched, or is not connected to the current revision
in them, the search walks the history of the current revision by commit date
instead of restricting it to the commits on a path between the two revisions,
as the calculation of the complete history does. A
commit that the previous revision already contains can only be reached first if
its committer date is more recent, which takes a skewed clock or a rewritten
history.

The second limit is that a local copy which already holds the complete history,
which is what a repo server that ran with the feature disabled leaves behind,
is never truncated to a window: fetching it again only transfers the commits
that are new anyway. The copies live under `/tmp/_argocd-repo`, which the
shipped manifests mount as an `emptyDir`, so restarting the monorepo repo
server is enough for the portioned fetch to take effect. An installation that
keeps that directory on a persistent volume keeps its complete histories until
they are removed. Turning the feature off again restores the complete history
of a truncated copy on the next request for it.

## Configuring network policies

The Monorepo Controller installation using kustomize/manifests 
contains Metwork Policies manifest for it's components. 

In the case when your Kubernetes cluster supports Network Policies, but 
your ArgoCD installation does not use network policies, 
you must disable them in the manifest or in the kustomize 
`kustomization.yaml` files.

The Helm Chart, however, has the Network Policies disabled by default
(in line with the ArgoCD Helm Chart default). If your ArgoCD
installation uses Network Policies, you should enable Network Policies
installation using the `global.networkPolicy.create: true` parameter
in your values.yaml.


## Configuring notifications

See sample triggers and templates in samples/notifications.

## Configuring Failures Handling

Handling an application can fail for various reasons that have nothing
to do with the application itself.  The controller classifies the
error and, when the operation may succeed later, puts the application
back into its queue and handles it again after a delay.

The delay grows exponentially with every attempt: it starts at
`controller.retry.base.delay`, is multiplied by
`controller.retry.backoff.factor` after every attempt and never
exceeds `controller.retry.max.delay`. A random fraction of up to
`controller.retry.jitter` is then added or subtracted.

An application is retried at most `controller.retry.max.attempts` times.
After that the failure is logged at the error level and the application is
left alone until its next event. Setting the parameter to `"0"` disables the
retries. See [Configuration](configuration.md) for the parameters and
[Metrics](metrics.md) for the metrics the retries are observable through.

An event which brings a new version of an application clears the retry count.

For multisource applications the sources are calculated independently,
so a failure of one source does not discard the work done for the
others: the annotations are patched with the [Change
Revisions](terminology.md#change-revision) that were calculated, and
the `mrp-controller.argoproj.io/git-revision(s)` entries of the failed
sources are left at their previous values, so the retry recalculates
exactly the sources which failed.

An application whose retries were all used up is only looked at again
when its next event arrives. The optional application resync period
asks the informer to redeliver applications after a fixed interval.
The resync is disabled by default becaise most Argo CD installations
have non-zero `timeout.reconciliation` configured, which causes
periodic reconciliation of applications, which delivers application
updates to monorepo controller as well.

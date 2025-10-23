# ArgoCD Monorepo Controller Architecture

## Tracking Change Revisions with Monorepo.

Currently, in Monorepo configurations, ArgoCD cannot accurately track
Change Revisions: application state and history contains commits for
the entire Git repository, not specifically the Change Revisions that
are relevant for the specific Application.

This has a lot of undesirable consequences from the users point of view:

* Bogus notifications on changes that are not relevant for the user's
  Application.
* Not-relevant entries in application history and timeline.
* Unneeded Application Synchronizations with no change in the manifests.

In large monorepo configurations, when there are hundreds of
applications sharing same Git repository, most of history entries and
notifications are bogus and do not contain relevant information for
Application users.

## The Monorepo Controller 

The Monorepo Controller is an add-on component for ArgoCD that
accurately tracks last commits that actually changed the application
(Change Revision).

It is a Kubernetes controller which continuously monitors running
applications and handles changes  in their Sync state: which Git commit
it is synchronized to.

Putting it in a separate controller gives a quick solution to for
tracking Change Revisions to monorepo users users without affecting
negatively ArgoCD performance or development.

This will as well allow us to use this controller as sort of
playground for finding optimal ways for accurate tracking of
application changes, so in the future will be able more easily
incorporate this functionality into Argo-CD itself.

## Controller functionality

The controller will listen to Application events (creation and change
in the application manifests) and it will update the following
annotations to to indicate actual Change Revisions of the
applications.

* `mrp-controller.argoproj.io/change-revision` - Contains Application Change Revision.
* `mrp-controller.argoproj.io/change-revisions` - Contains a JSON List of Change
  Revisions for each application source according to the order of Applicaton sources.
* `mrp-controller.argoproj.io/git-revision` - Contains Application Git Revision.
* `mrp-controller.argoproj.io/git-revisions` - Contains a JSON list of
  Git Revision for each application source according to the order of
  Applicaton sources.

The controller saves in the
`mrp-controller.argoproj.io/git-revision(s)` annotations current Git
revision of the application source, that allows it to avoid expensive
recalculation of the change revision when there is no actual change in
Git repository.

For multisource applications the `mrp-controller.argoproj.io/change-revision` 
and `mrp-controller.argoproj.io/git-revision` annotations contain the values
for the first application source only.

For single source applications the `mrp-controller.argoproj.io/change-revisions`
and `mrp-controller.argoproj.io/git-revisions`  contain lists of one element 
with the data of the only application source.

## The Manifest Paths Annotation

This [`argocd.argoproj.io/manifest-generate-paths`](https://argo-cd.readthedocs.io/en/latest/operator-manual/high_availability/#manifest-paths-annotation)
Application annotation specifies which paths whithin the Git 
repository are used during manifest generation. Use of
this annotation is used by ArgoCD to avoid unnecessary
regeneration of Application manifests when it is known that 
they  won't be affected by the changes in the commit.

In Argo CD usage of this annotation is optional and only affects
synchronization performance and load on Git repositories.

The Monorepo Controller, however, uses this annotation to distinguish
between the changes that would affect and the application manifests
and those that wouldn't. In its current implementation the Monorepo
Controller won't handle the application that do not have 
this annotation set. 

Therefore, setting this annotation correctly is critical
for accurate tracking of Application Change Revisions
by ArgoCD Monorepo Controller.

## The Initial Implementation and its Limitations

The initial controller version will only work for applications that
have the `argocd.argoproj.io/manifest-generate-paths` annotation
defined.  It will get list of revisions between currently synchronizing
(or last synchronized) git revision and previously synchronized 
git revision. Then it will use git `diff-tree` operation to each
revision in the list to determine if there are changed files and
will filter that list against the paths from the
`manifest-generate-paths` annotation.

Currently the ArgoCD Monorepo Controller consists of two components that 
are running in separate pods:

* argocd-monorepo-controller
* argocd-monorepo-repo-server

The former one is architecturally parallel to the ArgoCD application controller:
it listens to application events and updates application annotations.

The latter one is parallel to ArgoCD Repo Server: the former component
calls it to perform the actual calculation of Change Revision, which
includes actual checkout of Git repositories and running the `git
diff` operation.

Both components are supposed to run in the ArgoCD namespace and
reuse relevant ArgoCD configuration (such as configuration of Git repository connections,
list of namespaces to handle applications, etc.).


## Future plans

* Make an Argo-CD UI extension to display accurate Change
  Revision in the UI
* We'll try to introduce a more presize method for calculating the
  affected version that will run CM tools/plugins to determine if
  there was a change in manifests.
* Look into possible performance optimizations of the operations.

As the long-term goal, when the things will stabilize and performance
will look good we'll start working on integrating required changes
into the upstream repository server as well as on extending properly
the Application CR and integrating new functionality into the upstream
Application server.











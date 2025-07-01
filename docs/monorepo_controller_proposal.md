
# Summary

This proposal attempts to find a way solve the long-standing issue
with ArgoCD creating irrelevant history records and last commit information
when several Applications are looking at different paths at the same
repository/branch.

# Motivation

Multiple Issues has been opened that are related to this problem:

* btxbtx [Notifications should only be sent for apps that have actually changed. #12169](https://github.com/argoproj/argo-cd/issues/12169)
* Pasha Kostohryz [ArgoCD generates irrelevant history records/sync results in monorepo setups during manual sync. #17280](https://github.com/argoproj/argo-cd/issues/17280)
* Andrii Korotkov [Show an info about the last commit(s) which actually affects a given application #20592](https://github.com/argoproj/argo-cd/issues/20592)
* Andrii Korotkov [Hide author and comment for PRs under Sync Status and Last Sync or give such an option #20586](https://github.com/argoproj/argo-cd/issues/20586)

The main issues from the user's point of view seem to be display of
irrelevant last commit information in the CR / GUI (commit that
actually updated another application watching the same repository) as
well as getting excessive change notifications on applications that
really were not changed.

# Proposal

We propose to create a separate Application Change Revision (ACR)
Controller (as a argoproj-lab project) that will accurately track last
commits that actually changed the application (Change Revision).

Putting it in a separate controller will give a quick solution to
affected users without affecting ArgoCD itself development and
performance wise.

This will as well allow us to use this controller as sort of
playground for finding optimal ways for accurate tracking of
application changes, so in the future will be able more easily
incorporate this functionality into Argo-CD itself.

Such a controller already exists as part of the Codefresh product
(written by Pasha Kostohrys). It will be modified to be able to work
with the upstream Argo-CD.

## Controller functionality

The controller will listen to Application events and it will use
annotations to to indicate actual Change Revision of the application.

* `mrp-controller.argoproj.io/change-revision` - Application change revision
* `mrp-controller.argoproj.io/change-revisions` - List of change
  revisions for multisource applications.

## Initial implementation

The initial controller version will only work for applications that
have the `argocd.argoproj.io/manifest-generate-paths` annotation
defined.  It will get list of revisions between currently (or last)
synced git revision, it will use git `diff-tree` operation to on each
revision against each version determine if there are changed files and
will filter that list against the paths from the
`manifest-generate-paths` annotation.

For repository access it will contain "Lightweight Repo Server" using
relevant parts of ArgoCD and reusing it's configuration. 

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


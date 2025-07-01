# Argo Monorepo Controller

## What is Argo Monorepo Controller?

This controller os an ArgoCD addon that accurately tracks last commits
that actually changed the application (Change Revision). It is 
usefull when several Applications are looking at different paths at the same
repository/branch (monorepos). 

## Documentation

Please see [Proposal](https://github.com/argoproj-labs/argocd-monorepo-controller/docs/monorepo_controller_proposal.md) for project motivation, architecture 
and description the program functionality.

## What is its development status?

This is a newly created Argoproj-Labs project. It is WIP and still is
not production ready, does not have working tests, CI, release
process, etc.

_USE AT YOUR OWN RISK!_

## Installation

The controller should be installed into the namespace of an
existing ArgoCD instance (the `argocd` namespace  in most cases).

One quick way to try it is to use command like this:
```
kubectl apply -n argocd -f https://github.com/argoproj-labs/argocd-monorepo-controller/manifests/install.yaml
```

Or use `kustomize` to install kustomization from [Proposal](https://github.com/argoproj-labs/argocd-monorepo-controller/manifests/base).

(Note: referenced container images aren't yet unavailable)


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


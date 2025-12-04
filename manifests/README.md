# Argo Monorepo Controller Installation Manifests

Four sets of installation manifests are provided:

## Normal Installation:

* [install.yaml](install.yaml) - For standard Argo CD installations with
  cluster-admin access, when Application manifests can be located in
  any namespace in the local cluster.
  
* [namespace-install.yaml](namespace-install.yaml) - For installation of Argo CD which have namespace-only privileges 
  (does not need cluster roles). 


package common

// Default listener ports for ArgoCD components
const (
	DefaultPortMRPServerMetrics          = 8090
	DefaultPortMonorepoRepoServer        = 8091
	DefaultPortMonorepoRepoServerMetrics = 8094
)

// DefaultAddressAPIServer for ArgoCD components
const (
	DefaultAddressMRPControllerMetrics = "0.0.0.0"

	DefaultAddressMonorepoRepoServerMetrics = "0.0.0.0"
	DefaultAddressMonorepoRepoServer        = "0.0.0.0"
)

// Default service addresses and URLS of Argo CD internal services
const (
	// DefaultRepoServerAddr is the gRPC address of the Argo CD repo server
	DefaultMonorepoRepoServerAddr = "argocd-monorepo-repo-server:8091"
)

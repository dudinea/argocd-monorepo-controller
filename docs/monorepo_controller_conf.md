The ArgoCD Monorepo Controller is a service that listens for application events and updates the application's revision in the application CRD


**Usage:**

* argocd-monorepo-controller [flags]
* argocd-monorepo-controller [command]

**Available Commands:**

* completion  Generate the autocompletion script for the specified shell
* help        Help about any command
* version     Print version information

**Flags:**


| Argument                                | Type        | Environment Variable                         | Description                                                  |
| --------------------------------------- | ----------- | -------------------------------------------- | ------------------------------------------------------------ |
| --address                               | string      | MONOREPO_CONTROLLER_LISTEN_ADDRESS           | Metrics server will listen on given address (default         |
|                                         |             |                                              | "0.0.0.0")                                                   |
| --application-namespaces                | strings     | MONOREPO_CONTROLLER_APPLICATION_NAMESPACES   | Comma separated list of additional namespaces where          |
|                                         |             |                                              | application resources can be managed in                      |
| --as                                    | string      |                                              | Username to impersonate for the operation                    |
| --as-group                              | stringArray |                                              | Group to impersonate for the operation, this flag can be     |
|                                         |             |                                              | repeated to specify multiple groups.                         |
| --as-uid                                | string      |                                              | UID to impersonate for the operation                         |
| --certificate-authority                 | string      |                                              | Path to a cert file for the certificate authority            |
| --client-certificate                    | string      |                                              | Path to a client certificate file for TLS                    |
| --client-key                            | string      |                                              | Path to a client key file for TLS                            |
| --cluster                               | string      |                                              | The name of the kubeconfig cluster to use                    |
| --context                               | string      |                                              | The name of the kubeconfig context to use                    |
| --disable-compression                   |             |                                              | If true, opt-out of response compression for all requests to |
|                                         |             |                                              | the server                                                   |
| --gloglevel                             | int         |                                              | Set the glog logging level                                   |
| --help -h,                              |             |                                              | help for argocd-monorepo-controller                          |
| --insecure-skip-tls-verify              |             |                                              | If true, the server's certificate will not be checked for    |
|                                         |             |                                              | validity. This will make your HTTPS connections insecure     |
| --kubeconfig                            | string      |                                              | Path to a kube config. Only required if out-of-cluster       |
| --logformat                             | string      | MONOREPO_CONTROLLER_LOGFORMAT                | Set the logging format. One of: text,json (default "json")   |
| --loglevel                              | string      | MONOREPO_CONTROLLER_LOGLEVEL                 | Set the logging level. One of: debug,info,warn,error         |
|                                         |             |                                              | (default "info")                                             |
| --metrics-cache-expiration              | duration    | MONOREPO_CONTROLLER_METRICS_CACHE_EXPIRATION | Prometheus metrics cache expiration (disabled  by default.   |
|                                         |             |                                              | e.g. 24h0m0s)                                                |
| --monorepo-repo-server                  | string      | MONOREPO_REPO_SERVER                         | Monorepo Repo server address (default "argocd-monorepo-repo- |
|                                         |             |                                              | server:8091")                                                |
| --monorepo-repo-server-plaintext        |             | MONOREPO_REPO_SERVER_PLAINTEXT               | Use a plaintext client (non-TLS) to connect to repository    |
|                                         |             |                                              | server                                                       |
| --monorepo-repo-server-strict-tls       |             | MONOREPO_REPO_SERVER_STRICT_TLS              | Perform strict validation of TLS certificates when           |
|                                         |             |                                              | connecting to monorepo repo server                           |
| --monorepo-repo-server-timeout-seconds  | int         | MONOREPO_REPO_SERVER_TIMEOUT_SECONDS         | Repo server RPC call timeout seconds. (default 60)           |
| --namespace -n,                         | string      |                                              | If present, the namespace scope for this CLI request         |
| --password                              | string      |                                              | Password for basic authentication to the API server          |
| --port                                  | int         | MONOREPO_CONTROLLER_LISTEN_PORT              | Metrics server will listen on given port (default 8090)      |
| --proxy-url                             | string      |                                              | If provided, this URL will be used to connect via proxy      |
| --request-timeout                       | string      |                                              | The length of time to wait before giving up on a single      |
|                                         |             |                                              | server request. Non-zero values should contain a             |
|                                         |             |                                              | corresponding time unit (e.g. 1s, 2m, 3h). A value of zero   |
|                                         |             |                                              | means don't timeout requests. (default "0")                  |
| --server                                | string      |                                              | The address and port of the Kubernetes API server            |
| --tls-server-name                       | string      |                                              | If provided, this name will be used to validate server       |
|                                         |             |                                              | certificate. If this is not provided, hostname used to       |
|                                         |             |                                              | contact the server is used.                                  |
| --token                                 | string      |                                              | Bearer token for authentication to the API server            |
| --user                                  | string      |                                              | The name of the kubeconfig user to use                       |
| --username                              | string      |                                              | Username for basic authentication to the API server          |

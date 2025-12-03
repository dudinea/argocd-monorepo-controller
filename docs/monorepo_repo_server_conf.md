Monorepo Repository Server is an internal service that Monorepo Controller uses for Repository access.  This command runs Monorepo Repository Server in the foreground.  It can be configured by following options.


**Usage:**

* argocd-repo-server [flags]
* argocd-repo-server [command]

**Available Commands:**

* completion  Generate the autocompletion script for the specified shell
* help        Help about any command
* version     Print version information

**Flags:**


| Argument                                | Type        | Environment Variable                         | Description                                                  |
| --------------------------------------- | ----------- | -------------------------------------------- | ------------------------------------------------------------ |
| --address                               | string      | ARGOCD_MONOREPO_REPO_SERVER_LISTEN_ADDRESS   | Listen on given address for incoming connections (default    |
|                                         |             |                                              | "0.0.0.0")                                                   |
| --allow-oob-symlinks                    |             | ARGOCD_REPO_SERVER_ALLOW_OUT_OF_BOUNDS_SYMLINKS | Allow out-of-bounds symlinks in repositories (not            |
|                                         |             |                                              | recommended)                                                 |
| --default-cache-expiration              | duration    | ARGOCD_DEFAULT_CACHE_EXPIRATION              | Cache expiration default (default 24h0m0s)                   |
| --disable-helm-manifest-max-extracted-size |             | ARGOCD_REPO_SERVER_DISABLE_HELM_MANIFEST_MAX_EXTRACTED_SIZE | Disable maximum size of helm manifest archives when          |
|                                         |             |                                              | extracted                                                    |
| --disable-oci-manifest-max-extracted-size |             | ARGOCD_REPO_SERVER_DISABLE_OCI_MANIFEST_MAX_EXTRACTED_SIZE | Disable maximum size of oci manifest archives when extracted |
| --disable-tls                           |             | ARGOCD_REPO_SERVER_DISABLE_TLS               | Disable TLS on the gRPC endpoint                             |
| --helm-manifest-max-extracted-size      | string      | ARGOCD_REPO_SERVER_HELM_MANIFEST_MAX_EXTRACTED_SIZE | Maximum size of helm manifest archives when extracted        |
|                                         |             |                                              | (default "1G")                                               |
| --helm-registry-max-index-size          | string      | ARGOCD_REPO_SERVER_HELM_MANIFEST_MAX_INDEX_SIZE | Maximum size of registry index file (default "1G")           |
| --help -h,                              |             |                                              | help for argocd-repo-server                                  |
| --include-hidden-directories            |             | ARGOCD_REPO_SERVER_INCLUDE_HIDDEN_DIRECTORIES | Include hidden directories from Git                          |
| --logformat                             | string      | ARGOCD_REPO_SERVER_LOGFORMAT                 | Set the logging format. One of: json,text (default "json")   |
| --loglevel                              | string      | ARGOCD_REPO_SERVER_LOGLEVEL                  | Set the logging level. One of: debug,info,warn,error         |
|                                         |             |                                              | (default "info")                                             |
| --max-combined-directory-manifests-size | strin       | ARGOCD_REPO_SERVER_MAX_COMBINED_DIRECTORY_MANIFESTS_SIZE | g   Max combined size of manifest files in a directory-type  |
|                                         |             |                                              | Application (default "10M")                                  |
| --metrics-address                       | string      | ARGOCD_MONOREPO_REPO_SERVER_METRICS_LISTEN_ADDRESS | Listen on given address for metrics (default "0.0.0.0")      |
| --metrics-port                          | int         |                                              | Start metrics server on given port (default 8094)            |
| --monorepo-repo-server-use-cache        |             | ARGOCD_MONOREPO_REPO_SERVER_USE_CACHE        | Use Redis cache (default true)                               |
| --oci-layer-media-types                 | strings     | ARGOCD_REPO_SERVER_OCI_LAYER_MEDIA_TYPES     | Comma separated list of allowed media types for OCI media    |
|                                         |             |                                              | types. This only accounts for media types within layers.     |
|                                         |             |                                              | (default [application/vnd.oci.image.layer.v1.tar,application |
|                                         |             |                                              | /vnd.oci.image.layer.v1.tar+gzip,application/vnd.cncf.helm.c |
|                                         |             |                                              | hart.content.v1.tar+gzip])                                   |
| --oci-manifest-max-extracted-size       | string      | ARGOCD_REPO_SERVER_OCI_MANIFEST_MAX_EXTRACTED_SIZE | Maximum size of oci manifest archives when extracted         |
|                                         |             |                                              | (default "1G")                                               |
| --otlp-address                          | string      | ARGOCD_REPO_SERVER_OTLP_ADDRESS              | OpenTelemetry collector address to send traces to            |
| --otlp-attrs                            | strings     | ARGOCD_REPO_SERVER_OTLP_ATTRS                | List of OpenTelemetry collector extra attrs when send        |
|                                         |             |                                              | traces, each attribute is separated by a colon(e.g.          |
|                                         |             |                                              | key:value)                                                   |
| --otlp-headers                          | stringToString |                                              | List of OpenTelemetry collector extra headers sent with      |
|                                         |             |                                              | traces, headers are comma-separated key-value pairs(e.g.     |
|                                         |             |                                              | key1=value1,key2=value2) (default [])                        |
| --otlp-insecure                         |             | ARGOCD_REPO_SERVER_OTLP_INSECURE             | OpenTelemetry collector insecure mode (default true)         |
| --parallelismlimit                      | int         | ARGOCD_REPO_SERVER_PARALLELISM_LIMIT         | Limit on number of concurrent manifests generate requests.   |
|                                         |             |                                              | Any value less the 1 means no limit.                         |
| --plugin-tar-exclude                    | stringArray | ARGOCD_REPO_SERVER_PLUGIN_TAR_EXCLUSIONS     | Globs to filter when sending tarballs to plugins.            |
| --plugin-use-manifest-generate-paths    |             | ARGOCD_REPO_SERVER_PLUGIN_USE_MANIFEST_GENERATE_PATHS | Pass the resources described in argocd.argoproj.io/manifest- |
|                                         |             |                                              | generate-paths value to the cmpserver to generate the        |
|                                         |             |                                              | application manifests.                                       |
| --port                                  | int         |                                              | Listen on given port for incoming connections (default 8091) |
| --redis                                 | string      |                                              | Redis server hostname and port (e.g. argocd-redis:6379).     |
| --redis-ca-certificate                  | string      |                                              | Path to Redis server CA certificate (e.g.                    |
|                                         |             |                                              | /etc/certs/redis/ca.crt). If not specified, system trusted   |
|                                         |             |                                              | CAs will be used for server certificate validation.          |
| --redis-client-certificate              | string      |                                              | Path to Redis client certificate (e.g.                       |
|                                         |             |                                              | /etc/certs/redis/client.crt).                                |
| --redis-client-key                      | string      |                                              | Path to Redis client key (e.g. /etc/certs/redis/client.crt). |
| --redis-compress                        | string      |                                              | Enable compression for data sent to Redis with the required  |
|                                         |             |                                              | compression algorithm. (possible values: gzip, none)         |
|                                         |             |                                              | (default "gzip")                                             |
| --redis-insecure-skip-tls-verify        |             |                                              | Skip Redis server certificate validation.                    |
| --redis-use-tls                         |             |                                              | Use TLS when connecting to Redis.                            |
| --redisdb                               | int         |                                              | Redis database.                                              |
| --repo-cache-expiration                 | duration    |                                              | Cache expiration for repo state, incl. app lists, app        |
|                                         |             |                                              | details, manifest generation, revision meta-data (default    |
|                                         |             |                                              | 24h0m0s)                                                     |
| --revision-cache-expiration             | duration    |                                              | Cache expiration for cached revision (default 3m0s)          |
| --revision-cache-lock-timeout           | duration    |                                              | Cache TTL for locks to prevent duplicate requests on         |
|                                         |             |                                              | revisions, set to 0 to disable (default 10s)                 |
| --sentinel                              | stringArray |                                              | Redis sentinel hostname and port (e.g. argocd-redis-ha-      |
|                                         |             |                                              | announce-0:6379).                                            |
| --sentinelmaster                        | string      |                                              | Redis sentinel master group name. (default "master")         |
| --streamed-manifest-max-extracted-size  | string      | ARGOCD_REPO_SERVER_STREAMED_MANIFEST_MAX_EXTRACTED_SIZE | Maximum size of streamed manifest archives when extracted    |
|                                         |             |                                              | (default "1G")                                               |
| --streamed-manifest-max-tar-size        | string      | ARGOCD_REPO_SERVER_STREAMED_MANIFEST_MAX_TAR_SIZE | Maximum size of streamed manifest archives (default "100M")  |
| --tlsciphers                            | string      |                                              | The list of acceptable ciphers to be used when establishing  |
|                                         |             |                                              | TLS connections. Use 'list' to list available ciphers.       |
|                                         |             |                                              | (default "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384")            |
| --tlsmaxversion                         | string      |                                              | The maximum SSL/TLS version that is acceptable (one of:      |
|                                         |             |                                              | 1.0,1.1,1.2,1.3) (default "1.3")                             |
| --tlsminversion                         | string      |                                              | The minimum SSL/TLS version that is acceptable (one of:      |
|                                         |             |                                              | 1.0,1.1,1.2,1.3) (default "1.2")                             |

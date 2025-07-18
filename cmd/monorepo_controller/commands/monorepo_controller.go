package commands

import (
	"context"
	"fmt"
	"math"
	"time"

	mrp "github.com/argoproj/argo-cd/v3/mrp_controller"
	cacheutil "github.com/argoproj/argo-cd/v3/util/cache"

	"github.com/argoproj/pkg/stats"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	repoapiclient "github.com/argoproj/argo-cd/v3/reposerver/apiclient"

	cmdutil "github.com/argoproj/argo-cd/v3/cmd/util"
	"github.com/argoproj/argo-cd/v3/common"
	"github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	appclientset "github.com/argoproj/argo-cd/v3/pkg/client/clientset/versioned"
	servercache "github.com/argoproj/argo-cd/v3/server/cache"
	"github.com/argoproj/argo-cd/v3/util/cli"
	"github.com/argoproj/argo-cd/v3/util/env"
	"github.com/argoproj/argo-cd/v3/util/errors"
	"github.com/argoproj/argo-cd/v3/util/kube"
)

const (
	failureRetryCountEnv              = "EVENT_REPORTER_K8S_RETRY_COUNT"
	failureRetryPeriodMilliSecondsEnv = "EVENT_REPORTE_K8S_RETRY_DURATION_MILLISECONDS"
)

var (
	failureRetryCount              = 0
	failureRetryPeriodMilliSeconds = 100
)

func init() {
	failureRetryCount = env.ParseNumFromEnv(failureRetryCountEnv, failureRetryCount, 0, 10)
	failureRetryPeriodMilliSeconds = env.ParseNumFromEnv(failureRetryPeriodMilliSecondsEnv, failureRetryPeriodMilliSeconds, 0, 1000)
}

// NewCommand returns a new instance of an application change revision command
func NewCommand() *cobra.Command {
	var (
		redisClient              *redis.Client
		listenHost               string
		listenPort               int
		glogLevel                int
		clientConfig             clientcmd.ClientConfig
		applicationServerAddress string
		repoServerAddress        string
		repoServerTimeoutSeconds int
		repoServerPlaintext      bool
		repoServerStrictTLS      bool
		cacheSrc                 func() (*servercache.Cache, error)
		contentSecurityPolicy    string
		applicationNamespaces    []string
		argocdToken              string
		rootpath                 string
	)
	command := &cobra.Command{
		Use:               cliName,
		Short:             "Run the ArgoCD Monorepo Controller",
		Long:              "The ArgoCD Monorepo Controller is a service that listens for application events and updates the application's revision in the application CRD",
		DisableAutoGenTag: true,
		Run: func(c *cobra.Command, _ []string) {
			ctx := c.Context()

			vers := common.GetVersion()
			namespace, _, err := clientConfig.Namespace()
			errors.CheckError(err)
			vers.LogStartupInfo(
				"ArgoCD Monorepo Controller",
				map[string]any{
					"namespace": namespace,
					"port":      listenPort,
				},
			)

			cli.SetLogFormat(cmdutil.LogFormat)
			cli.SetLogLevel(cmdutil.LogLevel)
			cli.SetGLogLevel(glogLevel)

			config, err := clientConfig.ClientConfig()
			errors.CheckError(err)
			errors.CheckError(v1alpha1.SetK8SConfigDefaults(config))

			cache, err := cacheSrc()
			errors.CheckError(err)

			kubeclientset := kubernetes.NewForConfigOrDie(config)

			appclientsetConfig, err := clientConfig.ClientConfig()
			errors.CheckError(err)
			errors.CheckError(v1alpha1.SetK8SConfigDefaults(appclientsetConfig))
			config.UserAgent = fmt.Sprintf("argocd-server/%s (%s)", vers.Version, vers.Platform)

			if failureRetryCount > 0 {
				appclientsetConfig = kube.AddFailureRetryWrapper(appclientsetConfig, failureRetryCount, failureRetryPeriodMilliSeconds)
			}
			appClientSet := appclientset.NewForConfigOrDie(appclientsetConfig)

			tlsConfig := repoapiclient.TLSConfiguration{
				DisableTLS:       repoServerPlaintext,
				StrictValidation: repoServerStrictTLS,
			}
			repoClientSet := repoapiclient.NewRepoServerClientset(repoServerAddress,
				repoServerTimeoutSeconds,
				tlsConfig)

			changeRevisionServerOpts := mrp.MRPServerOpts{
				ListenPort:            listenPort,
				ListenHost:            listenHost,
				Namespace:             namespace,
				KubeClientset:         kubeclientset,
				AppClientset:          appClientSet,
				Cache:                 cache,
				RedisClient:           redisClient,
				ApplicationNamespaces: applicationNamespaces,
				RepoClientset:         repoClientSet,
			}

			log.Debug("Starting Monorepo Controller server")
			stats.RegisterStackDumper()
			stats.StartStatsTicker(10 * time.Minute)
			stats.RegisterHeapDumper("memprofile")
			changeRevisionServer := mrp.NewApplicationChangeRevisionServer(ctx, changeRevisionServerOpts)
			log.Debug("Initializing Monorepo Controller server")
			changeRevisionServer.Init(ctx)
			log.Debug("Starting listener")
			lns, err := changeRevisionServer.Listen()
			errors.CheckError(err)
			for {
				log.Debug("Running Monorepo controllerserver")
				ctx, cancel := context.WithCancel(ctx)
				changeRevisionServer.Run(ctx, lns)
				log.Debug("Change revision controller server finished")
				cancel()
			}
		},
	}

	clientConfig = cli.AddKubectlFlagsToCmd(command)
	command.Flags().StringVar(&rootpath, "argocd-server-path", env.StringFromEnv("ARGOCD_SERVER_ROOTPATH", ""), "Used if Argo CD is running behind reverse proxy under subpath different from /")
	command.Flags().StringVar(&cmdutil.LogFormat, "logformat", env.StringFromEnv("ACR_CONTROLLER_LOGFORMAT", "json"), "Set the logging format. One of: text|json")
	command.Flags().StringVar(&cmdutil.LogLevel, "loglevel", env.StringFromEnv("ACR_CONTROLLER_LOG_LEVEL", "info"), "Set the logging level. One of: debug|info|warn|error")
	command.Flags().IntVar(&glogLevel, "gloglevel", 0, "Set the glog logging level")
	command.Flags().StringVar(&applicationServerAddress, "application-server", env.StringFromEnv("ARGOCD_SERVER", common.DefaultApplicationServerAddr), "Application server address")
	command.Flags().StringVar(&argocdToken, "argocd-token", env.StringFromEnv("ARGOCD_TOKEN", ""), "ArgoCD server JWT token")
	command.AddCommand(cli.NewVersionCmd(cliName))
	command.Flags().StringVar(&listenHost, "address", env.StringFromEnv("ACR_CONTROLLER_LISTEN_ADDRESS", common.DefaultAddressMRPController), "Listen on given address")
	command.Flags().IntVar(&listenPort, "port", common.DefaultPortMRPServer, "Listen on given port")
	command.Flags().StringVar(&contentSecurityPolicy, "content-security-policy", env.StringFromEnv("ACR_CONTROLLER_CONTENT_SECURITY_POLICY", "frame-ancestors 'self';"), "Set Content-Security-Policy header in HTTP responses to `value`. To disable, set to \"\".")
	command.Flags().StringSliceVar(&applicationNamespaces, "application-namespaces", env.StringsFromEnv("ARGOCD_APPLICATION_NAMESPACES", []string{}, ","), "List of additional namespaces where application resources can be managed in")
	command.Flags().StringVar(&repoServerAddress, "monorepo-repo-server", env.StringFromEnv("ARGOCD_MONOREPO_REPO_SERVER", common.DefaultMonorepoRepoServerAddr), "Monorepo Repo server address")
	command.Flags().IntVar(&repoServerTimeoutSeconds, "monorepo-repo-server-timeout-seconds", env.ParseNumFromEnv("ARGOCD_MONOREPO_REPO_SERVER_TIMEOUT_SECONDS", 60, 0, math.MaxInt64), "Repo server RPC call timeout seconds.")
	command.Flags().BoolVar(&repoServerPlaintext, "monorepo-repo-server-plaintext", env.ParseBoolFromEnv("ARGOCD_SERVER_REPO_SERVER_PLAINTEXT", false), "Use a plaintext client (non-TLS) to connect to repository server")
	command.Flags().BoolVar(&repoServerStrictTLS, "monorepo-repo-server-strict-tls", env.ParseBoolFromEnv("ARGOCD_SERVER_REPO_SERVER_STRICT_TLS", false), "Perform strict validation of TLS certificates when connecting to monorepo repo server")
	cacheSrc = servercache.AddCacheFlagsToCmd(command, cacheutil.Options{
		OnClientCreated: func(client *redis.Client) {
			redisClient = client
		},
	})
	return command
}

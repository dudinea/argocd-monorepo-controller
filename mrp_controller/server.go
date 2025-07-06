package mrp_controller

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	repoapiclient "github.com/argoproj/argo-cd/v3/reposerver/apiclient"
	"github.com/argoproj/argo-cd/v3/util/db"

	mrp_controller "github.com/argoproj/argo-cd/v3/mrp_controller/controller"

	appclientset "github.com/argoproj/argo-cd/v3/pkg/client/clientset/versioned"
	appinformer "github.com/argoproj/argo-cd/v3/pkg/client/informers/externalversions"

	// applisters "github.com/argoproj/argo-cd/v3/pkg/client/listers/application/v1alpha1"
	servercache "github.com/argoproj/argo-cd/v3/server/cache"
	"github.com/argoproj/argo-cd/v3/util/healthz"
	settings_util "github.com/argoproj/argo-cd/v3/util/settings"
)

var backoff = wait.Backoff{
	Steps:    5,
	Duration: 500 * time.Millisecond,
	Factor:   1.0,
	Jitter:   0.1,
}

type MRPServer struct {
	MRPServerOpts

	// settings *settings_util.ArgoCDSettings
	// log                  *log.Entry
	appInformer cache.SharedIndexInformer
	// appLister            applisters.ApplicationLister
	applicationClientset appclientset.Interface
	db                   db.ArgoDB
	repoClientset        repoapiclient.Clientset
	// stopCh is the channel which when closed, will shutdown the Event Reporter server
	stopCh chan struct{}
	// serviceSet *MRPServerSet
}

type MRPServerSet struct{}

type MRPServerOpts struct {
	ListenPort            int
	ListenHost            string
	Namespace             string
	KubeClientset         kubernetes.Interface
	AppClientset          appclientset.Interface
	Cache                 *servercache.Cache
	RedisClient           *redis.Client
	ApplicationNamespaces []string
	BaseHRef              string
	RootPath              string
	RepoClientset         repoapiclient.Clientset
}

type handlerSwitcher struct {
	handler              http.Handler
	urlToHandler         map[string]http.Handler
	contentTypeToHandler map[string]http.Handler
}

type Listeners struct {
	Main net.Listener
}

func (l *Listeners) Close() error {
	if l.Main != nil {
		if err := l.Main.Close(); err != nil {
			return err
		}
		l.Main = nil
	}
	return nil
}

func (s *handlerSwitcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if urlHandler, ok := s.urlToHandler[r.URL.Path]; ok {
		urlHandler.ServeHTTP(w, r)
	} else if contentHandler, ok := s.contentTypeToHandler[r.Header.Get("content-type")]; ok {
		contentHandler.ServeHTTP(w, r)
	} else {
		s.handler.ServeHTTP(w, r)
	}
}

func (a *MRPServer) healthCheck(_ *http.Request) error {
	return nil
}

// Init starts informers used by the API server
func (a *MRPServer) Init(ctx context.Context) {
	go a.appInformer.Run(ctx.Done())
	// svcSet := newApplicationChangeRevisionServiceSet()
	// a.serviceSet = svcSet
}

func (a *MRPServer) RunController(ctx context.Context) {
	controller := mrp_controller.NewMonorepoController(a.appInformer, a.applicationClientset, a.db, a.repoClientset)
	go controller.Run(ctx)
}

// newHTTPServer returns the HTTP server to serve HTTP/HTTPS requests. This is implemented
// using grpc-gateway as a proxy to the gRPC server.
func (a *MRPServer) newHTTPServer(_ context.Context, port int) *http.Server { //nolint:golint,unparam
	endpoint := fmt.Sprintf("localhost:%d", port)
	mux := http.NewServeMux()
	httpS := http.Server{
		Addr: endpoint,
		Handler: &handlerSwitcher{
			handler: mux,
		},
	}

	healthz.ServeHealthCheck(mux, a.healthCheck)
	return &httpS
}

func (a *MRPServer) checkServeErr(name string, err error) {
	if err != nil {
		if a.stopCh != nil {
			log.Fatalf("%s: %v", name, err)
		}
		// a nil stopCh indicates a graceful shutdown
		log.Infof("graceful shutdown %s: %v", name, err)
	} else {
		log.Infof("graceful shutdown %s", name)
	}
}

func startListener(host string, port int) (net.Listener, error) {
	var conn net.Listener
	var realErr error
	_ = wait.ExponentialBackoff(backoff, func() (bool, error) {
		conn, realErr = net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
		if realErr != nil {
			return false, nil
		}
		return true, nil
	})
	return conn, realErr
}

func (a *MRPServer) Listen() (*Listeners, error) {
	mainLn, err := startListener(a.ListenHost, a.ListenPort)
	if err != nil {
		return nil, err
	}
	return &Listeners{Main: mainLn}, nil
}

// Run runs the API Server
// We use k8s.io/code-generator/cmd/go-to-protobuf to generate the .proto files from the API types.
// k8s.io/ go-to-protobuf uses protoc-gen-gogo, which comes from gogo/protobuf (a fork of
// golang/protobuf).
func (a *MRPServer) Run(ctx context.Context, lns *Listeners) {
	httpS := a.newHTTPServer(ctx, a.ListenPort)
	// tlsConfig := tls.Config{}
	// tlsConfig.GetCertificate = func(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
	// 	return a.settings.Certificate, nil
	// }
	go func() { a.checkServeErr("httpS", httpS.Serve(lns.Main)) }()
	go a.RunController(ctx)

	if !cache.WaitForCacheSync(ctx.Done(), a.appInformer.HasSynced) {
		log.Fatal("Timed out waiting for project cache to sync")
	}

	a.stopCh = make(chan struct{})
	<-a.stopCh
}

// NewServer returns a new instance of the Event Reporter server
func NewApplicationChangeRevisionServer(ctx context.Context, opts MRPServerOpts) *MRPServer {
	appInformerNs := opts.Namespace
	if len(opts.ApplicationNamespaces) > 0 {
		appInformerNs = ""
	}
	appFactory := appinformer.NewSharedInformerFactoryWithOptions(opts.AppClientset, 0, appinformer.WithNamespace(appInformerNs), appinformer.WithTweakListOptions(func(_ *metav1.ListOptions) {}))

	appInformer := appFactory.Argoproj().V1alpha1().Applications().Informer()
	// appLister := appFactory.Argoproj().V1alpha1().Applications().Lister()

	settingsMgr := settings_util.NewSettingsManager(ctx, opts.KubeClientset, opts.Namespace)
	dbInstance := db.NewDB(opts.Namespace, settingsMgr, opts.KubeClientset)
	// repoclientset := repoapiclient.NewRepoServerClientset(repoServerAddress,
	// 	repoServerTimeoutSeconds,
	// 	tlsConfig)

	server := &MRPServer{
		MRPServerOpts: opts,
		// log:                  log.NewEntry(log.StandardLogger()),
		appInformer: appInformer,
		// appLister:            appLister,
		applicationClientset: opts.AppClientset,
		db:                   dbInstance,
		repoClientset:        opts.RepoClientset,
	}

	return server
}

// func newApplicationChangeRevisionServiceSet() *MRPServerSet {
// 	return &MRPServerSet{}
// }

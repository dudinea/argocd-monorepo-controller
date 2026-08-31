package controller

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-cd/v3/mrp_controller/metrics"
	"github.com/argoproj/argo-cd/v3/mrp_controller/service"
	appclientset "github.com/argoproj/argo-cd/v3/pkg/client/clientset/versioned"
	repoapiclient "github.com/argoproj/argo-cd/v3/reposerver/apiclient"
	"github.com/argoproj/argo-cd/v3/util/db"

	appv1 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
)

type MRPController interface {
	Run(ctx context.Context)
}

type monorepoController struct {
	appBroadcaster AppEventHandler
	acrService     service.MRPService
	eventTimeout   time.Duration
	appQueue       *KeyedAppQueue
	numWorkers     int
}

func NewMonorepoController(appInformer cache.SharedIndexInformer, applicationClientset appclientset.Interface, db db.ArgoDB, repoClientset repoapiclient.Clientset, metricsServer *metrics.MetricsServer, eventTimeout time.Duration, numWorkers int) MRPController {
	appBroadcaster := NewAppEventHandler()
	_, err := appInformer.AddEventHandler(appBroadcaster)
	if err != nil {
		log.Error(err)
	}
	appQueue := NewKeyedAppQueue()
	appQueue.SetLengthGauge(metricsServer.QueueLengthGauge())
	return &monorepoController{
		appBroadcaster: appBroadcaster,
		acrService:     service.NewMRPService(applicationClientset, db, repoClientset, metricsServer),
		eventTimeout:   eventTimeout,
		appQueue:       appQueue,
		numWorkers:     numWorkers,
	}
}

func (c *monorepoController) Run(ctx context.Context) {
	var logCtx log.FieldLogger = log.StandardLogger()

	calculateIfPermitted := func(ctx context.Context, a appv1.Application) { //nolint:golint,unparam
		logCtx.Debugf("calculateIfPermitted called for application '%s/%s'", a.Namespace, a.Name)
		ctx, cancel := context.WithTimeout(ctx, c.eventTimeout)
		err := c.acrService.ChangeRevision(ctx, &a)
		if err != nil {
			logCtx.WithError(err).Error("failed to calculate change revision")
		}
		cancel()
	}

	enqueueFunc := func(event *appv1.ApplicationWatchEvent) error {
		c.appQueue.Enqueue(event.Application)
		return nil
	}
	filterFunc := func(event *appv1.ApplicationWatchEvent) bool {
		if event.Type == watch.Bookmark || event.Type == watch.Deleted {
			return false
		}
		val, ok := event.Application.Annotations[appv1.AnnotationKeyManifestGeneratePaths]
		if !ok || val == "" {
			return false
		}
		return true
	}

	unsubscribe := c.appBroadcaster.Subscribe(enqueueFunc, filterFunc)

	wp := NewWorkerPool(c.appQueue, c.numWorkers, calculateIfPermitted)

	defer func() {
		unsubscribe()
		wp.Stop()
	}()
	<-ctx.Done()
	logCtx.Debug("got Done event")
}

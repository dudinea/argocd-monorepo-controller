package application_change_revision_controller

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-cd/v3/util/db"

	"github.com/argoproj/argo-cd/v3/mrp_controller/service"
	appclientset "github.com/argoproj/argo-cd/v3/pkg/client/clientset/versioned"
	repoapiclient "github.com/argoproj/argo-cd/v3/reposerver/apiclient"

	appv1 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
)

var watchAPIBufferSize = 1000

type MRPController interface {
	Run(ctx context.Context)
}

type monorepoController struct {
	appBroadcaster Broadcaster
	// cache                *servercache.Cache
	// appLister            applisters.ApplicationLister
	acrService service.MRPService
	// applicationClientset appclientset.Interface
}

func NewMonorepoController(appInformer cache.SharedIndexInformer, applicationClientset appclientset.Interface, db db.ArgoDB, repoClientset repoapiclient.Clientset) MRPController {
	appBroadcaster := NewBroadcaster()
	_, err := appInformer.AddEventHandler(appBroadcaster)
	if err != nil {
		log.Error(err)
	}
	return &monorepoController{
		appBroadcaster: appBroadcaster,
		// cache:                cache,
		// appLister:            appLister,
		// applicationClientset: applicationClientset,
		acrService: service.NewMRPService(applicationClientset, db, repoClientset),
	}
}

func (c *monorepoController) Run(ctx context.Context) {
	var logCtx log.FieldLogger = log.StandardLogger()

	calculateIfPermitted := func(ctx context.Context, a appv1.Application, eventType watch.EventType) error { //nolint:golint,unparam
		logCtx.Infof("calculateIfPermitted called for application '%s' eventType '%v'", a.Name, eventType)
		if eventType == watch.Bookmark || eventType == watch.Deleted {
			return nil // ignore this event
		}

		return c.acrService.ChangeRevision(ctx, &a)
	}

	// TODO: move to abstraction
	eventsChannel := make(chan *appv1.ApplicationWatchEvent, watchAPIBufferSize)
	unsubscribe := c.appBroadcaster.Subscribe(eventsChannel)
	defer unsubscribe()
	for {
		select {
		case <-ctx.Done():
			logCtx.Infof("got Done event")
			return
		case event := <-eventsChannel:
			logCtx.Infof("got event: channel size is %d", len(eventsChannel))

			ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			err := calculateIfPermitted(ctx, event.Application, event.Type)
			if err != nil {
				logCtx.WithError(err).Error("failed to calculate change revision")
			}
			cancel()
		}
	}
}

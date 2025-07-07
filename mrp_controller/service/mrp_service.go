package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/argoproj/argo-cd/v3/util/db"

	// "k8s.io/utils/ptr"
	application "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	appclientset "github.com/argoproj/argo-cd/v3/pkg/client/clientset/versioned"
	repoapiclient "github.com/argoproj/argo-cd/v3/reposerver/apiclient"
	"github.com/argoproj/argo-cd/v3/util/app/path"
	utilio "github.com/argoproj/argo-cd/v3/util/io"
)

const (
	CHANGE_REVISION_ANN  = "mrp-controller.argoproj.io/change-revision"
	CHANGE_REVISIONS_ANN = "mrp-controller.argoproj.io/change-revisions"
	GIT_REVISION_ANN     = "mrp-controller.argoproj.io/git-revision"
)

type MRPService interface {
	ChangeRevision(ctx context.Context, application *application.Application) error
}

type mrpService struct {
	applicationClientset appclientset.Interface
	lock                 sync.Mutex
	logger               *log.Logger
	db                   db.ArgoDB
	repoClientset        repoapiclient.Clientset
}

func NewMRPService(applicationClientset appclientset.Interface, db db.ArgoDB, repoClientset repoapiclient.Clientset) MRPService {
	return &mrpService{
		applicationClientset: applicationClientset,
		logger:               log.New(),
		db:                   db,
		repoClientset:        repoClientset,
	}
}

// FIXME: remove?
// func getChangeRevisionFromRevisions(revisions []string) string {
// 	if len(revisions) > 0 {
// 		return revisions[0]
// 	}
// 	return ""
// }

// Get revisions info from the Application manifest:
// changeRevision   (from annotation),
// gitRevision      (from annotation)
// currentRevision  (from Application Manifest)
// previousRevision (from Application Manifest)
func getApplicationRevisions(app *application.Application) (string, string, string, string) {
	anns := app.Annotations
	changeRevision := anns[CHANGE_REVISION_ANN]
	gitRevision := anns[GIT_REVISION_ANN]
	currentRevision, previousRevision := getRevisions(app)
	// argoRevision := ""
	// if app.Status.OperationState != nil && app.Status.OperationState.Operation.Sync != nil {
	// 	argoRevision = app.Status.OperationState.Operation.Sync.Revision
	// }
	// if argoRevision == "" {
	// 	argoRevision = app.Status.Sync.Revision
	// }
	return changeRevision, gitRevision, currentRevision, previousRevision
}

// FIXME: multisource applications support!
func (c *mrpService) ChangeRevision(ctx context.Context, a *application.Application) error {
	c.logger.Infof("ChangeRevision called for application %s", a.Name)
	c.lock.Lock()
	defer c.lock.Unlock()

	app, err := c.applicationClientset.ArgoprojV1alpha1().Applications(a.Namespace).Get(ctx, a.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	c.logger.Debugf("ChangeRevision retrieved app: %s", app.Name)

	// FIXME: race condition: sync may already be completed!
	// if app.Operation == nil || app.Operation.Sync == nil {
	// 	c.logger.Infof("skipping because non-relevant operation: %v", app.Operation)
	// 	return nil
	// }
	changeRevision, gitRevision, currentRevision, previousRevision := getApplicationRevisions(a)
	// current argo revision not changed since the last time we red the revions info
	c.logger.Infof("changeRevision is %s, gitRevision is %s, currentRevision is %s, previousRevision is %s  for application %s",
		changeRevision, gitRevision, currentRevision, previousRevision, app.Name)
	if gitRevision != "" && gitRevision == currentRevision {
		c.logger.Infof("Change revision already calculated for application %s", app.Name)
		return nil
	}

	newChangeRevision, err := c.calculateChangeRevision(ctx, app, currentRevision, previousRevision)
	if err != nil {
		return err
	}

	if newChangeRevision == nil || *newChangeRevision == "" {
		c.logger.Infof("Revision for application %s is empty", app.Name)
		return nil
	}

	c.logger.Infof("New change revision for application %s is %s", app.Name, *newChangeRevision)

	if changeRevision == *newChangeRevision {
		c.logger.Infof("Application change revision for %s has not changed", app.Name)
	}

	c.logger.Infof("Patching operation for application %s", app.Name)
	return c.annotateAppWithChangeRevision(ctx, app, *newChangeRevision, currentRevision)
}

func (c *mrpService) calculateChangeRevision(ctx context.Context,
	a *application.Application,
	currentRevision string, previousRevision string,
) (*string, error) {
	c.logger.Debugf("Calculate revision for application '%s', current revision '%s', previous revision '%s'", a.Name, currentRevision, previousRevision)

	val, ok := a.Annotations[application.AnnotationKeyManifestGeneratePaths]
	if !ok || val == "" {
		c.logger.Infof("manifest generation paths not set for application  '%s/%s'", a.Namespace, a.Name)
		return nil, status.Errorf(codes.FailedPrecondition, "manifest generation paths not set")
	}

	repo, err := c.db.GetRepository(ctx, a.Spec.GetSource().RepoURL, a.Spec.Project)
	if err != nil {
		return nil, fmt.Errorf("error getting repository: %w", err)
	}
	c.logger.Debugf("repository is %s of type %s", repo.Name, repo.Type)

	closer, client, err := c.repoClientset.NewRepoServerClient()
	if err != nil {
		return nil, fmt.Errorf("error creating repo server client: %w", err)
	}
	defer utilio.Close(closer)
	changeRevisionResult, err := client.GetChangeRevision(ctx, &repoapiclient.ChangeRevisionRequest{
		AppName:          a.GetName(),
		Namespace:        a.GetNamespace(),
		CurrentRevision:  currentRevision,
		PreviousRevision: previousRevision,
		Paths:            path.GetAppRefreshPaths(a),
		Repo:             repo,
	})
	if err != nil {
		return nil, fmt.Errorf("error getting change revision: %w", err)
	}
	if changeRevisionResult == nil {
		return nil, errors.New("got nil change revision result, this cannot not happen")
	}
	c.logger.Infof("change revision result from repo server: %s", changeRevisionResult.Revision)
	return &changeRevisionResult.Revision, nil
}

// FIXME: multisource annotations support
func (c *mrpService) annotateAppWithChangeRevision(ctx context.Context, a *application.Application, changeRevision string, argoRevision string) error {
	// FIXME: make it smarter, annotate only what has changed
	// FIXME: fake multisource annotation for now
	changeRevisions := "[\"" + changeRevision + "\"]"
	patch, _ := json.Marshal(map[string]any{
		"metadata": map[string]any{
			"annotations": map[string]any{
				CHANGE_REVISION_ANN:  changeRevision,
				CHANGE_REVISIONS_ANN: changeRevisions,
				GIT_REVISION_ANN:     argoRevision,
			},
		},
	})
	_, err := c.applicationClientset.ArgoprojV1alpha1().Applications(a.Namespace).Patch(ctx, a.Name, types.MergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		c.logger.Errorf("failed to annotate: %v", err)
	}
	return err
}

func getCurrentRevisionForFirstSync(a *application.Application) string {
	if a.Operation != nil && a.Operation.Sync != nil {
		return a.Operation.Sync.Revision
	}
	if a.Status.Sync.Status == "Synced" && a.Status.Sync.Revision != "" {
		return a.Status.Sync.Revision
	}
	return ""
}

// Get revisions from AgoCD Application Manifest
// (operation and status sections).
// Current revision is the revision the application has been synchronized to last time
//
// Returns: currentRevision, previousRevision
func getRevisions(a *application.Application) (string, string) {
	if len(a.Status.History) == 0 {
		// it is first sync operation, and we have only current revision
		return getCurrentRevisionForFirstSync(a), ""
	}

	// in case if sync is already done, we need to use revision from sync result and previous revision from history
	if a.Status.Sync.Status == "Synced" && a.Status.OperationState != nil && a.Status.OperationState.SyncResult != nil {
		currentRevision := a.Status.OperationState.SyncResult.Revision
		// in case if we have only one history record, we need to return empty previous revision, because it is first sync result
		if len(a.Status.History) == 1 {
			return currentRevision, ""
		}
		return currentRevision, a.Status.History[len(a.Status.History)-2].Revision
	}
	// in case if sync is in progress, we need to use revision from operation and revision from latest history record
	currentRevision := getCurrentRevisionForFirstSync(a)
	previousRevision := a.Status.History[len(a.Status.History)-1].Revision
	return currentRevision, previousRevision
}

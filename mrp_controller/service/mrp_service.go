package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/argoproj/argo-cd/v3/util/db"

	// "k8s.io/utils/ptr"
	"github.com/argoproj/argo-cd/v3/mrp_controller/metrics"
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
	GIT_REVISIONS_ANN    = "mrp-controller.argoproj.io/git-revisions"
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
	metricsServer        *metrics.MetricsServer
}

func NewMRPService(applicationClientset appclientset.Interface, db db.ArgoDB, repoClientset repoapiclient.Clientset, metricsServer *metrics.MetricsServer) MRPService {
	return &mrpService{
		applicationClientset: applicationClientset,
		logger:               log.New(),
		db:                   db,
		repoClientset:        repoClientset,
		metricsServer:        metricsServer,
	}
}

// FIXME: remove?
// func getChangeRevisionFromRevisions(revisions []string) string {
// 	if len(revisions) > 0 {
// 		return revisions[0]
// 	}
// 	return ""
// }

//func isMultisource(app *application.Application) bool {
//	return len(a.Spec.GetSources()) > 0
//}

func (c *mrpService) getArrayFromAnnotation(app *application.Application, annotationName string) []string {
	var result []string
	annStr, ok := app.Annotations[annotationName]
	if ok && strings.TrimSpace(annStr) != "" {
		err := json.Unmarshal([]byte(annStr), &result)
		if err != nil {
			c.logger.Warnf("application %s/%s, Failed to parse annotation %s as array: %v",
				app.Namespace, app.Name, annotationName, err)
		}
	}
	return result

}

func (c *mrpService) getSourcesRevisions(app *application.Application) []sourceRevisions {
	var result []sourceRevisions
	sources := app.Spec.GetSources()
	fmt.Fprintf(os.Stderr, "sources=%v", sources)
	numSources := len(sources)
	anns := app.Annotations
	if app.Spec.HasMultipleSources() {
		result = make([]sourceRevisions, numSources)
		changeRevisions := c.getArrayFromAnnotation(app, CHANGE_REVISIONS_ANN)
		gitRevisions := c.getArrayFromAnnotation(app, GIT_REVISIONS_ANN)
		for idx, _ := range sources {
			//  FIXME: filter out helm sources
			currentRevision, previousRevision := getRevisionsMultiSource(app, idx)
			result[idx] = sourceRevisions{
				changeRevision:   sliceGetString(&changeRevisions, idx),
				gitRevision:      sliceGetString(&gitRevisions, idx),
				currentRevision:  currentRevision,
				previousRevision: previousRevision,
			}
		}
	} else {
		result = make([]sourceRevisions, 1)
		changeRevision := anns[CHANGE_REVISION_ANN]
		gitRevision := anns[GIT_REVISION_ANN]
		currentRevision, previousRevision := getRevisionsSingleSource(app)

		result[0] = sourceRevisions{
			changeRevision:   changeRevision,
			gitRevision:      gitRevision,
			currentRevision:  currentRevision,
			previousRevision: previousRevision,
		}
	}
	return result
}

// // Get revisions info from the Application manifest:
// // changeRevision   (from annotation),
// // gitRevision      (from annotation)
// // currentRevision  (from Application Manifest)
// // previousRevision (from Application Manifest)
// func getApplicationRevisions(app *application.Application, idx int) (string, string, string, string) {
// 	anns := app.Annotations
// 	changeRevision := anns[CHANGE_REVISION_ANN]
// 	gitRevision := anns[GIT_REVISION_ANN]
// 	currentRevision, previousRevision := getRevisions(app, idx)
// 	// argoRevision := ""
// 	// if app.Status.OperationState != nil && app.Status.OperationState.Operation.Sync != nil {
// 	// 	argoRevision = app.Status.OperationState.Operation.Sync.Revision
// 	// }
// 	// if argoRevision == "" {
// 	// 	argoRevision = app.Status.Sync.Revision
// 	// }
// 	return changeRevision, gitRevision, currentRevision, previousRevision
// }

// func getSourceIndices(a *application.Application) (from int, to int) {
// 	sources := a.Spec.GetSources()
// 	numSources := len(sources)
// 	// according to Argo Docs, if there is sources[] array, source attribute is ignored
// 	if numSources == 0 {
// 		return -1, 0
// 	} else {
// 		return 0, numSources
// 	}
// }

type sourceRevisions struct {
	changeRevision   string
	gitRevision      string
	currentRevision  string
	previousRevision string
}

// FIXME: multisource applications support!
func (c *mrpService) ChangeRevision(ctx context.Context, a *application.Application) error {
	startTime := time.Now()
	defer func() {
		reconcileDuration := time.Since(startTime)
		c.metricsServer.IncReconcile(a, reconcileDuration)
	}()
	c.logger.Infof("ChangeRevision called for application %s", a.Name)

	c.lock.Lock()
	defer c.lock.Unlock()

	app, err := c.applicationClientset.ArgoprojV1alpha1().Applications(a.Namespace).Get(ctx, a.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	c.logger.Debugf("ChangeRevision retrieved app: %s", app.Name)
	// we just need to know it exists, actual use of the value will be in calculateChangeRevision
	val, ok := a.Annotations[application.AnnotationKeyManifestGeneratePaths]
	if !ok || val == "" {
		c.logger.Infof("manifest generation paths not set for application  '%s/%s'", a.Namespace, a.Name)
		return status.Errorf(codes.FailedPrecondition, "manifest generation paths not set")
	}
	// FIXED: race condition: sync may already be completed!
	// if app.Operation == nil || app.Operation.Sync == nil {
	// 	c.logger.Infof("skipping because non-relevant operation: %v", app.Operation)
	// 	return nil
	// }
	//from, to := getSourceIndices(a)
	sourcesRevisions := c.getSourcesRevisions(a)
	for idx, r := range sourcesRevisions {
		//changeRevision, gitRevision, currentRevision, previousRevision := getApplicationRevisions(a, idx)
		c.logger.Infof("applicationSource %d changeRevision is %s, gitRevision is %s, currentRevision is %s, previousRevision is %s  for application %s",
			idx, r.changeRevision, r.gitRevision, r.currentRevision, r.previousRevision, app.Name)
		// current argo revision not changed since the last time we red the revions info
		if r.gitRevision != "" && r.gitRevision == r.currentRevision {
			c.logger.Infof("Change revision already calculated for application %s source %d", app.Name, idx)
			continue
		}
		newChangeRevision, err := c.calculateChangeRevision(ctx, app, r.currentRevision, r.previousRevision)
		if err != nil {
			// FIXME: probably need to continue and and finish updates for all repositories
			return err
		}
		c.logger.Infof("New change revision #%d for application %s is %s", idx, app.Name, *newChangeRevision)
		if newChangeRevision == nil || *newChangeRevision == "" {
			c.logger.Infof("Revision #%d for application %s is empty", idx, app.Name)
			continue
		}
		if r.changeRevision == *newChangeRevision {
			c.logger.Infof("Application change revision for %s has not changed", app.Name)
		}
		return c.annotateAppWithChangeRevision(ctx, app, *newChangeRevision, r.currentRevision)
	}

	c.logger.Infof("Patching operation for application %s", app.Name)
	return nil
}

func (c *mrpService) calculateChangeRevision(ctx context.Context,
	a *application.Application,
	currentRevision string, previousRevision string,
) (*string, error) {
	c.logger.Debugf("Calculate revision for application '%s', current revision '%s', previous revision '%s'", a.Name, currentRevision, previousRevision)

	repo, err := c.db.GetRepository(ctx, a.Spec.GetSource().RepoURL, a.Spec.Project)
	if err != nil {
		return nil, fmt.Errorf("error getting repository: %w", err)
	}
	c.logger.Debugf("repository is %s of type %s", repo.Name, repo.Type)

	closer, client, err := c.repoClientset.NewRepoServerClient()
	if err != nil {
		return nil, fmt.Errorf("error creating repo server client: %w", err)
	}

	repoRequestStartTime := time.Now()
	defer func() {
		repoRequestDuration := time.Since(repoRequestStartTime)
		c.metricsServer.ObserveRepoServerRequestDuration(repoRequestDuration)
		c.metricsServer.IncRepoServerRequest(err != nil)
		utilio.Close(closer)
	}()

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

func getCurrentRevisionForFirstSyncMultiSource(a *application.Application, idx int) string {
	if a.Operation != nil && a.Operation.Sync != nil {
		return sliceGetString(&a.Operation.Sync.Revisions, idx)
	}
	if a.Status.Sync.Status == "Synced" && a.Status.Sync.Revision != "" {
		return sliceGetString(&a.Status.Sync.Revisions, idx)
	}
	return ""
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

func sliceGetString(sl *[]string, idx int) string {
	if idx >= 0 && idx < len(*sl) {
		return (*sl)[idx]
	} else {
		return ""
	}
}

func getRevisionsFromHistoryMS(a *application.Application, historyIdx int, sourceIdx int) string {
	history := &a.Status.History[historyIdx]
	historicalSourceIdx := sourceIdx
	var historySrc *application.ApplicationSource = nil
	// History entry has enough sources
	if historicalSourceIdx < len(history.Sources) {
		// assume that in most cases historical source
		// has same index
		historySrc = &history.Sources[historicalSourceIdx]
	}
	src := &a.Spec.Sources[sourceIdx]
	if historySrc == nil || *historySrc != *src {
		// probably sources were reordered, try to
		// find source index
		historicalSourceIdx = -1
		for i := 0; i < len(history.Sources); i++ {
			if history.Sources[i] == *src {
				historicalSourceIdx = i
				break
			}
		}

	}
	if historicalSourceIdx >= 0 {
		return sliceGetString(&history.Revisions, historicalSourceIdx)
	} else {
		return ""
	}
}

// func getRevisions(a *application.Application, idx int) (string, string) {
// 	if idx == -1 {
// 		return getRevisionsSingleSource(a)
// 	}
// 	return getRevisionsMultiSource(a, idx)
// }

// Get revisions from AgoCD Application Manifest
// (operation and status sections).
// Current revision is the revision the application has been synchronized to last time
//
// Returns: currentRevision, previousRevision
func getRevisionsMultiSource(a *application.Application, idx int) (string, string) {
	// Revisions arrays may be shorter than current
	if len(a.Status.History) == 0 {
		// it is first sync operation, and we have only current revision
		return getCurrentRevisionForFirstSyncMultiSource(a, idx), ""
	}
	// in case if sync is already done, we need to use revision from sync result and previous revision from history
	if a.Status.Sync.Status == "Synced" && a.Status.OperationState != nil && a.Status.OperationState.SyncResult != nil {
		currentRevision := sliceGetString(&a.Status.OperationState.SyncResult.Revisions, idx)
		// in case if we have only one history record, we need to return empty previous revision, because it is first sync result
		if len(a.Status.History) == 1 {
			return currentRevision, ""
		}
		return currentRevision, getRevisionsFromHistoryMS(a, len(a.Status.History)-2, idx)
	}
	// in case if sync is in progress, we need to use revision from operation and revision from latest history record
	currentRevision := getCurrentRevisionForFirstSyncMultiSource(a, idx)
	previousRevision := getRevisionsFromHistoryMS(a, len(a.Status.History)-1, idx)
	return currentRevision, previousRevision
}

// Get revisions from AgoCD Application Manifest
// (operation and status sections).
// Current revision is the revision the application has been synchronized to last time
//
// Returns: currentRevision, previousRevision
func getRevisionsSingleSource(a *application.Application) (string, string) {
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

package service

import (
	"context"
	"encoding/json"
	"sync"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"


	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"github.com/argoproj/argo-cd/v3/util/db"
	
	// "k8s.io/utils/ptr"
	"github.com/argoproj/argo-cd/v3/util/app/path"
	repoapiclient "github.com/argoproj/argo-cd/v3/reposerver/apiclient"
	application "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	appclientset "github.com/argoproj/argo-cd/v3/pkg/client/clientset/versioned"
	ioutil "github.com/argoproj/argo-cd/v3/util/io"
)

const CHANGE_REVISION_ANN = "mrp-controller.argoproj.io/change-revision"
const GIT_REVISION_ANN = "mrp-controller.argoproj.io/git-revision"

type MRPService interface {
	ChangeRevision(ctx context.Context, application *application.Application) error
}

type acrService struct {
	applicationClientset     appclientset.Interface
	lock                     sync.Mutex
	logger                   *log.Logger
	db                       db.ArgoDB
	repoClientset            repoapiclient.Clientset
}

func NewMRPService(applicationClientset appclientset.Interface, db db.ArgoDB, repoClientset repoapiclient.Clientset) MRPService {
	return &acrService{
		applicationClientset:     applicationClientset,
		logger:                   log.New(),
		db:                       db,
		repoClientset:            repoClientset,
	}
}

// FIXME: remove?
func getChangeRevisionFromRevisions(revisions []string) string {
	if len(revisions) > 0 {
		return revisions[0]
	}
	return ""
}

// Return revisions info from the Application manifest:
// ChangeRevision (from annotation),
// GitRevision    (from annotation)
// ArgoRevision   (from Application Manifest)
func getApplicationRevisions(app *application.Application) (string, string, string)  {
	anns := app.Annotations
	changeRevision := anns[CHANGE_REVISION_ANN]
	gitRevision := anns[GIT_REVISION_ANN]
	argoRevision := ""
	if app.Status.OperationState != nil && app.Status.OperationState.Operation.Sync != nil {
		argoRevision = app.Status.OperationState.Operation.Sync.Revision
	}
	if "" == argoRevision {
		argoRevision = app.Status.Sync.Revision
	}
	return changeRevision, gitRevision, argoRevision
}

// FIXME: multisource applications support!
func (c *acrService) ChangeRevision(ctx context.Context, a *application.Application) error {
	c.logger.Infof("ChangeRevision called for application %s", a.Name)
	c.lock.Lock()
	defer c.lock.Unlock()

	app, err := c.applicationClientset.ArgoprojV1alpha1().Applications(a.Namespace).Get(ctx, a.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	c.logger.Infof("ChangeRevision got app with options: %s", app.Name)

	// FIXME: race condition: sync may already be completed!
	// if app.Operation == nil || app.Operation.Sync == nil {
	// 	c.logger.Infof("skipping because non-relevant operation: %v", app.Operation)
	// 	return nil
	// }
	changeRevision, gitRevision, argoRevision := getApplicationRevisions(a);
	
	// current argo revision not changed since the last time we red the revions info
	c.logger.Infof("ChangeRevision is %s, gitRevision is %s, ArgoRevision is %s for application %s",
		changeRevision, gitRevision, argoRevision, app.Name)
	if gitRevision != "" && gitRevision == argoRevision {
		c.logger.Infof("Change revision already calculated for application %s", app.Name)
		return nil
	}
	newChangeRevision, err := c.calculateRevision(ctx, app)
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
	//revisions := []string{*revision}

	/*if app.Status.OperationState != nil && app.Status.OperationState.Operation.Sync != nil {
		c.logger.Infof("Patch operation status for application %s", app.Name)
		return c.patchOperationSyncResultWithChangeRevision(ctx, app, revisions)
	}*/

	c.logger.Infof("Patching operation for application %s", app.Name)
	return c.annotateAppWithChangeRevision(ctx, app, *newChangeRevision, argoRevision)
}

func (c *acrService) calculateRevision(ctx context.Context, a *application.Application) (*string, error) {
	c.logger.Infof("Calculate revision called for application '%s'", a.Name)
	currentRevision, previousRevision := c.getRevisions(ctx, a)
	c.logger.Infof("Calculate revision for application '%s', current revision '%s', previous revision '%s'", a.Name, currentRevision, previousRevision)
	
	val, ok := a.Annotations[application.AnnotationKeyManifestGeneratePaths]
	if !ok || val == "" {
		c.logger.Infof("manifest generation paths not set for application  '%s/%s'", a.Namespace, a.Name)
		return nil, status.Errorf(codes.FailedPrecondition, "manifest generation paths not set")
	}

	
	repo, err := c.db.GetRepository(ctx, a.Spec.GetSource().RepoURL, a.Spec.Project)
	if err != nil {
 		return nil, fmt.Errorf("error getting repository: %w", err)
	}
	c.logger.Infof("repository is %v", repo)

	closer, client, err := c.repoClientset.NewRepoServerClient()
 	if err != nil {
 		return nil, fmt.Errorf("error creating repo server client: %w", err)
 	}
 	defer ioutil.Close(closer)
	c.logger.Infof("repository client  is %v", client)

	//changeRevisionResult, err := client.TestRepository(ctx, &repoapiclient.TestRepositoryRequest{Repo: repo})
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
	c.logger.Infof("repo response is %v", changeRevisionResult)
	// ED: end of application service logic
	// changeRevisionResult, err := c.applicationServiceClient.GetChangeRevision(ctx, &appclient.ChangeRevisionRequest{
	// 	AppName:          ptr.To(a.GetName()),
	// 	Namespace:        ptr.To(a.GetNamespace()),
	// 	CurrentRevision:  ptr.To(currentRevision),
	// 	PreviousRevision: ptr.To(previousRevision),
	// })
	//if err != nil {
	//		return nil, err
	//}
	return &changeRevisionResult.Revision, nil
}

// FIXME: multisource annotations support
func (c *acrService) annotateAppWithChangeRevision(ctx context.Context, a *application.Application, changeRevision string, argoRevision string) error {
	// FIXME: make it smarter, do not annotate both whe only one suffice
	patch, _ := json.Marshal(map[string]any{
		"metadata": map[string]any{
			"annotations": map[string]any{
				CHANGE_REVISION_ANN: changeRevision,
				GIT_REVISION_ANN: argoRevision,
			},
		},
	})
	_, err := c.applicationClientset.ArgoprojV1alpha1().Applications(a.Namespace).Patch(ctx, a.Name, types.MergePatchType, patch, metav1.PatchOptions{})
	if nil != err {
		c.logger.Errorf("failed to annotate: %v", err)
	}
	return err
 	//} else {
	//		c.logger.Errorf("annotating multiple with revisions not implemented")
	//}
	//return nil
}

// func (c *acrService) patchOperationWithChangeRevision(ctx context.Context, a *application.Application, revisions []string) error {
// 	if len(revisions) == 1 {
// 		patch, _ := json.Marshal(map[string]any{
// 			"operation": map[string]any{
// 				"sync": map[string]any{
// 					"changeRevision": revisions[0],
// 				},
// 			},
// 		})
// 		_, err := c.applicationClientset.ArgoprojV1alpha1().Applications(a.Namespace).Patch(ctx, a.Name, types.MergePatchType, patch, metav1.PatchOptions{})
// 		return err
// 	}

// 	patch, _ := json.Marshal(map[string]any{
// 		"operation": map[string]any{
// 			"sync": map[string]any{
// 				"changeRevisions": revisions,
// 			},
// 		},
// 	})
// 	_, err := c.applicationClientset.ArgoprojV1alpha1().Applications(a.Namespace).Patch(ctx, a.Name, types.MergePatchType, patch, metav1.PatchOptions{})
// 	return err
// }

// func (c *acrService) patchOperationSyncResultWithChangeRevision(ctx context.Context, a *application.Application, revisions []string) error {
// 	if len(revisions) == 1 {
// 		patch, _ := json.Marshal(map[string]any{
// 			"status": map[string]any{
// 				"operationState": map[string]any{
// 					"operation": map[string]any{
// 						"sync": map[string]any{
// 							"changeRevision": revisions[0],
// 						},
// 					},
// 				},
// 			},
// 		})
// 		_, err := c.applicationClientset.ArgoprojV1alpha1().Applications(a.Namespace).Patch(ctx, a.Name, types.MergePatchType, patch, metav1.PatchOptions{})
// 		return err
// 	}

// 	patch, _ := json.Marshal(map[string]any{
// 		"status": map[string]any{
// 			"operationState": map[string]any{
// 				"operation": map[string]any{
// 					"sync": map[string]any{
// 						"changeRevisions": revisions,
// 					},
// 				},
// 			},
// 		},
// 	})
// 	_, err := c.applicationClientset.ArgoprojV1alpha1().Applications(a.Namespace).Patch(ctx, a.Name, types.MergePatchType, patch, metav1.PatchOptions{})
// 	return err
// }

func getCurrentRevisionFromOperation(a *application.Application) string {
	if a.Operation != nil && a.Operation.Sync != nil {
		return a.Operation.Sync.Revision
	}
	return ""
}

func (c *acrService) getRevisions(_ context.Context, a *application.Application) (string, string) {
	if len(a.Status.History) == 0 {
		// it is first sync operation, and we have only current revision
		return getCurrentRevisionFromOperation(a), ""
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
	currentRevision := getCurrentRevisionFromOperation(a)
	previousRevision := a.Status.History[len(a.Status.History)-1].Revision
	return currentRevision, previousRevision
}

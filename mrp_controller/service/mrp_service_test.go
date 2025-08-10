package service

import (
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	// 	test2 "github.com/sirupsen/logrus/hooks/test"

	"github.com/argoproj/argo-cd/v3/mrp_controller/metrics"
	appsv1 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/v3/pkg/client/clientset/versioned/mocks"
	appmocks "github.com/argoproj/argo-cd/v3/pkg/client/clientset/versioned/typed/application/v1alpha1/mocks"
	applistermocks "github.com/argoproj/argo-cd/v3/pkg/client/listers/application/v1alpha1/mocks"
	repoapiclient "github.com/argoproj/argo-cd/v3/reposerver/apiclient"
	repomocks "github.com/argoproj/argo-cd/v3/reposerver/apiclient/mocks"

	"github.com/argoproj/argo-cd/v3/util/app/path"
	dbmocks "github.com/argoproj/argo-cd/v3/util/db/mocks"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"

	//      "github.com/argoproj/argo-cd/v3/reposerver/apiclient/mocks"
	// 	apps "github.com/argoproj/argo-cd/v3/pkg/client/clientset/versioned/fake"
	// 	"github.com/argoproj/argo-cd/v3/test"
	// 	"github.com/stretchr/testify/assert"
	// 	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// 	"k8s.io/utils/ptr"
)

const fakeApp = `
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: test-app
  namespace: default
spec:
  source:
    path: some/path
    repoURL: https://github.com/argoproj/argocd-example-apps.git
    targetRevision: HEAD
    ksonnet:
      environment: default
  destination:
    namespace: guestbook
    server: https://cluster-api.example.com
`

// const fakeAppWithOperation = `
// apiVersion: argoproj.io/v1alpha1
// kind: Application
// metadata:
//   annotations:
//     argocd.argoproj.io/manifest-generate-paths: .
//   finalizers:
//   - resources-finalizer.argocd.argoproj.io
//   labels:
//     app.kubernetes.io/instance: guestbook
//   name: guestbook
//   namespace: codefresh
// operation:
//   initiatedBy:
//     automated: true
//   retry:
//     limit: 5
//   sync:
//     prune: true
//     revision: c732f4d2ef24c7eeb900e9211ff98f90bb646505
//     syncOptions:
//     - CreateNamespace=true
// spec:
//   destination:
//     namespace: guestbook
//     server: https://kubernetes.default.svc
//   project: default
//   source:
//     path: apps/guestbook
//     repoURL: https://github.com/pasha-codefresh/precisely-gitsource.git
//     targetRevision: HEAD
// `

const syncedAppWithoutHistory = `
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  annotations:
    argocd.argoproj.io/manifest-generate-paths: /demo-applications/trioapp-dev
  name: demo-trioapp-dev
  namespace: argocd
spec:
  destination:
    name: in-cluster
    namespace: demo-dev
  project: default
  source:
    path: demo-applications/trioapp-dev
    repoURL: https://github.com/somehere/repo01.git
    targetRevision: dev
  syncPolicy:
    automated:
      allowEmpty: false
      prune: true
      selfHeal: true
    syncOptions:
    - PrunePropagationPolicy=foreground
    - Replace=false
    - PruneLast=false
    - Validate=true
    - CreateNamespace=true
    - ApplyOutOfSyncOnly=false
    - ServerSideApply=true
    - RespectIgnoreDifferences=false
status:
  controllerNamespace: argocd
  health:
    lastTransitionTime: "2025-07-05T16:24:49Z"
    status: Healthy
  reconciledAt: "2025-07-06T15:57:23Z"
  resourceHealthSource: appTree
  resources:
  - kind: ConfigMap
    name: config-cm
    namespace: demo-dev
    status: Synced
    version: v1
  sourceType: Helm
  sync:
    comparedTo:
      destination:
        name: in-cluster
        namespace: demo-dev
      source:
        path: demo-applications/trioapp-dev
        repoURL: https://github.com/somehere/repo01.git
        targetRevision: dev
    revision: 2b571ad9ceaab7ed1e6225ca674e367f2d07e41d
    status: Synced
`

const syncedAppWithSingleHistoryAnnotated = `
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  annotations:
    argocd.argoproj.io/manifest-generate-paths: .
    mrp-controller.argoproj.io/change-revision: 792822850fd2f6db63597533e16dfa27e6757dc5
    mrp-controller.argoproj.io/git-revision: 00d423763fbf56d2ea452de7b26a0ab20590f521
  finalizers:
  - resources-finalizer.argocd.argoproj.io
  labels:
    app.kubernetes.io/instance: guestbook
  name: guestbook
  namespace: argocd
operation:
  initiatedBy:
    automated: true
  retry:
    limit: 5
  sync:
    prune: true
    revision: c732f4d2ef24c7eeb900e9211ff98f90bb646505
    syncOptions:
    - CreateNamespace=true
spec:
  destination:
    namespace: guestbook
    server: https://kubernetes.default.svc
  project: default
  source:
    path: apps/guestbook
    repoURL: https://github.com/pasha-codefresh/precisely-gitsource.git
    targetRevision: HEAD
status:
  history:
  - deployStartedAt: "2024-06-20T19:35:36Z"
    deployedAt: "2024-06-20T19:35:44Z"
    id: 3
    initiatedBy: {}
    revision: 792822850fd2f6db63597533e16dfa27e6757dc5
    source:
      path: apps/guestbook
      repoURL: https://github.com/pasha-codefresh/precisely-gitsource.git
      targetRevision: HEAD
  operationState:
    operation:
      sync:
        prune: true
        revision: c732f4d2ef24c7eeb900e9211ff98f90bb646506
        syncOptions:
        - CreateNamespace=true
    phase: Running
    startedAt: "2024-06-20T19:47:34Z"
    syncResult:
      revision: c732f4d2ef24c7eeb900e9211ff98f90bb646506
      source:
        path: apps/guestbook
        repoURL: https://github.com/pasha-codefresh/precisely-gitsource.git
        targetRevision: HEAD
  sync:
    revision: 00d423763fbf56d2ea452de7b26a0ab20590f521
    status: Synced
`

// const syncedAppWithHistory = `
// apiVersion: argoproj.io/v1alpha1
// kind: Application
// metadata:
//   annotations:
//     argocd.argoproj.io/manifest-generate-paths: .
//   finalizers:
//   - resources-finalizer.argocd.argoproj.io
//   labels:
//     app.kubernetes.io/instance: guestbook
//   name: guestbook
//   namespace: argocd
// operation:
//   initiatedBy:
//     automated: true
//   retry:
//     limit: 5
//   sync:
//     prune: true
//     revision: c732f4d2ef24c7eeb900e9211ff98f90bb646505
//     syncOptions:
//     - CreateNamespace=true
// spec:
//   destination:
//     namespace: guestbook
//     server: https://kubernetes.default.svc
//   project: default
//   source:
//     path: apps/guestbook
//     repoURL: https://github.com/pasha-codefresh/precisely-gitsource.git
//     targetRevision: HEAD
// status:
//   history:
//   - deployStartedAt: "2024-06-20T19:35:36Z"
//     deployedAt: "2024-06-20T19:35:44Z"
//     id: 3
//     initiatedBy: {}
//     revision: 792822850fd2f6db63597533e16dfa27e6757dc5
//     source:
//       path: apps/guestbook
//       repoURL: https://github.com/pasha-codefresh/precisely-gitsource.git
//       targetRevision: HEAD
//   - deployStartedAt: "2024-06-20T19:36:34Z"
//     deployedAt: "2024-06-20T19:36:42Z"
//     id: 4
//     initiatedBy: {}
//     revision: ee5373eb9814e247ec6944e8b8897a8ec2f8528e
//     source:
//       path: apps/guestbook
//       repoURL: https://github.com/pasha-codefresh/precisely-gitsource.git
//       targetRevision: HEAD
//   operationState:
//     operation:
//       sync:
//         prune: true
//         revision: c732f4d2ef24c7eeb900e9211ff98f90bb646506
//         syncOptions:
//         - CreateNamespace=true
//     phase: Running
//     startedAt: "2024-06-20T19:47:34Z"
//     syncResult:
//       revision: c732f4d2ef24c7eeb900e9211ff98f90bb646505
//       source:
//         path: apps/guestbook
//         repoURL: https://github.com/pasha-codefresh/precisely-gitsource.git
//         targetRevision: HEAD
//   sync:
//     revision: 00d423763fbf56d2ea452de7b26a0ab20590f521
//     status: Synced
// `

func Test_GetApplicationRevisions(t *testing.T) {
	anapp := createTestApp(t, syncedAppWithSingleHistoryAnnotated)
	changeRevision, gitRevision, currentRevision, previousRevision := getApplicationRevisions(anapp)
	assert.Equal(t, "c732f4d2ef24c7eeb900e9211ff98f90bb646506", currentRevision)
	assert.Empty(t, previousRevision)
	assert.Equal(t, "792822850fd2f6db63597533e16dfa27e6757dc5", changeRevision)
	assert.Equal(t, "00d423763fbf56d2ea452de7b26a0ab20590f521", gitRevision)
}

func Test_GetApplicationRevisionsWithoutHistory(t *testing.T) {
	anapp := createTestApp(t, syncedAppWithoutHistory)
	changeRevision, gitRevision, currentRevision, previousRevision := getApplicationRevisions(anapp)
	assert.Equal(t, "2b571ad9ceaab7ed1e6225ca674e367f2d07e41d", currentRevision)
	assert.Empty(t, previousRevision)
	assert.Empty(t, changeRevision)
	assert.Empty(t, gitRevision)
}

func Test_CalculateRevision_no_paths(t *testing.T) {
	mrpService := newTestMRPService(t,
		&repomocks.Clientset{},
		&mocks.Interface{},
		&dbmocks.ArgoDB{})
	app := createTestApp(t, fakeApp)
	revision, err := mrpService.calculateChangeRevision(t.Context(), app, "", "")
	assert.Nil(t, revision)
	require.Error(t, err)
	assert.Equal(t, "rpc error: code = FailedPrecondition desc = manifest generation paths not set", err.Error())
}

func Test_CalculateRevision(t *testing.T) {
	expectedRevision := "ffffffffffffffffffffffffffffffffffffffff"
	repo := appsv1.Repository{Repo: "myrepo"}
	app := createTestApp(t, syncedAppWithSingleHistoryAnnotated)
	db := createTestArgoDbForAppAndRepo(t, app, &repo)
	changeRevisionRequest := repoapiclient.ChangeRevisionRequest{
		AppName:          app.GetName(),
		Namespace:        app.GetNamespace(),
		CurrentRevision:  "c732f4d2ef24c7eeb900e9211ff98f90bb646506",
		PreviousRevision: "",
		Paths:            path.GetAppRefreshPaths(app),
		Repo:             &repo,
	}
	changeRevisionResponce := repoapiclient.ChangeRevisionResponse{}
	changeRevisionResponce.Revision = expectedRevision
	clientsetmock := createTestRepoclientForApp(t, &changeRevisionRequest, &changeRevisionResponce)
	mrpService := newTestMRPService(t, clientsetmock, &mocks.Interface{}, db)
	currentRevision, previousRevision := getRevisions(app)
	revision, err := mrpService.calculateChangeRevision(t.Context(), app, currentRevision, previousRevision)
	require.NoError(t, err)
	assert.NotNil(t, revision)
	assert.Equal(t, expectedRevision, *revision)
}

func Test_ChangeRevision(t *testing.T) {
	expectedRevision := "ffffffffffffffffffffffffffffffffffffffff"
	repo := appsv1.Repository{Repo: "myrepo"}
	app := createTestApp(t, syncedAppWithSingleHistoryAnnotated)
	appClientMock := createTestAppClientForApp(t, app)
	db := createTestArgoDbForAppAndRepo(t, app, &repo)
	changeRevisionRequest := repoapiclient.ChangeRevisionRequest{
		AppName:          app.GetName(),
		Namespace:        app.GetNamespace(),
		CurrentRevision:  "c732f4d2ef24c7eeb900e9211ff98f90bb646506",
		PreviousRevision: "",
		Paths:            path.GetAppRefreshPaths(app),
		Repo:             &repo,
	}
	changeRevisionResponce := repoapiclient.ChangeRevisionResponse{}
	changeRevisionResponce.Revision = expectedRevision
	clientsetmock := createTestRepoclientForApp(t, &changeRevisionRequest, &changeRevisionResponce)
	mrpService := newTestMRPService(t, clientsetmock, appClientMock, db)
	err := mrpService.ChangeRevision(t.Context(), app)
	assert.NoError(t, err)
}

func newTestMetricsServer(t *testing.T, dbMock *dbmocks.ArgoDB) *metrics.MetricsServer {
	healthcheck := func(_ *http.Request) error { return nil }
	var appConditions []string
	var appLabels []string
	canProcessApp := func(obj any) bool { return false }
	appLister := &applistermocks.ApplicationLister{}
	result, err := metrics.NewMetricsServer("0.0.0.0:8091",
		appLister,     /*AppLister*/
		canProcessApp, /*AppFilter*/
		healthcheck,   /*healthCheck */
		appLabels,     /* appLabels */
		appConditions, /* appCondition */
		dbMock,
	)
	assert.NoError(t, err)
	return result
}

func newTestMRPService(t *testing.T, repoClientMock *repomocks.Clientset,
	applicationClientsetMock *mocks.Interface,
	dbMock *dbmocks.ArgoDB,
) *mrpService {
	return &mrpService{
		applicationClientset: applicationClientsetMock,
		repoClientset:        repoClientMock,
		db:                   dbMock,
		logger:               logrus.New(),
		metricsServer:        newTestMetricsServer(t, dbMock),
	}
}

func createTestApp(t *testing.T, testApp string, opts ...func(app *appsv1.Application)) *appsv1.Application {
	t.Helper()
	var app appsv1.Application
	err := yaml.Unmarshal([]byte(testApp), &app)
	if err != nil {
		panic(err)
	}
	for i := range opts {
		opts[i](&app)
	}
	return &app
}

func createTestRepoclientForApp(t *testing.T,
	changeRevisionRequest *repoapiclient.ChangeRevisionRequest,
	changeRevisionResponse *repoapiclient.ChangeRevisionResponse,
) *repomocks.Clientset {
	t.Helper()
	clientsetmock := repomocks.Clientset{}
	clientmock := repomocks.RepoServerServiceClient{}
	clientmock.On("GetChangeRevision", t.Context(), changeRevisionRequest).
		Return(changeRevisionResponse, nil).Once()
	clientsetmock.RepoServerServiceClient = &clientmock
	return &clientsetmock
}

func createTestAppClientForApp(t *testing.T, app *appsv1.Application) *mocks.Interface {
	t.Helper()
	appintMock := &appmocks.ApplicationInterface{}
	appintMock.On("Get", t.Context(), app.Name, metav1.GetOptions{}).Return(app, nil)
	appintMock.On("Patch", t.Context(), app.Name, types.MergePatchType,
		// FIXME: test for the patch
		mock.MatchedBy(func(_ any) bool { return true }),
		metav1.PatchOptions{}).Return(app, nil)

	av1alpha1 := &appmocks.ArgoprojV1alpha1Interface{}
	av1alpha1.On("Applications", app.Namespace).Return(appintMock)

	mock := &mocks.Interface{}
	mock.On("ArgoprojV1alpha1").Return(av1alpha1)
	return mock
}

func createTestArgoDbForAppAndRepo(t *testing.T, app *appsv1.Application, repo *appsv1.Repository) *dbmocks.ArgoDB {
	t.Helper()
	db := dbmocks.ArgoDB{}
	db.On("GetRepository", t.Context(), app.Spec.Source.RepoURL, app.Spec.Project).
		Return(repo, nil).Once()
	return &db
}

// func Test_ChangeRevision(r *testing.T) {
// 	r.Run("history list is empty", func(t *testing.T) {
// 		acrService := newTestACRService(&mocks.ApplicationClient{})
// 		current, previous := acrService.getRevisions(r.Context(), createTestApp(fakeApp))
// 		assert.Equal(t, "", current)
// 		assert.Equal(t, "", previous)
// 	})
// }

// func Test_getRevisions(r *testing.T) {
// 	r.Run("history list is empty", func(t *testing.T) {
// 		acrService := newTestACRService(&mocks.ApplicationClient{})
// 		current, previous := acrService.getRevisions(r.Context(), createTestApp(fakeApp))
// 		assert.Equal(t, "", current)
// 		assert.Equal(t, "", previous)
// 	})

// 	r.Run("history list is empty, but operation happens right now", func(t *testing.T) {
// 		acrService := newTestACRService(&mocks.ApplicationClient{})
// 		current, previous := acrService.getRevisions(r.Context(), createTestApp(fakeAppWithOperation))
// 		assert.Equal(t, "c732f4d2ef24c7eeb900e9211ff98f90bb646505", current)
// 		assert.Equal(t, "", previous)
// 	})

// 	r.Run("history list contains only one element, also sync result is here", func(t *testing.T) {
// 		acrService := newTestACRService(&mocks.ApplicationClient{})
// 		current, previous := acrService.getRevisions(r.Context(), createTestApp(syncedAppWithSingleHistory))
// 		assert.Equal(t, "c732f4d2ef24c7eeb900e9211ff98f90bb646505", current)
// 		assert.Equal(t, "", previous)
// 	})

// 	r.Run("application is synced", func(t *testing.T) {
// 		acrService := newTestACRService(&mocks.ApplicationClient{})
// 		app := createTestApp(syncedAppWithHistory)
// 		current, previous := acrService.getRevisions(r.Context(), app)
// 		assert.Equal(t, app.Status.OperationState.SyncResult.Revision, current)
// 		assert.Equal(t, app.Status.History[len(app.Status.History)-2].Revision, previous)
// 	})

// 	r.Run("application sync is in progress", func(t *testing.T) {
// 		acrService := newTestACRService(&mocks.ApplicationClient{})
// 		app := createTestApp(syncedAppWithHistory)
// 		app.Status.Sync.Status = "Syncing"
// 		current, previous := acrService.getRevisions(r.Context(), app)
// 		assert.Equal(t, app.Operation.Sync.Revision, current)
// 		assert.Equal(t, app.Status.History[len(app.Status.History)-1].Revision, previous)
// 	})
// }

// func Test_ChangeRevision(r *testing.T) {
// 	r.Run("Change revision", func(t *testing.T) {
// 		client := &mocks.ApplicationClient{}
// 		client.On("GetChangeRevision", mock.Anything, mock.Anything).Return(&appclient.ChangeRevisionResponse{
// 			Revision: ptr.To("new-revision"),
// 		}, nil)
// 		acrService := newTestACRService(client)
// 		app := createTestApp(syncedAppWithHistory)

// 		err := acrService.ChangeRevision(r.Context(), app)
// 		require.NoError(t, err)

// 		app, err = acrService.applicationClientset.ArgoprojV1alpha1().Applications(app.Namespace).Get(r.Context(), app.Name, metav1.GetOptions{})
// 		require.NoError(t, err)

// 		assert.Equal(t, "new-revision", app.Status.OperationState.Operation.Sync.ChangeRevision)
// 	})

// 	r.Run("Change revision already exists", func(t *testing.T) {
// 		client := &mocks.ApplicationClient{}
// 		client.On("GetChangeRevision", mock.Anything, mock.Anything).Return(&appclient.ChangeRevisionResponse{
// 			Revision: ptr.To("new-revision"),
// 		}, nil)

// 		logger, logHook := test2.NewNullLogger()

// 		acrService := newTestACRService(client)
// 		acrService.logger = logger

// 		app := createTestApp(syncedAppWithHistory)

// 		err := acrService.ChangeRevision(r.Context(), app)
// 		require.NoError(t, err)

// 		app, err = acrService.applicationClientset.ArgoprojV1alpha1().Applications(app.Namespace).Get(r.Context(), app.Name, metav1.GetOptions{})
// 		require.NoError(t, err)

// 		assert.Equal(t, "new-revision", app.Status.OperationState.Operation.Sync.ChangeRevision)

// 		err = acrService.ChangeRevision(r.Context(), app)

// 		require.NoError(t, err)

// 		lastLogEntry := logHook.LastEntry()
// 		if lastLogEntry == nil {
// 			t.Fatal("No log entry")
// 		}

// 		require.Equal(t, "Change revision already calculated for application guestbook", lastLogEntry.Message)
// 	})
// }

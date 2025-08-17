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

const runningAppWithSingleHistory1Annotated = `
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
    revision: c732f4d2ef24c7eeb900e9211ff98f90bb646506
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
    status: Running
`

const syncedAppWithSingleHistory1Annotated = `
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
    revision: c732f4d2ef24c7eeb900e9211ff98f90bb646506
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
    phase: Synced
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

const syncedAppWithSingleHistory2Annotated = `
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
  - deployStartedAt: "2024-06-20T18:30:00Z"
    deployedAt: "2024-06-20T18:30:01Z"
    id: 2
    initiatedBy: {}
    revision: 1af87672323345954554587665757e0999678678
    source:
      path: apps/guestbook
      repoURL: https://github.com/pasha-codefresh/precisely-gitsource.git
      targetRevision: HEAD
  - deployStartedAt: "2024-06-20T19:35:36Z"
    deployedAt: "2024-06-20T19:35:44Z"
    id: 3
    initiatedBy: {}
    revision: c732f4d2ef24c7eeb900e9211ff98f90bb646506
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
    phase: Synced
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

const multiSourceAppAnnotations = `
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  annotations:
    valid: '["a","b", null]'
    empty-array: '[]'
    empty: ''
    invalid-json: '["fooo"'
    invalid-entries: '["fooo", "2"]'
    invalid-map: '{}'
    invalid-string: '"foo"'
  name: demo-ms-a
  namespace: argocd
`

const syncedMSAppWithSingleHistory1Annotated = `
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  annotations:
    argocd.argoproj.io/manifest-generate-paths: /demo-applications/try-ms02a;/demo-applications/try-ms02b
    mrp-controller.argoproj.io/change-revisions: '["HISTORY-2_REPO02_00000000000000000000000","HISTORY-1_REPO02_00000000000000000000000","HISTORY-1_REPO01_00000000000000000000000","CURRENT_REPO_01_000000000000000000000000"]'
    mrp-controller.argoproj.io/git-revisions:    '["HISTORY-1_REPO02_00000000000000000000000","HISTORY-1_REPO02_00000000000000000000000","CURRENT_REPO_01_000000000000000000000000","CURRENT_REPO_01_000000000000000000000000"]'
  name: demo-ms-a
  namespace: argocd
spec:
  destination:
    name: in-cluster
    namespace: demo-ms-a
  project: default
  sources:
  - path: demo-applications/try-ms02a
    repoURL: https://github.com/dudinea/cfrepo02.git
    targetRevision: dev
  - path: demo-applications/try-ms02b
    repoURL: https://github.com/dudinea/cfrepo02.git
    targetRevision: dev
  - path: demo-applications/try-ms01a
    repoURL: https://github.com/dudinea/cfrepo01.git
    targetRevision: main
  - path: demo-applications/try-ms01b
    repoURL: https://github.com/dudinea/cfrepo01.git
    targetRevision: main
  syncPolicy:
    automated:
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
    lastTransitionTime: "2025-08-10T12:01:35Z"
    status: Healthy
  history:
  - deployStartedAt: "2025-08-10T11:39:57Z"
    deployedAt: "2025-08-10T11:39:57Z"
    id: 3
    initiatedBy:
      username: admin
    revisions:
    - HISTORY-2_REPO02_00000000000000000000000
    - HISTORY-2_REPO02_00000000000000000000000
    - HISTORY-1_REPO01_00000000000000000000000
    - HISTORY-1_REPO01_00000000000000000000000
    source:
      repoURL: ""
    sources:
    - path: demo-applications/try-ms02a
      repoURL: https://github.com/dudinea/cfrepo02.git
      targetRevision: dev
    - path: demo-applications/try-ms02b
      repoURL: https://github.com/dudinea/cfrepo02.git
      targetRevision: dev
    - path: demo-applications/try-ms01a
      repoURL: https://github.com/dudinea/cfrepo01.git
      targetRevision: main
    - path: demo-applications/try-ms01b
      repoURL: https://github.com/dudinea/cfrepo01.git
      targetRevision: main
  - deployStartedAt: "2025-08-10T11:52:03Z"
    deployedAt: "2025-08-10T11:52:04Z"
    id: 4
    initiatedBy:
      username: admin
    revisions:
    - HISTORY-1_REPO02_00000000000000000000000
    - HISTORY-1_REPO02_00000000000000000000000
    - CURRENT_REPO_01_000000000000000000000000
    - CURRENT_REPO_01_000000000000000000000000
    source:
      repoURL: ""
    sources:
    - path: demo-applications/try-ms02a
      repoURL: https://github.com/dudinea/cfrepo02.git
      targetRevision: dev
    - path: demo-applications/try-ms02b
      repoURL: https://github.com/dudinea/cfrepo02.git
      targetRevision: dev
    - path: demo-applications/try-ms01a
      repoURL: https://github.com/dudinea/cfrepo01.git
      targetRevision: main
    - path: demo-applications/try-ms01b
      repoURL: https://github.com/dudinea/cfrepo01.git
      targetRevision: main
  - deployStartedAt: "2025-08-10T12:01:33Z"
    deployedAt: "2025-08-10T12:01:33Z"
    id: 5
    initiatedBy:
      automated: true
    revisions:
    - HISTORY-1_REPO02_00000000000000000000000
    - HISTORY-1_REPO02_00000000000000000000000
    - CURRENT_REPO_01_000000000000000000000000
    - CURRENT_REPO_01_000000000000000000000000
    source:
      repoURL: ""
    sources:
    - path: demo-applications/try-ms02a
      repoURL: https://github.com/dudinea/cfrepo02.git
      targetRevision: dev
    - path: demo-applications/try-ms02b
      repoURL: https://github.com/dudinea/cfrepo02.git
      targetRevision: dev
    - path: demo-applications/try-ms01a
      repoURL: https://github.com/dudinea/cfrepo01.git
      targetRevision: main
    - path: demo-applications/try-ms01b
      repoURL: https://github.com/dudinea/cfrepo01.git
      targetRevision: main
  operationState:
    finishedAt: "2025-08-10T12:01:33Z"
    message: successfully synced (all tasks run)
    operation:
      initiatedBy:
        automated: true
      retry:
        limit: 5
      sync:
        prune: true
        revisions:
        - CURRENT_REPO_02_000000000000000000000000
        - CURRENT_REPO_02_000000000000000000000000
        - CURRENT_REPO_01_000000000000000000000000
        - CURRENT_REPO_01_000000000000000000000000
        syncOptions:
        - PrunePropagationPolicy=foreground
        - Replace=false
        - PruneLast=false
        - Validate=true
        - CreateNamespace=true
        - ApplyOutOfSyncOnly=false
        - ServerSideApply=true
        - RespectIgnoreDifferences=false
    phase: Succeeded
    startedAt: "2025-08-10T12:01:33Z"
    syncResult:
      resources:
      - group: ""
        hookPhase: Running
        kind: ConfigMap
        message: configmap/config-cm-ms02a serverside-applied
        name: config-cm-ms02a
        namespace: demo-ms-a
        status: Synced
        syncPhase: Sync
        version: v1
      - group: ""
        hookPhase: Running
        kind: ConfigMap
        message: configmap/config-cm-ms02b serverside-applied
        name: config-cm-ms02b
        namespace: demo-ms-a
        status: Synced
        syncPhase: Sync
        version: v1
      - group: ""
        hookPhase: Running
        kind: ConfigMap
        message: configmap/config-cm-ms01b serverside-applied
        name: config-cm-ms01b
        namespace: demo-ms-a
        status: Synced
        syncPhase: Sync
        version: v1
      - group: ""
        hookPhase: Running
        kind: ConfigMap
        message: configmap/config-cm-ms01a serverside-applied
        name: config-cm-ms01a
        namespace: demo-ms-a
        status: Synced
        syncPhase: Sync
        version: v1
      revision: ""
      revisions:
      - CURRENT_REPO_02_000000000000000000000000
      - CURRENT_REPO_02_000000000000000000000000
      - CURRENT_REPO_01_000000000000000000000000
      - CURRENT_REPO_01_000000000000000000000000
      source:
        repoURL: ""
      sources:
      - path: demo-applications/try-ms02a
        repoURL: https://github.com/dudinea/cfrepo02.git
        targetRevision: dev
      - path: demo-applications/try-ms02b
        repoURL: https://github.com/dudinea/cfrepo02.git
        targetRevision: dev
      - path: demo-applications/try-ms01a
        repoURL: https://github.com/dudinea/cfrepo01.git
        targetRevision: main
      - path: demo-applications/try-ms01b
        repoURL: https://github.com/dudinea/cfrepo01.git
        targetRevision: main
  reconciledAt: "2025-08-11T14:05:27Z"
  resourceHealthSource: appTree
  resources:
  - kind: ConfigMap
    name: config-cm-ms01a
    namespace: demo-ms-a
    status: Synced
    version: v1
  - kind: ConfigMap
    name: config-cm-ms01b
    namespace: demo-ms-a
    status: Synced
    version: v1
  - kind: ConfigMap
    name: config-cm-ms02a
    namespace: demo-ms-a
    status: Synced
    version: v1
  - kind: ConfigMap
    name: config-cm-ms02b
    namespace: demo-ms-a
    status: Synced
    version: v1
  sourceHydrator: {}
  sourceTypes:
  - Directory
  - Directory
  - Directory
  - Directory
  summary: {}
  sync:
    comparedTo:
      destination:
        name: in-cluster
        namespace: demo-ms-a
      source:
        repoURL: ""
      sources:
      - path: demo-applications/try-ms02a
        repoURL: https://github.com/dudinea/cfrepo02.git
        targetRevision: dev
      - path: demo-applications/try-ms02b
        repoURL: https://github.com/dudinea/cfrepo02.git
        targetRevision: dev
      - path: demo-applications/try-ms01a
        repoURL: https://github.com/dudinea/cfrepo01.git
        targetRevision: main
      - path: demo-applications/try-ms01b
        repoURL: https://github.com/dudinea/cfrepo01.git
        targetRevision: main
    revisions:
    - CURRENT_REPO_02_000000000000000000000000
    - CURRENT_REPO_02_000000000000000000000000
    - CURRENT_REPO_01_000000000000000000000000
    - CURRENT_REPO_01_000000000000000000000000
    status: Synced
`

const syncedMSAppWithSingleHistory2Annotated = `
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  annotations:
    argocd.argoproj.io/manifest-generate-paths: /demo-applications/try-ms02a;/demo-applications/try-ms02b
    mrp-controller.argoproj.io/change-revisions: '["HISTORY-2_REPO02_00000000000000000000000","HISTORY-1_REPO02_00000000000000000000000","HISTORY-1_REPO01_00000000000000000000000","CURRENT_REPO_01_000000000000000000000000"]'
    mrp-controller.argoproj.io/git-revisions:    '["HISTORY-1_REPO02_00000000000000000000000","HISTORY-1_REPO02_00000000000000000000000","CURRENT_REPO_01_000000000000000000000000","CURRENT_REPO_01_000000000000000000000000"]'
  name: demo-ms-a
  namespace: argocd
spec:
  destination:
    name: in-cluster
    namespace: demo-ms-a
  project: default
  sources:
  - path: demo-applications/try-ms02a
    repoURL: https://github.com/dudinea/cfrepo02.git
    targetRevision: dev
  - path: demo-applications/try-ms02b
    repoURL: https://github.com/dudinea/cfrepo02.git
    targetRevision: dev
  - path: demo-applications/try-ms01a
    repoURL: https://github.com/dudinea/cfrepo01.git
    targetRevision: main
  - path: demo-applications/try-ms01b
    repoURL: https://github.com/dudinea/cfrepo01.git
    targetRevision: main
  syncPolicy:
    automated:
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
    lastTransitionTime: "2025-08-10T12:01:35Z"
    status: Healthy
  history:
  - deployStartedAt: "2025-08-10T11:39:57Z"
    deployedAt: "2025-08-10T11:39:57Z"
    id: 3
    initiatedBy:
      username: admin
    revisions:
    - HISTORY-2_REPO02_00000000000000000000000
    - HISTORY-2_REPO02_00000000000000000000000
    - HISTORY-1_REPO01_00000000000000000000000
    - HISTORY-1_REPO01_00000000000000000000000
    source:
      repoURL: ""
    sources:
    - path: demo-applications/try-ms02b
      repoURL: https://github.com/dudinea/cfrepo02.git
      targetRevision: dev
    - path: demo-applications/try-ms01a
      repoURL: https://github.com/dudinea/cfrepo01.git
      targetRevision: main
    - path: demo-applications/try-ms02a
      repoURL: https://github.com/dudinea/cfrepo02.git
      targetRevision: dev
    - path: demo-applications/try-ms01b
      repoURL: https://github.com/dudinea/cfrepo01.git
      targetRevision: main
  - deployStartedAt: "2025-08-10T11:52:03Z"
    deployedAt: "2025-08-10T11:52:04Z"
    id: 4
    initiatedBy:
      username: admin
    revisions:
    - HISTORY-1_REPO02_00000000000000000000000
    - CURRENT_REPO_01_000000000000000000000000
    - HISTORY-1_REPO02_00000000000000000000000
    - CURRENT_REPO_01_000000000000000000000000
    source:
      repoURL: ""
    sources:
    - path: demo-applications/try-ms02b
      repoURL: https://github.com/dudinea/cfrepo02.git
      targetRevision: dev
    - path: demo-applications/try-ms01a
      repoURL: https://github.com/dudinea/cfrepo01.git
      targetRevision: main
    - path: demo-applications/try-ms02a
      repoURL: https://github.com/dudinea/cfrepo02.git
      targetRevision: dev
    - path: demo-applications/try-ms01b
      repoURL: https://github.com/dudinea/cfrepo01.git
      targetRevision: main
  - deployStartedAt: "2025-08-10T12:01:33Z"
    deployedAt: "2025-08-10T12:01:33Z"
    id: 5
    initiatedBy:
      automated: true
    revisions:
    - HISTORY-1_REPO02_00000000000000000000000
    - HISTORY-1_REPO02_00000000000000000000000
    - CURRENT_REPO_01_000000000000000000000000
    - CURRENT_REPO_01_000000000000000000000000
    source:
      repoURL: ""
    sources:
    - path: demo-applications/try-ms02a
      repoURL: https://github.com/dudinea/cfrepo02.git
      targetRevision: dev
    - path: demo-applications/try-ms02b
      repoURL: https://github.com/dudinea/cfrepo02.git
      targetRevision: dev
    - path: demo-applications/try-ms01a
      repoURL: https://github.com/dudinea/cfrepo01.git
      targetRevision: main
    - path: demo-applications/try-ms01b
      repoURL: https://github.com/dudinea/cfrepo01.git
      targetRevision: main
  operationState:
    finishedAt: "2025-08-10T12:01:33Z"
    message: successfully synced (all tasks run)
    operation:
      initiatedBy:
        automated: true
      retry:
        limit: 5
      sync:
        prune: true
        revisions:
        - CURRENT_REPO_02_000000000000000000000000
        - CURRENT_REPO_02_000000000000000000000000
        - CURRENT_REPO_01_000000000000000000000000
        - CURRENT_REPO_01_000000000000000000000000
        syncOptions:
        - PrunePropagationPolicy=foreground
        - Replace=false
        - PruneLast=false
        - Validate=true
        - CreateNamespace=true
        - ApplyOutOfSyncOnly=false
        - ServerSideApply=true
        - RespectIgnoreDifferences=false
    phase: Succeeded
    startedAt: "2025-08-10T12:01:33Z"
    syncResult:
      resources:
      - group: ""
        hookPhase: Running
        kind: ConfigMap
        message: configmap/config-cm-ms02a serverside-applied
        name: config-cm-ms02a
        namespace: demo-ms-a
        status: Synced
        syncPhase: Sync
        version: v1
      - group: ""
        hookPhase: Running
        kind: ConfigMap
        message: configmap/config-cm-ms02b serverside-applied
        name: config-cm-ms02b
        namespace: demo-ms-a
        status: Synced
        syncPhase: Sync
        version: v1
      - group: ""
        hookPhase: Running
        kind: ConfigMap
        message: configmap/config-cm-ms01b serverside-applied
        name: config-cm-ms01b
        namespace: demo-ms-a
        status: Synced
        syncPhase: Sync
        version: v1
      - group: ""
        hookPhase: Running
        kind: ConfigMap
        message: configmap/config-cm-ms01a serverside-applied
        name: config-cm-ms01a
        namespace: demo-ms-a
        status: Synced
        syncPhase: Sync
        version: v1
      revision: ""
      revisions:
      - CURRENT_REPO_02_000000000000000000000000
      - CURRENT_REPO_02_000000000000000000000000
      - CURRENT_REPO_01_000000000000000000000000
      - CURRENT_REPO_01_000000000000000000000000
      source:
        repoURL: ""
      sources:
      - path: demo-applications/try-ms02a
        repoURL: https://github.com/dudinea/cfrepo02.git
        targetRevision: dev
      - path: demo-applications/try-ms02b
        repoURL: https://github.com/dudinea/cfrepo02.git
        targetRevision: dev
      - path: demo-applications/try-ms01a
        repoURL: https://github.com/dudinea/cfrepo01.git
        targetRevision: main
      - path: demo-applications/try-ms01b
        repoURL: https://github.com/dudinea/cfrepo01.git
        targetRevision: main
  reconciledAt: "2025-08-11T14:05:27Z"
  resourceHealthSource: appTree
  resources:
  - kind: ConfigMap
    name: config-cm-ms01a
    namespace: demo-ms-a
    status: Synced
    version: v1
  - kind: ConfigMap
    name: config-cm-ms01b
    namespace: demo-ms-a
    status: Synced
    version: v1
  - kind: ConfigMap
    name: config-cm-ms02a
    namespace: demo-ms-a
    status: Synced
    version: v1
  - kind: ConfigMap
    name: config-cm-ms02b
    namespace: demo-ms-a
    status: Synced
    version: v1
  sourceHydrator: {}
  sourceTypes:
  - Directory
  - Directory
  - Directory
  - Directory
  summary: {}
  sync:
    comparedTo:
      destination:
        name: in-cluster
        namespace: demo-ms-a
      source:
        repoURL: ""
      sources:
      - path: demo-applications/try-ms02a
        repoURL: https://github.com/dudinea/cfrepo02.git
        targetRevision: dev
      - path: demo-applications/try-ms02b
        repoURL: https://github.com/dudinea/cfrepo02.git
        targetRevision: dev
      - path: demo-applications/try-ms01a
        repoURL: https://github.com/dudinea/cfrepo01.git
        targetRevision: main
      - path: demo-applications/try-ms01b
        repoURL: https://github.com/dudinea/cfrepo01.git
        targetRevision: main
    revisions:
    - CURRENT_REPO_02_000000000000000000000000
    - CURRENT_REPO_02_000000000000000000000000
    - CURRENT_REPO_01_000000000000000000000000
    - CURRENT_REPO_01_000000000000000000000000
    status: Synced
`

const syncedMSAppWithSingleHistory3Annotated = `
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  annotations:
    argocd.argoproj.io/manifest-generate-paths: /demo-applications/try-ms02a;/demo-applications/try-ms02b
    mrp-controller.argoproj.io/change-revisions: '["HISTORY-2_REPO02_00000000000000000000000","HISTORY-1_REPO02_00000000000000000000000","HISTORY-1_REPO01_00000000000000000000000","CURRENT_REPO_01_000000000000000000000000"]'
    mrp-controller.argoproj.io/git-revisions:    '["HISTORY-1_REPO02_00000000000000000000000","HISTORY-1_REPO02_00000000000000000000000","CURRENT_REPO_01_000000000000000000000000","CURRENT_REPO_01_000000000000000000000000"]'
  name: demo-ms-a
  namespace: argocd
spec:
  destination:
    name: in-cluster
    namespace: demo-ms-a
  project: default
  sources:
  - path: demo-applications/try-ms02a
    repoURL: https://github.com/dudinea/cfrepo02.git
    targetRevision: dev
  - path: demo-applications/try-ms02b
    repoURL: https://github.com/dudinea/cfrepo02.git
    targetRevision: dev
  - path: demo-applications/try-ms01a
    repoURL: https://github.com/dudinea/cfrepo01.git
    targetRevision: main
  - path: demo-applications/try-ms01b
    repoURL: https://github.com/dudinea/cfrepo01.git
    targetRevision: main
  syncPolicy:
    automated:
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
    lastTransitionTime: "2025-08-10T12:01:35Z"
    status: Healthy
  history:
  - deployStartedAt: "2025-08-10T11:39:57Z"
    deployedAt: "2025-08-10T11:39:57Z"
    id: 3
    initiatedBy:
      username: admin
    revisions:
    - HISTORY-2_REPO02_00000000000000000000000
    source:
      repoURL: ""
    sources:
    - path: demo-applications/try-ms01a
      repoURL: https://github.com/dudinea/cfrepo01.git
      targetRevision: main
  - deployStartedAt: "2025-08-10T11:52:03Z"
    deployedAt: "2025-08-10T11:52:04Z"
    id: 4
    initiatedBy:
      username: admin
    revisions:
    - HISTORY-1_REPO02_00000000000000000000000
    - CURRENT_REPO_01_000000000000000000000000
    - HISTORY-1_REPO02_00000000000000000000000
    - CURRENT_REPO_01_000000000000000000000000
    source:
      repoURL: ""
    sources:
    - path: demo-applications/try-ms02b
      repoURL: https://github.com/dudinea/cfrepo02.git
      targetRevision: dev
    - path: demo-applications/try-ms01a
      repoURL: https://github.com/dudinea/cfrepo01.git
      targetRevision: main
    - path: demo-applications/try-ms02a
      repoURL: https://github.com/dudinea/cfrepo02.git
      targetRevision: dev
    - path: demo-applications/try-ms01b
      repoURL: https://github.com/dudinea/cfrepo01.git
      targetRevision: main
  - deployStartedAt: "2025-08-10T12:01:33Z"
    deployedAt: "2025-08-10T12:01:33Z"
    id: 5
    initiatedBy:
      automated: true
    revisions:
    - HISTORY-1_REPO02_00000000000000000000000
    - HISTORY-1_REPO02_00000000000000000000000
    - CURRENT_REPO_01_000000000000000000000000
    - CURRENT_REPO_01_000000000000000000000000
    source:
      repoURL: ""
    sources:
    - path: demo-applications/try-ms02a
      repoURL: https://github.com/dudinea/cfrepo02.git
      targetRevision: dev
    - path: demo-applications/try-ms02b
      repoURL: https://github.com/dudinea/cfrepo02.git
      targetRevision: dev
    - path: demo-applications/try-ms01a
      repoURL: https://github.com/dudinea/cfrepo01.git
      targetRevision: main
    - path: demo-applications/try-ms01b
      repoURL: https://github.com/dudinea/cfrepo01.git
      targetRevision: main
  operationState:
    finishedAt: "2025-08-10T12:01:33Z"
    message: successfully synced (all tasks run)
    operation:
      initiatedBy:
        automated: true
      retry:
        limit: 5
      sync:
        prune: true
        revisions:
        - CURRENT_REPO_02_000000000000000000000000
        - CURRENT_REPO_02_000000000000000000000000
        - CURRENT_REPO_01_000000000000000000000000
        - CURRENT_REPO_01_000000000000000000000000
        syncOptions:
        - PrunePropagationPolicy=foreground
        - Replace=false
        - PruneLast=false
        - Validate=true
        - CreateNamespace=true
        - ApplyOutOfSyncOnly=false
        - ServerSideApply=true
        - RespectIgnoreDifferences=false
    phase: Succeeded
    startedAt: "2025-08-10T12:01:33Z"
    syncResult:
      resources:
      - group: ""
        hookPhase: Running
        kind: ConfigMap
        message: configmap/config-cm-ms02a serverside-applied
        name: config-cm-ms02a
        namespace: demo-ms-a
        status: Synced
        syncPhase: Sync
        version: v1
      - group: ""
        hookPhase: Running
        kind: ConfigMap
        message: configmap/config-cm-ms02b serverside-applied
        name: config-cm-ms02b
        namespace: demo-ms-a
        status: Synced
        syncPhase: Sync
        version: v1
      - group: ""
        hookPhase: Running
        kind: ConfigMap
        message: configmap/config-cm-ms01b serverside-applied
        name: config-cm-ms01b
        namespace: demo-ms-a
        status: Synced
        syncPhase: Sync
        version: v1
      - group: ""
        hookPhase: Running
        kind: ConfigMap
        message: configmap/config-cm-ms01a serverside-applied
        name: config-cm-ms01a
        namespace: demo-ms-a
        status: Synced
        syncPhase: Sync
        version: v1
      revision: ""
      revisions:
      - CURRENT_REPO_02_000000000000000000000000
      - CURRENT_REPO_02_000000000000000000000000
      - CURRENT_REPO_01_000000000000000000000000
      - CURRENT_REPO_01_000000000000000000000000
      source:
        repoURL: ""
      sources:
      - path: demo-applications/try-ms02a
        repoURL: https://github.com/dudinea/cfrepo02.git
        targetRevision: dev
      - path: demo-applications/try-ms02b
        repoURL: https://github.com/dudinea/cfrepo02.git
        targetRevision: dev
      - path: demo-applications/try-ms01a
        repoURL: https://github.com/dudinea/cfrepo01.git
        targetRevision: main
      - path: demo-applications/try-ms01b
        repoURL: https://github.com/dudinea/cfrepo01.git
        targetRevision: main
  reconciledAt: "2025-08-11T14:05:27Z"
  resourceHealthSource: appTree
  resources:
  - kind: ConfigMap
    name: config-cm-ms01a
    namespace: demo-ms-a
    status: Synced
    version: v1
  - kind: ConfigMap
    name: config-cm-ms01b
    namespace: demo-ms-a
    status: Synced
    version: v1
  - kind: ConfigMap
    name: config-cm-ms02a
    namespace: demo-ms-a
    status: Synced
    version: v1
  - kind: ConfigMap
    name: config-cm-ms02b
    namespace: demo-ms-a
    status: Synced
    version: v1
  sourceHydrator: {}
  sourceTypes:
  - Directory
  - Directory
  - Directory
  - Directory
  summary: {}
  sync:
    comparedTo:
      destination:
        name: in-cluster
        namespace: demo-ms-a
      source:
        repoURL: ""
      sources:
      - path: demo-applications/try-ms02a
        repoURL: https://github.com/dudinea/cfrepo02.git
        targetRevision: dev
      - path: demo-applications/try-ms02b
        repoURL: https://github.com/dudinea/cfrepo02.git
        targetRevision: dev
      - path: demo-applications/try-ms01a
        repoURL: https://github.com/dudinea/cfrepo01.git
        targetRevision: main
      - path: demo-applications/try-ms01b
        repoURL: https://github.com/dudinea/cfrepo01.git
        targetRevision: main
    revisions:
    - CURRENT_REPO_02_000000000000000000000000
    - CURRENT_REPO_02_000000000000000000000000
    - CURRENT_REPO_01_000000000000000000000000
    - CURRENT_REPO_01_000000000000000000000000
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

func Test_getArrayFromAnnotation(t *testing.T) {
	anapp := createTestApp(t, multiSourceAppAnnotations)
	mrpService := newTestMRPService(t, nil, &mocks.Interface{}, nil)
	arr := mrpService.getArrayFromAnnotation(anapp, "valid")
	assert.NotNil(t, arr)
	assert.Equal(t, 3, len(arr))
	assert.Equal(t, arr[0], "a")
	assert.Equal(t, arr[1], "b")
	assert.Equal(t, arr[2], "")

	arr = mrpService.getArrayFromAnnotation(anapp, "empty-array")
	assert.NotNil(t, arr)
	assert.Equal(t, 0, len(arr))

	arr = mrpService.getArrayFromAnnotation(anapp, "empty")
	assert.Nil(t, arr)
	assert.Equal(t, 0, len(arr))

	arr = mrpService.getArrayFromAnnotation(anapp, "invalid-json")
	assert.Nil(t, arr)
	assert.Equal(t, 0, len(arr))

	arr = mrpService.getArrayFromAnnotation(anapp, "invalid-map")
	assert.Nil(t, arr)
	assert.Equal(t, 0, len(arr))

	arr = mrpService.getArrayFromAnnotation(anapp, "invalid-string")
	assert.Nil(t, arr)
	assert.Equal(t, 0, len(arr))

	arr = mrpService.getArrayFromAnnotation(anapp, "unknown-annotation")
	assert.Nil(t, arr)
	assert.Equal(t, 0, len(arr))

}

func Test_GetSourceRevisionsSSWithoutHistory(t *testing.T) {
	anapp := createTestApp(t, syncedAppWithoutHistory)
	mrpService := newTestMRPService(t, nil, &mocks.Interface{}, nil)
	sourcesRevisions := mrpService.getSourcesRevisions(anapp)
	//changeRevision, gitRevision, currentRevision, previousRevision := getApplicationRevisions(anapp, -1)
	assert.NotNil(t, sourcesRevisions)
	assert.Equal(t, 1, len(sourcesRevisions))
	assert.Equal(t, "2b571ad9ceaab7ed1e6225ca674e367f2d07e41d", sourcesRevisions[0].currentRevision)
	assert.Equal(t, "", sourcesRevisions[0].previousRevision)
	assert.Equal(t, "", sourcesRevisions[0].gitRevision)
	assert.Equal(t, "", sourcesRevisions[0].changeRevision)
}

func Test_GetSourceRevisionsSSWithHistory1Running(t *testing.T) {
	anapp := createTestApp(t, runningAppWithSingleHistory1Annotated)
	mrpService := newTestMRPService(t, nil, &mocks.Interface{}, nil)
	sourcesRevisions := mrpService.getSourcesRevisions(anapp)
	assert.NotNil(t, sourcesRevisions)
	assert.Equal(t, 1, len(sourcesRevisions))
	assert.Equal(t, "00d423763fbf56d2ea452de7b26a0ab20590f521", sourcesRevisions[0].gitRevision)
	assert.Equal(t, "792822850fd2f6db63597533e16dfa27e6757dc5", sourcesRevisions[0].changeRevision)
	assert.Equal(t, "c732f4d2ef24c7eeb900e9211ff98f90bb646506", sourcesRevisions[0].currentRevision)
	assert.Equal(t, "792822850fd2f6db63597533e16dfa27e6757dc5", sourcesRevisions[0].previousRevision)
}

func Test_GetSourceRevisionsSSWithHistory1Synced(t *testing.T) {
	anapp := createTestApp(t, syncedAppWithSingleHistory1Annotated)
	mrpService := newTestMRPService(t, nil, &mocks.Interface{}, nil)
	sourcesRevisions := mrpService.getSourcesRevisions(anapp)
	assert.NotNil(t, sourcesRevisions)
	assert.Equal(t, 1, len(sourcesRevisions))
	assert.Equal(t, "00d423763fbf56d2ea452de7b26a0ab20590f521", sourcesRevisions[0].gitRevision)
	assert.Equal(t, "792822850fd2f6db63597533e16dfa27e6757dc5", sourcesRevisions[0].changeRevision)
	assert.Equal(t, "c732f4d2ef24c7eeb900e9211ff98f90bb646506", sourcesRevisions[0].currentRevision)
	assert.Equal(t, "", sourcesRevisions[0].previousRevision)
}

func Test_GetSourceRevisionsSSWithHistory2Synced(t *testing.T) {
	anapp := createTestApp(t, syncedAppWithSingleHistory2Annotated)
	mrpService := newTestMRPService(t, nil, &mocks.Interface{}, nil)
	sourcesRevisions := mrpService.getSourcesRevisions(anapp)
	assert.NotNil(t, sourcesRevisions)
	assert.Equal(t, 1, len(sourcesRevisions))
	assert.Equal(t, "00d423763fbf56d2ea452de7b26a0ab20590f521", sourcesRevisions[0].gitRevision)
	assert.Equal(t, "792822850fd2f6db63597533e16dfa27e6757dc5", sourcesRevisions[0].changeRevision)
	assert.Equal(t, "c732f4d2ef24c7eeb900e9211ff98f90bb646506", sourcesRevisions[0].currentRevision)
	assert.Equal(t, "1af87672323345954554587665757e0999678678", sourcesRevisions[0].previousRevision)
}

func Test_GetSourceRevisionsMSWithHistory(t *testing.T) {
	anapp := createTestApp(t, syncedMSAppWithSingleHistory1Annotated)
	mrpService := newTestMRPService(t, nil, &mocks.Interface{}, nil)
	sourcesRevisions := mrpService.getSourcesRevisions(anapp)
	assert.NotNil(t, sourcesRevisions)
	assert.Equal(t, 4, len(sourcesRevisions))

	assert.Equal(t, "HISTORY-1_REPO02_00000000000000000000000", sourcesRevisions[0].gitRevision)
	assert.Equal(t, "HISTORY-2_REPO02_00000000000000000000000", sourcesRevisions[0].changeRevision)
	assert.Equal(t, "CURRENT_REPO_02_000000000000000000000000", sourcesRevisions[0].currentRevision)
	assert.Equal(t, "HISTORY-1_REPO02_00000000000000000000000", sourcesRevisions[0].previousRevision)

	assert.Equal(t, "HISTORY-1_REPO02_00000000000000000000000", sourcesRevisions[1].gitRevision)
	assert.Equal(t, "HISTORY-1_REPO02_00000000000000000000000", sourcesRevisions[1].changeRevision)
	assert.Equal(t, "CURRENT_REPO_02_000000000000000000000000", sourcesRevisions[1].currentRevision)
	assert.Equal(t, "HISTORY-1_REPO02_00000000000000000000000", sourcesRevisions[1].previousRevision)

	assert.Equal(t, "CURRENT_REPO_01_000000000000000000000000", sourcesRevisions[2].gitRevision)
	assert.Equal(t, "HISTORY-1_REPO01_00000000000000000000000", sourcesRevisions[2].changeRevision)
	assert.Equal(t, "CURRENT_REPO_01_000000000000000000000000", sourcesRevisions[2].currentRevision)
	assert.Equal(t, "CURRENT_REPO_01_000000000000000000000000", sourcesRevisions[2].previousRevision)

	assert.Equal(t, "CURRENT_REPO_01_000000000000000000000000", sourcesRevisions[3].gitRevision)
	assert.Equal(t, "CURRENT_REPO_01_000000000000000000000000", sourcesRevisions[3].changeRevision)
	assert.Equal(t, "CURRENT_REPO_01_000000000000000000000000", sourcesRevisions[3].currentRevision)
	assert.Equal(t, "CURRENT_REPO_01_000000000000000000000000", sourcesRevisions[3].previousRevision)
}

func Test_GetSourceRevisionsMSWithHistorySwapped(t *testing.T) {
	anapp := createTestApp(t, syncedMSAppWithSingleHistory2Annotated)
	mrpService := newTestMRPService(t, nil, &mocks.Interface{}, nil)
	sourcesRevisions := mrpService.getSourcesRevisions(anapp)
	assert.NotNil(t, sourcesRevisions)
	assert.Equal(t, 4, len(sourcesRevisions))

	assert.Equal(t, "HISTORY-1_REPO02_00000000000000000000000", sourcesRevisions[0].gitRevision)
	assert.Equal(t, "HISTORY-2_REPO02_00000000000000000000000", sourcesRevisions[0].changeRevision)
	assert.Equal(t, "CURRENT_REPO_02_000000000000000000000000", sourcesRevisions[0].currentRevision)
	assert.Equal(t, "HISTORY-1_REPO02_00000000000000000000000", sourcesRevisions[0].previousRevision)

	assert.Equal(t, "HISTORY-1_REPO02_00000000000000000000000", sourcesRevisions[1].gitRevision)
	assert.Equal(t, "HISTORY-1_REPO02_00000000000000000000000", sourcesRevisions[1].changeRevision)
	assert.Equal(t, "CURRENT_REPO_02_000000000000000000000000", sourcesRevisions[1].currentRevision)
	assert.Equal(t, "HISTORY-1_REPO02_00000000000000000000000", sourcesRevisions[1].previousRevision)

	assert.Equal(t, "CURRENT_REPO_01_000000000000000000000000", sourcesRevisions[2].gitRevision)
	assert.Equal(t, "HISTORY-1_REPO01_00000000000000000000000", sourcesRevisions[2].changeRevision)
	assert.Equal(t, "CURRENT_REPO_01_000000000000000000000000", sourcesRevisions[2].currentRevision)
	assert.Equal(t, "CURRENT_REPO_01_000000000000000000000000", sourcesRevisions[2].previousRevision)

	assert.Equal(t, "CURRENT_REPO_01_000000000000000000000000", sourcesRevisions[3].gitRevision)
	assert.Equal(t, "CURRENT_REPO_01_000000000000000000000000", sourcesRevisions[3].changeRevision)
	assert.Equal(t, "CURRENT_REPO_01_000000000000000000000000", sourcesRevisions[3].currentRevision)
	assert.Equal(t, "CURRENT_REPO_01_000000000000000000000000", sourcesRevisions[3].previousRevision)
}

func Test_GetSourceRevisionsMSWithHistoryAdded(t *testing.T) {
	anapp := createTestApp(t, syncedMSAppWithSingleHistory3Annotated)
	mrpService := newTestMRPService(t, nil, &mocks.Interface{}, nil)
	sourcesRevisions := mrpService.getSourcesRevisions(anapp)
	assert.NotNil(t, sourcesRevisions)
	assert.Equal(t, 4, len(sourcesRevisions))

	assert.Equal(t, "HISTORY-1_REPO02_00000000000000000000000", sourcesRevisions[0].gitRevision)
	assert.Equal(t, "HISTORY-2_REPO02_00000000000000000000000", sourcesRevisions[0].changeRevision)
	assert.Equal(t, "CURRENT_REPO_02_000000000000000000000000", sourcesRevisions[0].currentRevision)
	assert.Equal(t, "HISTORY-1_REPO02_00000000000000000000000", sourcesRevisions[0].previousRevision)

	assert.Equal(t, "HISTORY-1_REPO02_00000000000000000000000", sourcesRevisions[1].gitRevision)
	assert.Equal(t, "HISTORY-1_REPO02_00000000000000000000000", sourcesRevisions[1].changeRevision)
	assert.Equal(t, "CURRENT_REPO_02_000000000000000000000000", sourcesRevisions[1].currentRevision)
	assert.Equal(t, "HISTORY-1_REPO02_00000000000000000000000", sourcesRevisions[1].previousRevision)

	assert.Equal(t, "CURRENT_REPO_01_000000000000000000000000", sourcesRevisions[2].gitRevision)
	assert.Equal(t, "HISTORY-1_REPO01_00000000000000000000000", sourcesRevisions[2].changeRevision)
	assert.Equal(t, "CURRENT_REPO_01_000000000000000000000000", sourcesRevisions[2].currentRevision)
	assert.Equal(t, "CURRENT_REPO_01_000000000000000000000000", sourcesRevisions[2].previousRevision)

	assert.Equal(t, "CURRENT_REPO_01_000000000000000000000000", sourcesRevisions[3].gitRevision)
	assert.Equal(t, "CURRENT_REPO_01_000000000000000000000000", sourcesRevisions[3].changeRevision)
	assert.Equal(t, "CURRENT_REPO_01_000000000000000000000000", sourcesRevisions[3].currentRevision)
	assert.Equal(t, "CURRENT_REPO_01_000000000000000000000000", sourcesRevisions[3].previousRevision)
}

func Test_GetApplicationRevisionsWithoutHistory(t *testing.T) {
	anapp := createTestApp(t, syncedAppWithoutHistory)
	mrpService := newTestMRPService(t, nil, &mocks.Interface{}, nil)
	sourceRevisions := mrpService.getSourcesRevisions(anapp)
	assert.Equal(t, 1, len(sourceRevisions))
	assert.Equal(t, "2b571ad9ceaab7ed1e6225ca674e367f2d07e41d", sourceRevisions[0].currentRevision)
	assert.Empty(t, sourceRevisions[0].previousRevision)
	assert.Empty(t, sourceRevisions[0].changeRevision)
	assert.Empty(t, sourceRevisions[0].gitRevision)
}

// func Test_CalculateRevision_no_paths(t *testing.T) {
// 	mrpService := newTestMRPService(t,
// 		&repomocks.Clientset{},
// 		&mocks.Interface{},
// 		&dbmocks.ArgoDB{})
// 	app := createTestApp(t, fakeApp)
// 	revision, err := mrpService.calculateChangeRevision(t.Context(), app, "", "")
// 	assert.Nil(t, revision)
// 	require.Error(t, err)
// 	assert.Equal(t, "rpc error: code = FailedPrecondition desc = manifest generation paths not set", err.Error())
// }

func Test_CalculateRevision(t *testing.T) {
	expectedRevision := "ffffffffffffffffffffffffffffffffffffffff"
	repo := appsv1.Repository{Repo: "myrepo"}
	app := createTestApp(t, syncedAppWithSingleHistory1Annotated)
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
	currentRevision, previousRevision := getRevisionsSingleSource(app)
	revision, err := mrpService.calculateChangeRevision(t.Context(), app, currentRevision, previousRevision)
	require.NoError(t, err)
	assert.NotNil(t, revision)
	assert.Equal(t, expectedRevision, *revision)
}

// WHY FAILS?
// func Test_ChangeRevision(t *testing.T) {
// 	expectedRevision := "ffffffffffffffffffffffffffffffffffffffff"
// 	repo := appsv1.Repository{Repo: "myrepo"}
// 	app := createTestApp(t, syncedAppWithSingleHistoryAnnotated)
// 	appClientMock := createTestAppClientForApp(t, app)
// 	db := createTestArgoDbForAppAndRepo(t, app, &repo)
// 	changeRevisionRequest := repoapiclient.ChangeRevisionRequest{
// 		AppName:          app.GetName(),
// 		Namespace:        app.GetNamespace(),
// 		CurrentRevision:  "c732f4d2ef24c7eeb900e9211ff98f90bb646506",
// 		PreviousRevision: "",
// 		Paths:            path.GetAppRefreshPaths(app),
// 		Repo:             &repo,
// 	}
// 	changeRevisionResponce := repoapiclient.ChangeRevisionResponse{}
// 	changeRevisionResponce.Revision = expectedRevision
// 	clientsetmock := createTestRepoclientForApp(t, &changeRevisionRequest, &changeRevisionResponce)
// 	mrpService := newTestMRPService(t, clientsetmock, appClientMock, db)
// 	err := mrpService.ChangeRevision(t.Context(), app)
// 	assert.NoError(t, err)
// }

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

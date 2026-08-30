package metrics

import (
	"context"
	"math"
	"time"

	"golang.org/x/sync/semaphore"

	"github.com/argoproj/argo-cd/v3/util/env"
	"github.com/argoproj/argo-cd/v3/util/git"
)

var (
	lsRemoteParallelismLimit          = env.ParseInt64FromEnv("ARGOCD_GIT_LS_REMOTE_PARALLELISM_LIMIT", 0, 0, math.MaxInt64)
	lsRemoteParallelismLimitSemaphore *semaphore.Weighted
)

func init() {
	if lsRemoteParallelismLimit > 0 {
		lsRemoteParallelismLimitSemaphore = semaphore.NewWeighted(lsRemoteParallelismLimit)
	}
}

// NewGitClientEventHandlers creates event handlers that update Git related metrics
func NewGitClientEventHandlers(metricsServer *MetricsServer) git.EventHandlers {
	return git.EventHandlers{
		OnFetch: func(repo string) func() {
			startTime := time.Now()
			return func() {
				metricsServer.IncGitRequest(repo, GitRequestTypeFetch, false, true)
				metricsServer.ObserveGitRequestDuration(repo, GitRequestTypeFetch, false, true, time.Since(startTime))
			}
		},
		OnLsRemote: func(repo string) func() {
			startTime := time.Now()
			if lsRemoteParallelismLimitSemaphore != nil {
				// The `Acquire` method returns either `nil` or error of the provided context. The
				// context.Background() is never canceled, so it is safe to ignore the error.
				_ = lsRemoteParallelismLimitSemaphore.Acquire(context.Background(), 1)
			}
			return func() {
				if lsRemoteParallelismLimitSemaphore != nil {
					lsRemoteParallelismLimitSemaphore.Release(1)
				}
				metricsServer.IncGitRequest(repo, GitRequestTypeLsRemote, false, true)
				metricsServer.ObserveGitRequestDuration(repo, GitRequestTypeLsRemote, false, true, time.Since(startTime))
			}
		},
		OnDiffTree: func(repo string) func(cached, success bool) {
			startTime := time.Now()
			return func(cached, success bool) {
				metricsServer.IncGitRequest(repo, GitRequestTypeDiffTree, cached, success)
				metricsServer.ObserveGitRequestDuration(repo, GitRequestTypeDiffTree, cached, success, time.Since(startTime))
			}
		},
		OnRevList: func(repo string) func(cached, success bool) {
			startTime := time.Now()
			return func(cached, success bool) {
				metricsServer.IncGitRequest(repo, GitRequestTypeRevList, cached, success)
				metricsServer.ObserveGitRequestDuration(repo, GitRequestTypeRevList, cached, success, time.Since(startTime))
			}
		},
	}
}

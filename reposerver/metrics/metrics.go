package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricsServer struct {
	handler                           http.Handler
	gitFetchFailCounter               *prometheus.CounterVec
	gitLsRemoteFailCounter            *prometheus.CounterVec
	gitDiffTreeFailCounter            *prometheus.CounterVec
	gitRevListFailCounter             *prometheus.CounterVec
	gitRequestCounter                 *prometheus.CounterVec
	gitRequestHistogram               *prometheus.HistogramVec
	getChangeRevisionRequestCounter   *prometheus.CounterVec
	getChangeRevisionRequestHistogram *prometheus.HistogramVec
	repoPendingRequestsGauge          *prometheus.GaugeVec
	redisRequestCounter               *prometheus.CounterVec
	redisRequestHistogram             *prometheus.HistogramVec
	incrementalFetchExhaustedCounter  *prometheus.CounterVec
	incrementalFetchCommitsHistogram  *prometheus.HistogramVec
	incrementalFetchBatchesHistogram  *prometheus.HistogramVec
	PrometheusRegistry                *prometheus.Registry
}

type GitRequestType string

const (
	GitRequestTypeLsRemote = "ls-remote"
	GitRequestTypeFetch    = "fetch"
	GitRequestTypeDiffTree = "diff-tree"
	GitRequestTypeRevList  = "rev-list"
)

// NewMetricsServer returns a new prometheus server which collects application metrics.
func NewMetricsServer() *MetricsServer {
	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	registry.MustRegister(collectors.NewGoCollector())

	getChangeRevisionRequestCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "monorepo_getchangerevision_request_total",
			Help: "Number of GetChangeRevision requests executed.",
		},
		[]string{"repo", "failed", "application", "namespace"},
	)
	registry.MustRegister(getChangeRevisionRequestCounter)

	getChangeRevisionRequestHistogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "monorepo_getchangerevision_request_duration_seconds",
			Help:    "GetChangeRevision requests duration seconds.",
			Buckets: []float64{0.1, 0.25, .5, 1, 2, 4, 10, 20},
		},
		[]string{"repo", "application", "namespace"},
	)
	registry.MustRegister(getChangeRevisionRequestHistogram)

	gitFetchFailCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "monorepo_git_fetch_fail_total",
			Help: "Number of git fetch requests failures by repo server",
		},
		[]string{"repo", "revision"},
	)
	registry.MustRegister(gitFetchFailCounter)

	gitLsRemoteFailCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "monorepo_git_lsremote_fail_total",
			Help: "Number of git ls-remote requests failures by repo server",
		},
		[]string{"repo", "revision"},
	)
	registry.MustRegister(gitLsRemoteFailCounter)

	gitRevListFailCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "monorepo_git_revlist_fail_total",
			Help: "Number of git rev-list requests failures by repo server",
		},
		[]string{"repo", "app", "namespace"},
	)
	registry.MustRegister(gitRevListFailCounter)

	gitDiffTreeFailCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "monorepo_git_difftree_fail_total",
			Help: "Number of git diff-tree requests failures by repo server",
		},
		[]string{"repo", "app", "namespace"},
	)
	registry.MustRegister(gitDiffTreeFailCounter)

	gitRequestCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "monorepo_git_request_total",
			Help: "Number of git requests performed by repo server",
		},
		[]string{"repo", "request_type", "cached", "success"},
	)
	registry.MustRegister(gitRequestCounter)

	gitRequestHistogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "monorepo_git_request_duration_seconds",
			Help:    "Git requests duration seconds.",
			Buckets: []float64{0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 4, 8, 16},
		},
		[]string{"repo", "request_type", "cached", "success"},
	)
	registry.MustRegister(gitRequestHistogram)

	repoPendingRequestsGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "monorepo_repo_pending_request_total",
			Help: "Number of pending requests requiring repository lock",
		},
		[]string{"repo"},
	)
	registry.MustRegister(repoPendingRequestsGauge)

	redisRequestCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "monorepo_redis_request_total",
			Help: "Number of redis requests executed.",
		},
		[]string{"initiator", "command", "failed"},
	)
	registry.MustRegister(redisRequestCounter)

	redisRequestHistogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "monorepo_redis_request_duration_seconds",
			Help:    "Redis requests duration seconds.",
			Buckets: []float64{0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 4, 8, 16},
		},
		[]string{"initiator"},
	)
	registry.MustRegister(redisRequestHistogram)

	incrementalFetchExhaustedCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "monorepo_incremental_fetch_exhausted_total",
			Help: "Number of incremental change revision searches that ran out of their depth or time budget.",
		},
		[]string{"repo", "application", "namespace", "behavior"},
	)
	registry.MustRegister(incrementalFetchExhaustedCounter)

	incrementalFetchCommitsHistogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "monorepo_incremental_fetch_commits",
			Help:    "Number of commits of the application history that were available when the incremental change revision search answered.",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 5000},
		},
		[]string{"repo", "application", "namespace"},
	)
	registry.MustRegister(incrementalFetchCommitsHistogram)

	incrementalFetchBatchesHistogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "monorepo_incremental_fetch_batches",
			Help:    "Number of history windows the incremental change revision search had to fetch.",
			Buckets: []float64{1, 2, 3, 5, 10, 20},
		},
		[]string{"repo", "application", "namespace"},
	)
	registry.MustRegister(incrementalFetchBatchesHistogram)

	return &MetricsServer{
		handler:                           promhttp.HandlerFor(registry, promhttp.HandlerOpts{}),
		gitFetchFailCounter:               gitFetchFailCounter,
		gitLsRemoteFailCounter:            gitLsRemoteFailCounter,
		gitRevListFailCounter:             gitRevListFailCounter,
		gitDiffTreeFailCounter:            gitDiffTreeFailCounter,
		getChangeRevisionRequestCounter:   getChangeRevisionRequestCounter,
		getChangeRevisionRequestHistogram: getChangeRevisionRequestHistogram,
		gitRequestCounter:                 gitRequestCounter,
		gitRequestHistogram:               gitRequestHistogram,
		repoPendingRequestsGauge:          repoPendingRequestsGauge,
		redisRequestCounter:               redisRequestCounter,
		redisRequestHistogram:             redisRequestHistogram,
		incrementalFetchExhaustedCounter:  incrementalFetchExhaustedCounter,
		incrementalFetchCommitsHistogram:  incrementalFetchCommitsHistogram,
		incrementalFetchBatchesHistogram:  incrementalFetchBatchesHistogram,
		PrometheusRegistry:                registry,
	}
}

func (m *MetricsServer) GetHandler() http.Handler {
	return m.handler
}

func (m *MetricsServer) IncGitFetchFail(repo string, revision string) {
	m.gitFetchFailCounter.WithLabelValues(repo, revision).Inc()
}

func (m *MetricsServer) IncGitLsRemoteFail(repo string, revision string) {
	m.gitLsRemoteFailCounter.WithLabelValues(repo, revision).Inc()
}

func (m *MetricsServer) IncDiffTreeFail(repo string, app string, namespace string) {
	m.gitDiffTreeFailCounter.WithLabelValues(repo, app, namespace).Inc()
}

func (m *MetricsServer) IncRevListFail(repo string, app string, namespace string) {
	m.gitRevListFailCounter.WithLabelValues(repo, app, namespace).Inc()
}

// IncGitRequest increments the git requests counter
func (m *MetricsServer) IncGitRequest(repo string, requestType GitRequestType, cached, success bool) {
	m.gitRequestCounter.WithLabelValues(repo, string(requestType), strconv.FormatBool(cached), strconv.FormatBool(success)).Inc()
}

func (m *MetricsServer) IncPendingRepoRequest(repo string) {
	m.repoPendingRequestsGauge.WithLabelValues(repo).Inc()
}

func (m *MetricsServer) ObserveGitRequestDuration(repo string, requestType GitRequestType, cached, success bool, duration time.Duration) {
	m.gitRequestHistogram.WithLabelValues(repo, string(requestType), strconv.FormatBool(cached), strconv.FormatBool(success)).Observe(duration.Seconds())
}

func (m *MetricsServer) DecPendingRepoRequest(repo string) {
	m.repoPendingRequestsGauge.WithLabelValues(repo).Dec()
}

func (m *MetricsServer) IncRedisRequest(command string, failed bool) {
	m.redisRequestCounter.WithLabelValues("argocd-monorepo-repo-server", command, strconv.FormatBool(failed)).Inc()
}

func (m *MetricsServer) ObserveRedisRequestDuration(duration time.Duration) {
	m.redisRequestHistogram.WithLabelValues("argocd-monorepo-repo-server").Observe(duration.Seconds())
}

func (m *MetricsServer) IncGetChangeRevisionRequest(repo string, failed bool, app string, namespace string) {
	m.getChangeRevisionRequestCounter.WithLabelValues(repo, strconv.FormatBool(failed), app, namespace).Inc()
}

func (m *MetricsServer) ObserveGetChangeRevisionRequestDuration(duration time.Duration, repo string, app string, namespace string) {
	m.getChangeRevisionRequestHistogram.WithLabelValues(repo, app, namespace).Observe(duration.Seconds())
}

// IncIncrementalFetchExhausted counts the incremental change revision searches
// that could not answer within their depth or time budget, by the behavior
// they fell back to.
func (m *MetricsServer) IncIncrementalFetchExhausted(repo string, app string, namespace string, behavior string) {
	m.incrementalFetchExhaustedCounter.WithLabelValues(repo, app, namespace, behavior).Inc()
}

// ObserveIncrementalFetchCommits records how much of the application history
// the incremental change revision search had available when it answered.
func (m *MetricsServer) ObserveIncrementalFetchCommits(repo string, app string, namespace string, commits int) {
	m.incrementalFetchCommitsHistogram.WithLabelValues(repo, app, namespace).Observe(float64(commits))
}

// ObserveIncrementalFetchBatches records how many history windows the
// incremental change revision search had to fetch before it answered. A value
// above one means that the initial depth is too small for this application.
func (m *MetricsServer) ObserveIncrementalFetchBatches(repo string, app string, namespace string, batches int) {
	m.incrementalFetchBatchesHistogram.WithLabelValues(repo, app, namespace).Observe(float64(batches))
}

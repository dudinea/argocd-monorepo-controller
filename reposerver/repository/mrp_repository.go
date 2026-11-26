package repository

import (
	"context"
	"fmt"
	goio "io"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/v3/reposerver/apiclient"
	argopath "github.com/argoproj/argo-cd/v3/util/app/path"
	"github.com/argoproj/argo-cd/v3/util/git"
	"github.com/argoproj/argo-cd/v3/util/io"

	log "github.com/sirupsen/logrus"
)

// Behavior of the incremental change revision search when it runs out of its
// depth or time budget before it can tell which commit changed the
// application.
const (
	// ExhaustionFullFetch fetches the complete history and calculates the
	// change revision the way a disabled incremental fetch would.
	ExhaustionFullFetch = "full-fetch"
	// ExhaustionError fails the request with a retryable error. The history
	// fetched so far stays on disk, so the next attempt resumes from the depth
	// this one reached instead of starting over.
	ExhaustionError = "error"
	// ExhaustionCurrentRevision reports the current revision as the change
	// revision, which is what the calculation does anyway when it finds no
	// path between the revisions.
	ExhaustionCurrentRevision = "current-revision"
)

// Defaults of the incremental change revision search.
const (
	DefaultIncrementalFetchInitialDepth = 50
	DefaultIncrementalFetchBatchDepth   = 200
	DefaultIncrementalFetchMaxDepth     = 5000
	DefaultIncrementalFetchBudget       = 60 * time.Second
)

// IncrementalFetchOptions configures the incremental change revision search,
// which fetches the history of a repository in portions instead of in one
// fetch of everything. A fetch of the complete history of a large monorepo
// does not finish within the timeout of a git command, and nearly all of it is
// wasted: the commit that changed an application is usually close to the tip,
// and an application whose paths nothing touches keeps the revisions it was
// annotated with earlier.
type IncrementalFetchOptions struct {
	// Enabled turns the incremental search on. It is off by default.
	Enabled bool
	// InitialDepth is the number of commits the first window holds.
	InitialDepth int
	// BatchDepth is the number of commits every deepening of the window adds.
	BatchDepth int
	// MaxDepth caps the number of commits of the history of the current
	// revision the search may examine. Not capped when 0.
	MaxDepth int
	// Budget caps the time the search may spend on fetching and walking the
	// history of one request.
	Budget time.Duration
	// OnExhaustion selects what the search does when it reaches one of the
	// limits above without an answer.
	OnExhaustion string
}

func (o IncrementalFetchOptions) Validate() error {
	if !o.Enabled {
		return nil
	}
	var errs []string
	if o.InitialDepth <= 0 {
		errs = append(errs, fmt.Sprintf("initial depth is %d, must be positive", o.InitialDepth))
	}
	if o.BatchDepth <= 0 {
		errs = append(errs, fmt.Sprintf("batch depth is %d, must be positive", o.BatchDepth))
	}
	if o.MaxDepth < 0 {
		errs = append(errs, fmt.Sprintf("max depth is %d, must not be negative", o.MaxDepth))
	}
	if o.MaxDepth > 0 && o.MaxDepth < o.InitialDepth {
		errs = append(errs, fmt.Sprintf("max depth %d is smaller than the initial depth %d", o.MaxDepth, o.InitialDepth))
	}
	if o.Budget <= 0 {
		errs = append(errs, fmt.Sprintf("budget is %v, must be positive", o.Budget))
	}
	switch o.OnExhaustion {
	case ExhaustionFullFetch, ExhaustionError, ExhaustionCurrentRevision:
	default:
		errs = append(errs, fmt.Sprintf("behavior on exhaustion is %q, must be one of %q, %q, %q",
			o.OnExhaustion, ExhaustionFullFetch, ExhaustionError, ExhaustionCurrentRevision))
	}
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("invalid incremental fetch options: %s", strings.Join(errs, "; "))
}

func (s *Service) GetChangeRevision(ctx context.Context, request *apiclient.ChangeRevisionRequest) (result *apiclient.ChangeRevisionResponse, err error) {
	startTime := time.Now()

	logCtx := log.WithFields(log.Fields{"application": request.AppName, "appNamespace": request.Namespace})
	app := request.AppName
	namespace := request.Namespace
	repo := request.GetRepo()
	currentRevision := request.GetCurrentRevision()
	previousRevision := request.GetPreviousRevision()
	refreshPaths := request.GetPaths()
	defer func() {
		s.metricsServer.ObserveGetChangeRevisionRequestDuration(time.Since(startTime),
			repo.Repo, app, namespace)
		s.metricsServer.IncGetChangeRevisionRequest(repo.Repo, err != nil, app, namespace)
	}()

	logCtx.WithFields(log.Fields{
		"repo":             repo,
		"app":              app,
		"namespace":        namespace,
		"currentRevision":  currentRevision,
		"previousRevision": previousRevision,
		"refreshPaths":     refreshPaths,
	}).Info("GetChangeRevision called")

	if repo == nil {
		return nil, status.Error(codes.InvalidArgument, "must pass a valid repo")
	}

	if len(refreshPaths) == 0 {
		return nil, status.Error(codes.InvalidArgument, "must pass a refresh path")
	}

	var gitClientOpts []git.ClientOpts
	if s.initConstants.UseCache {
		gitClientOpts = append(gitClientOpts, git.WithCache(s.cache, true))
	}
	if s.initConstants.UseListRevisionsCache {
		gitClientOpts = append(gitClientOpts, git.WithListRevisionsCache(s.cache))
	}
	if s.initConstants.UseDiffTreeCache {
		gitClientOpts = append(gitClientOpts, git.WithDiffTreeCache(s.cache))
	}
	gitClient, revision, err := s.newClientResolveRevision(repo, currentRevision, gitClientOpts...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "unable to resolve git revision %s: %v", revision, err)
	}
	if previousRevision == "" {
		logCtx.Infof("there is no previous revision (new app or source), using current revision as change revision")
		return &apiclient.ChangeRevisionResponse{
			Revision: currentRevision,
		}, nil
	}

	s.metricsServer.IncPendingRepoRequest(repo.Repo)
	defer s.metricsServer.DecPendingRepoRequest(repo.Repo)

	search := &changeRevisionSearch{
		service:   s,
		logCtx:    logCtx,
		gitClient: gitClient,
		repo:      repo,
		app:       app,
		namespace: namespace,
		target:    revision,
		previous:  previousRevision,
		paths:     refreshPaths,
		examined:  map[string]bool{},
	}
	if s.initConstants.IncrementalFetch.Enabled {
		return search.incremental(ctx)
	}
	return search.full()
}

// changeRevisionSearch is a single search for the commit that changed the
// manifests of one application source between two revisions.
type changeRevisionSearch struct {
	service   *Service
	logCtx    *log.Entry
	gitClient git.Client
	repo      *v1alpha1.Repository
	app       string
	namespace string
	// target is the current revision of the application, resolved to a SHA.
	target string
	// previous is the revision the application was synchronized to before.
	previous string
	// paths are the paths the application generates its manifests from.
	paths []string
	// examined remembers the commits whose changed files have already been
	// checked, so that a deepened window does not check them again.
	examined map[string]bool
	// depth is the depth of the window this search has fetched so far, 0
	// before its first fetch.
	depth int
}

// full calculates the change revision from the complete history of the
// repository. This is what the monorepo repo server does when the incremental
// fetch is turned off.
func (c *changeRevisionSearch) full() (*apiclient.ChangeRevisionResponse, error) {
	s := c.service
	closer, err := s.repoLock.Lock(c.gitClient.Root(), c.target, true, func() (goio.Closer, error) {
		return s.checkoutRevision(c.gitClient, c.target, false)
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "unable to checkout git repo %s with revision %s: %v", c.repo.Repo, c.target, err)
	}
	defer io.Close(closer)

	return c.answerFromRange()
}

// incremental calculates the change revision from a history that it fetches in
// portions, deepening it only as long as the answer is not in the portion that
// is already on disk.
func (c *changeRevisionSearch) incremental(ctx context.Context) (*apiclient.ChangeRevisionResponse, error) {
	s := c.service
	root := c.gitClient.Root()

	closer, err := s.repoLock.Lock(root, c.target, true, func() (goio.Closer, error) {
		return s.initRepo(c.gitClient)
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "unable to initialize git repo %s: %v", c.repo.Repo, err)
	}
	defer io.Close(closer)

	shallow, err := c.gitClient.IsShallow()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "unable to check whether git repo %s is shallow: %v", c.repo.Repo, err)
	}
	if !shallow {
		hasCommits, err := c.gitClient.HasCommits()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "unable to check the history of git repo %s: %v", c.repo.Repo, err)
		}
		if hasCommits {
			// The repository already holds the complete history. Fetching it
			// transfers only the objects that are new, while truncating it to
			// a window would hide history that is already on disk behind a
			// graft and make `git rev-list` stop there.
			c.logCtx.Info("local repository holds the complete history, calculating the change revision without a window")
			if err := c.fetchFullHistory(); err != nil {
				return nil, err
			}
			return c.answerFromRange()
		}
	}
	return c.window(ctx)
}

// window searches the history of the current revision for the change
// revision, deepening the locally available part of it in steps. The history
// it fetches stays on disk, so a later request, including a retry of this one,
// resumes from the depth this search reached.
func (c *changeRevisionSearch) window(ctx context.Context) (*apiclient.ChangeRevisionResponse, error) {
	opts := c.service.initConstants.IncrementalFetch
	deadline := time.Now().Add(opts.Budget)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}

	for batch := 0; ; batch++ {
		response, available, deeper, err := c.tryWindow()
		if err != nil {
			return nil, err
		}
		if response != nil {
			c.logCtx.Infof("the incremental search answered with %d commits available after fetching %d window(s)", available, batch)
			c.service.metricsServer.ObserveIncrementalFetchCommits(c.repo.Repo, c.app, c.namespace, available)
			c.service.metricsServer.ObserveIncrementalFetchBatches(c.repo.Repo, c.app, c.namespace, batch)
			return response, nil
		}
		switch {
		case !deeper:
			return c.exhausted(fmt.Sprintf("the end of the history of repo %s without finding the previous revision %s",
				c.repo.Repo, c.previous))
		case ctx.Err() != nil:
			return nil, status.FromContextError(ctx.Err()).Err()
		case opts.MaxDepth > 0 && available >= opts.MaxDepth:
			return c.exhausted(fmt.Sprintf("its depth limit of %d commits", opts.MaxDepth))
		case !time.Now().Before(deadline):
			return c.exhausted(fmt.Sprintf("its time budget of %v", opts.Budget))
		}
		if err := c.extendWindow(); err != nil {
			return nil, err
		}
	}
}

// tryWindow answers the search from the history that is available locally. A
// nil response means that the answer, if there is one, lies deeper: deeper
// then reports whether fetching more of the history can still help.
// available is the number of commits of the history of the current revision
// that the search could look at.
func (c *changeRevisionSearch) tryWindow() (response *apiclient.ChangeRevisionResponse, available int, deeper bool, err error) {
	boundaries, err := c.gitClient.ShallowBoundaries()
	if err != nil {
		return nil, 0, false, status.Errorf(codes.Internal, "unable to read the boundary of the fetched history of repo %s: %v", c.repo.Repo, err)
	}
	deeper = len(boundaries) > 0
	if !deeper {
		// A repository without a boundary either holds its complete history,
		// so that fetching more of it cannot help, or has not been fetched at
		// all yet.
		hasCommits, err := c.gitClient.HasCommits()
		if err != nil {
			return nil, 0, false, status.Errorf(codes.Internal, "unable to check the history of git repo %s: %v", c.repo.Repo, err)
		}
		deeper = !hasCommits
	}
	boundary := make(map[string]bool, len(boundaries))
	for _, b := range boundaries {
		boundary[b] = true
	}

	if !c.gitClient.IsRevisionPresent(c.target) {
		c.logCtx.Debugf("current revision '%s' is not available locally yet", c.target)
		return nil, 0, deeper, nil
	}

	// Once the previous revision is on disk, the range between the revisions
	// is known and the search is the same as the one without a window.
	if c.gitClient.IsRevisionPresent(c.previous) {
		revisions, err := c.listRevisions()
		if err != nil {
			return nil, 0, false, err
		}
		if crossesBoundary(revisions, boundary) {
			// The range reaches a graft, so commits of it are still missing.
			// Their changed files cannot be read, and a commit at the graft
			// reports no changed files at all.
			c.logCtx.Debug("the range between the revisions reaches the boundary of the fetched history")
			return nil, len(revisions), deeper, nil
		}
		response, err := c.answerFromRevisions(revisions)
		return response, len(revisions), deeper, err
	}

	// The previous revision is not available, so the exact range cannot be
	// listed. Walk the history of the current revision instead, newest first,
	// and answer with the first commit that changed the application.
	//
	// Note that the walk is not restricted to the commits on a path between
	// the two revisions the way `git rev-list --ancestry-path` restricts it.
	// A commit that is an ancestor of the previous revision can only be
	// reached before the previous revision itself when its committer date is
	// more recent, which takes a skewed clock or a rewritten history.
	window, err := c.gitClient.ListRevisionsFrom(c.target, 0)
	if err != nil {
		c.service.metricsServer.IncRevListFail(c.repo.Repo, c.app, c.namespace)
		return nil, 0, false, status.Errorf(codes.Internal, "failed to list the history of %s: %v", c.target, err)
	}
	revision, complete, err := c.scan(window, boundary)
	if err != nil {
		return nil, len(window), false, err
	}
	if revision != "" {
		c.logCtx.Infof("changes found in repo %s from revision %s to revision %s in revision %s",
			c.repo.Repo, c.previous, c.target, revision)
		return &apiclient.ChangeRevisionResponse{Revision: revision}, len(window), deeper, nil
	}
	if complete {
		c.logCtx.Debugf("no application change in the %d commits available, and the previous revision is not among them", len(window))
	}
	return nil, len(window), deeper, nil
}

// answerFromRange lists the revisions between the previous and the current
// revision and answers the search from them. The previous revision has to be
// available locally.
func (c *changeRevisionSearch) answerFromRange() (*apiclient.ChangeRevisionResponse, error) {
	revisions, err := c.listRevisions()
	if err != nil {
		return nil, err
	}
	return c.answerFromRevisions(revisions)
}

func (c *changeRevisionSearch) answerFromRevisions(revisions []string) (*apiclient.ChangeRevisionResponse, error) {
	if len(revisions) == 0 {
		c.logCtx.Infof("no path between revisions '%s' and '%s', using current revision as change revision", c.previous, c.target)
		return &apiclient.ChangeRevisionResponse{
			Revision: c.target,
		}, nil
	}
	revision, _, err := c.scan(revisions, nil)
	if err != nil {
		return nil, err
	}
	if revision == "" {
		c.logCtx.Infof("changes not found in repo %s from revision %s to revision %s", c.repo.Repo, c.previous, c.target)
		return &apiclient.ChangeRevisionResponse{}, nil
	}
	c.logCtx.Infof("changes found in repo %s from revision %s to revision %s in revision %s",
		c.repo.Repo, c.previous, c.target, revision)
	return &apiclient.ChangeRevisionResponse{
		Revision: revision,
	}, nil
}

func (c *changeRevisionSearch) listRevisions() ([]string, error) {
	c.logCtx.Debugf("running list revisions '%s' .. '%s'", c.previous, c.target)
	revisions, err := c.gitClient.ListRevisions(c.previous, c.target)
	if err != nil {
		c.logCtx.Errorf("failed to get revisions %s..%s: %v", c.previous, c.target, err)
		c.service.metricsServer.IncRevListFail(c.repo.Repo, c.app, c.namespace)
		return nil, status.Errorf(codes.Internal, "failed to get revisions %s..%s: %v", c.previous, c.target, err)
	}
	c.logCtx.Debugf("got list of %d revisions: %v", len(revisions), revisions)
	return revisions, nil
}

// scan examines the revisions, newest first, and returns the first one that
// changed a file the application generates its manifests from. It returns an
// empty revision when none of them did.
//
// A commit at the boundary of a shallow history cannot be examined: its
// parents are missing, so git reports no changed files for it and it would
// pass for a commit that changed nothing. scan stops at it instead of skipping
// it, because a commit further down the walk would then be reported as the
// newest change while the commit at the boundary may be a newer one.
//
// The boundary belongs to the repository as a whole and not to a single
// branch: git lists a commit in it only while its parents are really missing,
// and drops it from the list as soon as a fetch brings them, whichever branch
// they arrive with. A commit that merely sits as deep as the window of one
// branch reaches is therefore examined like any other.
func (c *changeRevisionSearch) scan(revisions []string, boundary map[string]bool) (string, bool, error) {
	for _, revision := range revisions {
		if boundary[revision] {
			c.logCtx.Debugf("revision '%s' is at the boundary of the fetched history and cannot be examined", revision)
			return "", false, nil
		}
		if c.examined[revision] {
			continue
		}
		c.logCtx.Debugf("checking for changes in revision '%s'", revision)
		files, err := c.gitClient.DiffTree(revision)
		if err != nil {
			c.service.metricsServer.IncDiffTreeFail(c.repo.Repo, c.app, c.namespace)
			c.logCtx.Errorf("Difftree for revision %s returned error: %s", revision, err.Error())
			return "", false, status.Errorf(codes.Internal, "Difftree for revision %s returned error: %s", revision, err.Error())
		}
		c.logCtx.Debugf("refreshpath is '%v'", c.paths)
		c.logCtx.Debugf("files are '%v'", files)
		c.examined[revision] = true
		if len(files) == 0 {
			continue
		}
		if argopath.AppFilesHaveChanged(c.paths, files) {
			return revision, true, nil
		}
	}
	return "", true, nil
}

// exhausted applies the configured behavior for a search that could not answer
// within its budget.
func (c *changeRevisionSearch) exhausted(reason string) (*apiclient.ChangeRevisionResponse, error) {
	opts := c.service.initConstants.IncrementalFetch
	c.logCtx.Warnf("the incremental search for the change revision of repo %s between revisions %s and %s reached %s, continuing with %q",
		c.repo.Repo, c.previous, c.target, reason, opts.OnExhaustion)
	c.service.metricsServer.IncIncrementalFetchExhausted(c.repo.Repo, c.app, c.namespace, opts.OnExhaustion)

	switch opts.OnExhaustion {
	case ExhaustionCurrentRevision:
		return &apiclient.ChangeRevisionResponse{
			Revision: c.target,
		}, nil
	case ExhaustionFullFetch:
		if err := c.fetchFullHistory(); err != nil {
			return nil, err
		}
		return c.answerFromRange()
	default:
		return nil, status.Errorf(codes.Internal,
			"the incremental search for the change revision of repo %s between revisions %s and %s reached %s",
			c.repo.Repo, c.previous, c.target, reason)
	}
}

// fetchFullHistory brings the complete history of the repository, undoing the
// truncation that an earlier incremental fetch has left behind. The work tree
// is not checked out: the calculation reads commits and trees only.
func (c *changeRevisionSearch) fetchFullHistory() error {
	s := c.service
	root := c.gitClient.Root()
	s.fetchLock.Lock(root)
	defer s.fetchLock.Unlock(root)

	shallow, err := c.gitClient.IsShallow()
	if err != nil {
		return status.Errorf(codes.Internal, "unable to check whether git repo %s is shallow: %v", c.repo.Repo, err)
	}
	start := time.Now()
	if shallow {
		// A plain fetch does not undo the truncation on its own.
		err = c.gitClient.Unshallow()
	} else {
		err = c.gitClient.Fetch("")
	}
	if err != nil {
		s.metricsServer.IncGitFetchFail(root, c.target)
		return status.Errorf(codes.Internal, "failed to fetch the complete history of repo %s: %v", c.repo.Repo, err)
	}
	c.logCtx.Infof("fetched the complete history of repo %s in %v", c.repo.Repo, time.Since(start))
	return nil
}

// extendWindow fetches more of the history of the repository.
func (c *changeRevisionSearch) extendWindow() error {
	s := c.service
	opts := s.initConstants.IncrementalFetch
	root := c.gitClient.Root()
	// Only one fetch may change the window of a local repository at a time.
	s.fetchLock.Lock(root)
	defer s.fetchLock.Unlock(root)

	// The state is read again under the lock, because a concurrent request for
	// another application of the same repository may have fetched meanwhile.
	shallow, err := c.gitClient.IsShallow()
	if err != nil {
		return status.Errorf(codes.Internal, "unable to check whether git repo %s is shallow: %v", c.repo.Repo, err)
	}

	depth := opts.InitialDepth
	if shallow {
		// The depth of a fetch is absolute, so the batch has to be added to
		// the depth the window already has. The depth of the window is also
		// measured, and not only taken from the last fetch of this search,
		// because a request for another application of the same repository
		// may have deepened the window further meanwhile, and asking for less
		// than its depth would shorten it. The greater of the two is used: git
		// counts the depth from the tip of every ref, while the measurement
		// counts from the current revision, which may sit below the tip, so
		// the measurement alone could ask for less than this search has
		// already fetched.
		current, err := c.currentDepth()
		if err != nil {
			return err
		}
		depth = max(current, c.depth) + opts.BatchDepth
	}

	start := time.Now()
	c.logCtx.Infof("fetching the history of repo %s to a depth of %d commits", c.repo.Repo, depth)
	if err := c.gitClient.FetchShallow(depth); err != nil {
		s.metricsServer.IncGitFetchFail(root, c.target)
		return status.Errorf(codes.Internal, "failed to fetch the history of repo %s: %v", c.repo.Repo, err)
	}
	c.depth = depth
	c.logCtx.Debugf("fetched the history of repo %s in %v", c.repo.Repo, time.Since(start))
	return nil
}

// currentDepth is the number of commits of the history of the current
// revision that the local repository holds, and 0 while the revision itself
// is missing.
func (c *changeRevisionSearch) currentDepth() (int, error) {
	if !c.gitClient.IsRevisionPresent(c.target) {
		return 0, nil
	}
	window, err := c.gitClient.ListRevisionsFrom(c.target, 0)
	if err != nil {
		c.service.metricsServer.IncRevListFail(c.repo.Repo, c.app, c.namespace)
		return 0, status.Errorf(codes.Internal, "failed to measure the fetched history of %s: %v", c.target, err)
	}
	return len(window), nil
}

// crossesBoundary reports whether any of the revisions sits at the boundary of
// a shallow history.
func crossesBoundary(revisions []string, boundary map[string]bool) bool {
	for _, revision := range revisions {
		if boundary[revision] {
			return true
		}
	}
	return false
}

// initRepo initializes the local repository without fetching anything or
// checking anything out. It returns the closer that takes the permissions of
// the repository directory away again.
func (s *Service) initRepo(gitClient git.Client) (goio.Closer, error) {
	closer := s.gitRepoInitializer(gitClient.Root())
	if err := gitClient.Init(); err != nil {
		return closer, status.Errorf(codes.Internal, "Failed to initialize git repo: %v", err)
	}
	return closer, nil
}

package repository

import (
	"context"
	"errors"
	"fmt"
	goio "io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/v3/reposerver/apiclient"
	"github.com/argoproj/argo-cd/v3/reposerver/metrics"
	"github.com/argoproj/argo-cd/v3/util/git"
	gitmocks "github.com/argoproj/argo-cd/v3/util/git/mocks"
	utilio "github.com/argoproj/argo-cd/v3/util/io"
	iomocks "github.com/argoproj/argo-cd/v3/util/io/mocks"
)

const testRepoURL = "https://example.com/monorepo.git"

func testSHA(i int) string {
	return fmt.Sprintf("%040x", i)
}

// fakeRepo models a linear history and how much of it has been fetched into
// the local repository, so that the incremental search can be driven without a
// real git repository. The commits are ordered newest first, as git lists them.
type fakeRepo struct {
	// commits holds the SHAs of the history, newest first.
	commits []string
	// changed holds the files every commit changed.
	changed map[string][]string
	// fetched is the number of commits available locally, counted from the
	// tip. Zero means that nothing has been fetched yet.
	fetched int
	// fetches records every fetch the search performed, in order.
	fetches []string
	// diffed counts how often the changed files of a commit were read.
	diffed map[string]int
	// rangeHook, when set, replaces the range between two revisions. It models
	// a history that is not linear, where a commit of the range can sit below
	// the boundary of the fetched history although both ends of the range are
	// available.
	rangeHook func(linear []string) []string
	// t reports the calls a correct search must never make.
	t *testing.T
}

func newFakeRepo(t *testing.T, commits int, changedAt map[int][]string) *fakeRepo {
	t.Helper()
	repo := &fakeRepo{
		changed: map[string][]string{},
		diffed:  map[string]int{},
		t:       t,
	}
	for i := 0; i < commits; i++ {
		sha := testSHA(i)
		repo.commits = append(repo.commits, sha)
		if files, ok := changedAt[i]; ok {
			repo.changed[sha] = files
		} else {
			repo.changed[sha] = []string{"other/unrelated.yaml"}
		}
	}
	return repo
}

func (r *fakeRepo) index(sha string) int {
	for i, commit := range r.commits {
		if commit == sha {
			return i
		}
	}
	return -1
}

func (r *fakeRepo) isShallow() bool {
	return r.fetched > 0 && r.fetched < len(r.commits)
}

func (r *fakeRepo) boundary() string {
	if !r.isShallow() {
		return ""
	}
	return r.commits[r.fetched-1]
}

// register wires the fake repository to the git client mock.
func (r *fakeRepo) register(client *gitmocks.Client, root string) {
	client.On("Root").Return(root)
	client.On("Init").Return(nil)
	client.On("LsRemote", mock.Anything).Return(func(revision string) (string, error) {
		return revision, nil
	})

	client.EXPECT().IsShallow().RunAndReturn(func() (bool, error) {
		return r.isShallow(), nil
	})
	client.EXPECT().HasCommits().RunAndReturn(func() (bool, error) {
		return r.fetched > 0, nil
	})
	client.EXPECT().ShallowBoundaries().RunAndReturn(func() ([]string, error) {
		if boundary := r.boundary(); boundary != "" {
			return []string{boundary}, nil
		}
		return nil, nil
	})
	client.EXPECT().IsRevisionPresent(mock.Anything).RunAndReturn(func(revision string) bool {
		idx := r.index(revision)
		return idx >= 0 && idx < r.fetched
	})

	client.EXPECT().FetchShallow(mock.Anything).RunAndReturn(func(depth int) error {
		// git counts the depth from the tip of every ref and shortens a
		// window that is deeper than it, which a search must never ask for.
		if depth < r.fetched {
			r.t.Errorf("FetchShallow(%d) on a window of %d commits would discard the fetched history", depth, r.fetched)
		}
		r.fetches = append(r.fetches, fmt.Sprintf("depth:%d", depth))
		r.fetched = min(depth, len(r.commits))
		return nil
	})
	client.EXPECT().Unshallow().RunAndReturn(func() error {
		r.fetches = append(r.fetches, "unshallow")
		r.fetched = len(r.commits)
		return nil
	})
	client.EXPECT().Fetch(mock.Anything).RunAndReturn(func(revision string) error {
		r.fetches = append(r.fetches, "fetch:"+revision)
		if !r.isShallow() {
			r.fetched = len(r.commits)
		}
		return nil
	})
	client.EXPECT().Checkout(mock.Anything, mock.Anything).RunAndReturn(func(revision string, _ bool) (string, error) {
		return revision, nil
	})

	client.EXPECT().ListRevisionsFrom(mock.Anything, mock.Anything).RunAndReturn(func(revision string, limit int) ([]string, error) {
		idx := r.index(revision)
		if idx < 0 || idx >= r.fetched {
			return nil, errors.New("unknown revision " + revision)
		}
		window := r.commits[idx:r.fetched]
		if limit > 0 && len(window) > limit {
			window = window[:limit]
		}
		return append([]string(nil), window...), nil
	})
	client.EXPECT().ListRevisions(mock.Anything, mock.Anything).RunAndReturn(func(previous string, target string) ([]string, error) {
		previousIdx, targetIdx := r.index(previous), r.index(target)
		if previousIdx < 0 || previousIdx >= r.fetched {
			return nil, errors.New("unknown revision " + previous)
		}
		if targetIdx < 0 || targetIdx >= r.fetched {
			return nil, errors.New("unknown revision " + target)
		}
		if previousIdx <= targetIdx {
			return nil, nil
		}
		revisions := append([]string(nil), r.commits[targetIdx:previousIdx]...)
		if r.rangeHook != nil {
			revisions = r.rangeHook(revisions)
		}
		return revisions, nil
	})
	client.EXPECT().DiffTree(mock.Anything).RunAndReturn(func(revision string) ([]string, error) {
		r.diffed[revision]++
		// A commit at the boundary of a shallow history has no parent
		// locally, so git reports no changed files for it.
		if revision == r.boundary() {
			return nil, nil
		}
		return r.changed[revision], nil
	})
}

func newTestService(t *testing.T, repo *fakeRepo, opts IncrementalFetchOptions) *Service {
	t.Helper()
	root := t.TempDir()
	client := &gitmocks.Client{}
	repo.register(client, root)

	paths := &iomocks.TempPaths{}
	paths.On("GetPath", mock.Anything).Return(root, nil)

	service := NewService(metrics.NewMetricsServer(), nil, RepoServerInitConstants{
		IncrementalFetch: opts,
	}, &git.NoopCredsStore{}, root)
	service.newGitClient = func(_ string, _ string, _ git.Creds, _ bool, _ bool, _ string, _ string, _ ...git.ClientOpts) (git.Client, error) {
		return client, nil
	}
	service.gitRepoInitializer = func(_ string) goio.Closer {
		return utilio.NopCloser
	}
	service.gitRepoPaths = paths
	return service
}

func incrementalOptions() IncrementalFetchOptions {
	return IncrementalFetchOptions{
		Enabled:      true,
		InitialDepth: 50,
		BatchDepth:   100,
		MaxDepth:     DefaultIncrementalFetchMaxDepth,
		Budget:       DefaultIncrementalFetchBudget,
		OnExhaustion: ExhaustionFullFetch,
	}
}

func changeRevisionRequest(current string, previous string) *apiclient.ChangeRevisionRequest {
	return &apiclient.ChangeRevisionRequest{
		AppName:          "guestbook",
		Namespace:        "argocd",
		CurrentRevision:  current,
		PreviousRevision: previous,
		Paths:            []string{"apps/guestbook"},
		Repo:             &v1alpha1.Repository{Repo: testRepoURL},
	}
}

const appFile = "apps/guestbook/deployment.yaml"

func TestGetChangeRevision_Incremental_AnswerInFirstWindow(t *testing.T) {
	repo := newFakeRepo(t, 1000, map[int][]string{2: {appFile}})
	service := newTestService(t, repo, incrementalOptions())

	// The previous revision is inside the first window, so the exact range
	// between the revisions is known straight away.
	response, err := service.GetChangeRevision(context.Background(), changeRevisionRequest(testSHA(0), testSHA(40)))
	require.NoError(t, err)
	assert.Equal(t, testSHA(2), response.Revision)
	assert.Equal(t, []string{"depth:50"}, repo.fetches)
}

func TestGetChangeRevision_Incremental_AnswerWithoutPreviousRevision(t *testing.T) {
	repo := newFakeRepo(t, 1000, map[int][]string{3: {appFile}})
	service := newTestService(t, repo, incrementalOptions())

	// The previous revision is far below the window: the search answers from
	// the commits it has without fetching the history down to it.
	response, err := service.GetChangeRevision(context.Background(), changeRevisionRequest(testSHA(0), testSHA(900)))
	require.NoError(t, err)
	assert.Equal(t, testSHA(3), response.Revision)
	assert.Equal(t, []string{"depth:50"}, repo.fetches)
}

func TestGetChangeRevision_Incremental_DeepensUntilFound(t *testing.T) {
	repo := newFakeRepo(t, 1000, map[int][]string{120: {appFile}})
	service := newTestService(t, repo, incrementalOptions())

	response, err := service.GetChangeRevision(context.Background(), changeRevisionRequest(testSHA(0), testSHA(900)))
	require.NoError(t, err)
	assert.Equal(t, testSHA(120), response.Revision)
	assert.Equal(t, []string{"depth:50", "depth:150"}, repo.fetches)

	for revision, times := range repo.diffed {
		assert.LessOrEqual(t, times, 1, "revision %s was examined %d times, a deepened window must not examine a commit again", revision, times)
	}
}

// TestGetChangeRevision_Incremental_RevisionBelowTheTip covers a current
// revision that sits further below the tip of the branch than a single
// portion, which an application that tracks a revision instead of a branch
// head leaves behind. git counts the depth of a fetch from the tip, so the
// depth has to grow past the distance to the revision for the window to reach
// it at all.
func TestGetChangeRevision_Incremental_RevisionBelowTheTip(t *testing.T) {
	repo := newFakeRepo(t, 1000, map[int][]string{400: {appFile}})
	service := newTestService(t, repo, incrementalOptions())

	response, err := service.GetChangeRevision(context.Background(), changeRevisionRequest(testSHA(150), testSHA(900)))
	require.NoError(t, err)
	assert.Equal(t, testSHA(400), response.Revision)
	assert.Equal(t, []string{"depth:50", "depth:150", "depth:250", "depth:350", "depth:450"}, repo.fetches,
		"every fetch has to ask for a greater depth than the last one, or the window stops growing")
}

// TestGetChangeRevision_Incremental_WindowOfAnotherSearch covers a window that
// a request for another application of the same repository has already
// deepened: the depth of the next fetch is measured from it instead of
// shortening it back.
func TestGetChangeRevision_Incremental_WindowOfAnotherSearch(t *testing.T) {
	repo := newFakeRepo(t, 1000, map[int][]string{450: {appFile}})
	repo.fetched = 400
	service := newTestService(t, repo, incrementalOptions())

	response, err := service.GetChangeRevision(context.Background(), changeRevisionRequest(testSHA(0), testSHA(900)))
	require.NoError(t, err)
	assert.Equal(t, testSHA(450), response.Revision)
	assert.Equal(t, []string{"depth:500"}, repo.fetches)
}

func TestGetChangeRevision_Incremental_NoRelevantChange(t *testing.T) {
	repo := newFakeRepo(t, 1000, nil)
	service := newTestService(t, repo, incrementalOptions())

	response, err := service.GetChangeRevision(context.Background(), changeRevisionRequest(testSHA(0), testSHA(10)))
	require.NoError(t, err)
	assert.Empty(t, response.Revision, "no commit between the revisions changed the application")
	assert.Equal(t, []string{"depth:50"}, repo.fetches)
}

// TestGetChangeRevision_Incremental_BoundaryCommit covers the commit at the
// boundary of the fetched history: git reports no changed files for it, so the
// search must fetch more history instead of taking it for a commit that
// changed nothing.
func TestGetChangeRevision_Incremental_BoundaryCommit(t *testing.T) {
	repo := newFakeRepo(t, 1000, map[int][]string{49: {appFile}})
	service := newTestService(t, repo, incrementalOptions())

	response, err := service.GetChangeRevision(context.Background(), changeRevisionRequest(testSHA(0), testSHA(900)))
	require.NoError(t, err)
	assert.Equal(t, testSHA(49), response.Revision)
	assert.Equal(t, []string{"depth:50", "depth:150"}, repo.fetches)
}

// TestGetChangeRevision_Incremental_RangeCrossingBoundary covers a range whose
// both ends are available while a commit of it still sits below the boundary
// of the fetched history, which a merge of an older branch can cause. Such a
// range cannot be trusted: the commits below the boundary are missing, and the
// one at it reports no changed files at all.
func TestGetChangeRevision_Incremental_RangeCrossingBoundary(t *testing.T) {
	repo := newFakeRepo(t, 1000, map[int][]string{30: {appFile}})
	repo.rangeHook = func(linear []string) []string {
		if repo.fetched <= 50 {
			return append(linear, repo.boundary())
		}
		return linear
	}
	service := newTestService(t, repo, incrementalOptions())

	response, err := service.GetChangeRevision(context.Background(), changeRevisionRequest(testSHA(0), testSHA(40)))
	require.NoError(t, err)
	assert.Equal(t, testSHA(30), response.Revision)
	assert.Equal(t, []string{"depth:50", "depth:150"}, repo.fetches,
		"the range reaching the boundary has to be fetched further before it is examined")
}

func TestGetChangeRevision_Incremental_CompleteHistoryIsNotTruncated(t *testing.T) {
	repo := newFakeRepo(t, 100, map[int][]string{5: {appFile}})
	// The repository already holds the complete history, as it would after a
	// run with the incremental fetch turned off.
	repo.fetched = len(repo.commits)
	service := newTestService(t, repo, incrementalOptions())

	response, err := service.GetChangeRevision(context.Background(), changeRevisionRequest(testSHA(0), testSHA(40)))
	require.NoError(t, err)
	assert.Equal(t, testSHA(5), response.Revision)
	assert.Equal(t, []string{"fetch:"}, repo.fetches, "a complete history is fetched plainly, never truncated to a window")
}

func TestGetChangeRevision_Incremental_Exhausted(t *testing.T) {
	tests := []struct {
		name         string
		onExhaustion string
		expected     string
		expectError  bool
	}{
		{name: "full fetch", onExhaustion: ExhaustionFullFetch, expected: testSHA(600)},
		{name: "current revision", onExhaustion: ExhaustionCurrentRevision, expected: testSHA(0)},
		{name: "error", onExhaustion: ExhaustionError, expectError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := newFakeRepo(t, 1000, map[int][]string{600: {appFile}})
			opts := incrementalOptions()
			opts.MaxDepth = 150
			opts.OnExhaustion = test.onExhaustion
			service := newTestService(t, repo, opts)

			response, err := service.GetChangeRevision(context.Background(), changeRevisionRequest(testSHA(0), testSHA(900)))
			if test.expectError {
				require.Error(t, err)
				assert.Equal(t, codes.Internal, status.Code(err), "the error has to be retryable")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.expected, response.Revision)
		})
	}
}

func TestGetChangeRevision_Incremental_ExhaustedFullFetchUnshallows(t *testing.T) {
	repo := newFakeRepo(t, 1000, map[int][]string{600: {appFile}})
	opts := incrementalOptions()
	opts.MaxDepth = 150
	service := newTestService(t, repo, opts)

	_, err := service.GetChangeRevision(context.Background(), changeRevisionRequest(testSHA(0), testSHA(900)))
	require.NoError(t, err)
	assert.Equal(t, []string{"depth:50", "depth:150", "unshallow"}, repo.fetches,
		"the fall back to the complete history has to undo the truncation, a plain fetch would not")
}

// TestGetChangeRevision_Incremental_PreviousRevisionNotInHistory covers a
// previous revision that is not part of the history at all, which a force push
// leaves behind. The search answers from the commits it walked, and stops
// instead of fetching the same complete history over and over.
func TestGetChangeRevision_Incremental_PreviousRevisionNotInHistory(t *testing.T) {
	t.Run("with a change of the application", func(t *testing.T) {
		repo := newFakeRepo(t, 20, map[int][]string{15: {appFile}})
		opts := incrementalOptions()
		opts.OnExhaustion = ExhaustionError
		service := newTestService(t, repo, opts)

		response, err := service.GetChangeRevision(context.Background(), changeRevisionRequest(testSHA(0), testSHA(500)))
		require.NoError(t, err)
		assert.Equal(t, testSHA(15), response.Revision)
		assert.Equal(t, []string{"depth:50"}, repo.fetches)
	})

	t.Run("without a change of the application", func(t *testing.T) {
		repo := newFakeRepo(t, 20, nil)
		opts := incrementalOptions()
		opts.OnExhaustion = ExhaustionError
		service := newTestService(t, repo, opts)

		_, err := service.GetChangeRevision(context.Background(), changeRevisionRequest(testSHA(0), testSHA(500)))
		require.Error(t, err)
		assert.Equal(t, codes.Internal, status.Code(err))
		assert.Equal(t, []string{"depth:50"}, repo.fetches, "the complete history is fetched once, not again and again")
	})
}

func TestGetChangeRevision_Incremental_ContextCancelled(t *testing.T) {
	repo := newFakeRepo(t, 1000, map[int][]string{600: {appFile}})
	service := newTestService(t, repo, incrementalOptions())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := service.GetChangeRevision(ctx, changeRevisionRequest(testSHA(0), testSHA(900)))
	require.Error(t, err)
	assert.Equal(t, codes.Canceled, status.Code(err))
}

func TestGetChangeRevision_Incremental_NoPreviousRevision(t *testing.T) {
	repo := newFakeRepo(t, 100, nil)
	service := newTestService(t, repo, incrementalOptions())

	response, err := service.GetChangeRevision(context.Background(), changeRevisionRequest(testSHA(0), ""))
	require.NoError(t, err)
	assert.Equal(t, testSHA(0), response.Revision)
	assert.Empty(t, repo.fetches, "a source without a previous revision needs no history at all")
}

// TestGetChangeRevision_Full_RestoresTruncatedHistory covers turning the
// incremental fetch off again: a plain fetch does not undo the truncation an
// earlier run left behind, and the walk of the revisions would stop at it.
func TestGetChangeRevision_Full_RestoresTruncatedHistory(t *testing.T) {
	repo := newFakeRepo(t, 1000, map[int][]string{700: {appFile}})
	repo.fetched = 50
	service := newTestService(t, repo, IncrementalFetchOptions{})

	response, err := service.GetChangeRevision(context.Background(), changeRevisionRequest(testSHA(0), testSHA(900)))
	require.NoError(t, err)
	assert.Equal(t, testSHA(700), response.Revision)
	assert.Equal(t, "unshallow", repo.fetches[0])
}

func TestGetChangeRevision_Full_UntouchedRepository(t *testing.T) {
	repo := newFakeRepo(t, 100, map[int][]string{7: {appFile}})
	service := newTestService(t, repo, IncrementalFetchOptions{})

	response, err := service.GetChangeRevision(context.Background(), changeRevisionRequest(testSHA(0), testSHA(40)))
	require.NoError(t, err)
	assert.Equal(t, testSHA(7), response.Revision)
	assert.NotContains(t, repo.fetches, "unshallow", "a repository that was never truncated is fetched as before")
}

func TestIncrementalFetchOptions_Validate(t *testing.T) {
	valid := IncrementalFetchOptions{
		Enabled:      true,
		InitialDepth: 50,
		BatchDepth:   100,
		MaxDepth:     1000,
		Budget:       time.Minute,
		OnExhaustion: ExhaustionFullFetch,
	}
	require.NoError(t, valid.Validate())
	// The options are not checked while the feature is off.
	require.NoError(t, IncrementalFetchOptions{}.Validate())

	invalid := map[string]func(o *IncrementalFetchOptions){
		"initial depth is 0, must be positive":     func(o *IncrementalFetchOptions) { o.InitialDepth = 0 },
		"batch depth is 0, must be positive":       func(o *IncrementalFetchOptions) { o.BatchDepth = 0 },
		"max depth is -1, must not be negative":    func(o *IncrementalFetchOptions) { o.MaxDepth = -1 },
		"max depth 10 is smaller than the initial": func(o *IncrementalFetchOptions) { o.MaxDepth = 10 },
		"budget is 0s, must be positive":           func(o *IncrementalFetchOptions) { o.Budget = 0 },
		`behavior on exhaustion is "nonsense"`:     func(o *IncrementalFetchOptions) { o.OnExhaustion = "nonsense" },
	}
	for expected, invalidate := range invalid {
		t.Run(expected, func(t *testing.T) {
			opts := valid
			invalidate(&opts)
			require.ErrorContains(t, opts.Validate(), expected)
		})
	}
}

// runGit runs a git command in the given directory, failing the test when it
// does not succeed.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v in %s: %s", args, dir, out)
	return strings.TrimSpace(string(out))
}

// createGitRepo creates a repository with the given number of commits. The
// commits whose distance from the tip is in appCommits change a file of the
// application, the others change a file nothing cares about. It returns the
// repository and the SHAs of its commits, the tip first.
func createGitRepo(t *testing.T, commits int, appCommits map[int]bool) (string, []string) {
	t.Helper()
	origin := t.TempDir()
	runGit(t, origin, "init", "--initial-branch=master")
	runGit(t, origin, "config", "user.email", "test@example.com")
	runGit(t, origin, "config", "user.name", "test")

	for i := commits - 1; i >= 0; i-- {
		path := "other/notes.md"
		if appCommits[i] {
			path = "apps/guestbook/deployment.yaml"
		}
		file := filepath.Join(origin, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(file), 0o755))
		require.NoError(t, os.WriteFile(file, []byte(fmt.Sprintf("commit %d\n", i)), 0o644))
		runGit(t, origin, "add", "-A")
		runGit(t, origin, "commit", "-m", fmt.Sprintf("commit at distance %d", i))
	}

	shas := make([]string, commits)
	for i := 0; i < commits; i++ {
		shas[i] = runGit(t, origin, "rev-parse", fmt.Sprintf("HEAD~%d", i))
	}
	return origin, shas
}

// newRealGitService returns a service that works on real git repositories.
func newRealGitService(t *testing.T, opts IncrementalFetchOptions) *Service {
	t.Helper()
	root := t.TempDir()
	restoreTempDirPermissions(t, root)
	return NewService(metrics.NewMetricsServer(), nil, RepoServerInitConstants{
		IncrementalFetch: opts,
	}, &git.NoopCredsStore{}, root)
}

// restoreTempDirPermissions gives the permissions of the repository
// directories back once the test is over. The service takes them away after
// every request, which would stop the test framework from removing them.
func restoreTempDirPermissions(t *testing.T, root string) {
	t.Helper()
	t.Cleanup(func() {
		_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, _ error) error {
			if entry != nil && entry.IsDir() {
				_ = os.Chmod(path, 0o700)
			}
			return nil
		})
	})
}

func realGitRequest(origin string, current string, previous string) *apiclient.ChangeRevisionRequest {
	request := changeRevisionRequest(current, previous)
	request.Repo = &v1alpha1.Repository{Repo: "file://" + origin}
	return request
}

// TestGetChangeRevision_RealRepo_IncrementalMatchesFullFetch runs the search
// on a real repository, with the incremental fetch on and off, and compares
// the answers. The repository the incremental search works on has to stay
// truncated: the point of the feature is that the complete history is never
// fetched.
func TestGetChangeRevision_RealRepo_IncrementalMatchesFullFetch(t *testing.T) {
	origin, shas := createGitRepo(t, 60, map[int]bool{3: true, 40: true})

	tests := []struct {
		name     string
		current  int
		previous int
		expected int
	}{
		{name: "change close to the tip", current: 0, previous: 50, expected: 3},
		{name: "change below the first window", current: 10, previous: 50, expected: 40},
		{name: "no change of the application", current: 10, previous: 35, expected: -1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			opts := IncrementalFetchOptions{
				Enabled:      true,
				InitialDepth: 10,
				BatchDepth:   10,
				MaxDepth:     DefaultIncrementalFetchMaxDepth,
				Budget:       DefaultIncrementalFetchBudget,
				OnExhaustion: ExhaustionError,
			}
			incremental := newRealGitService(t, opts)
			request := realGitRequest(origin, shas[test.current], shas[test.previous])
			response, err := incremental.GetChangeRevision(context.Background(), request)
			require.NoError(t, err)

			full := newRealGitService(t, IncrementalFetchOptions{})
			expected, err := full.GetChangeRevision(context.Background(), realGitRequest(origin, shas[test.current], shas[test.previous]))
			require.NoError(t, err)

			if test.expected < 0 {
				assert.Empty(t, response.Revision)
			} else {
				assert.Equal(t, shas[test.expected], response.Revision)
			}
			assert.Equal(t, expected.Revision, response.Revision,
				"the incremental search has to answer what the search of the complete history answers")

			// The local repository of the incremental search holds a part of
			// the history, not all of it.
			assert.True(t, localRepoIsShallow(t, incremental, request.Repo.Repo),
				"the complete history must not have been fetched")
		})
	}
}

// TestGetChangeRevision_RealRepo_UnshallowsWhenDisabled covers a repository
// that an incremental search truncated and that is then used with the feature
// turned off again.
func TestGetChangeRevision_RealRepo_UnshallowsWhenDisabled(t *testing.T) {
	origin, shas := createGitRepo(t, 40, map[int]bool{35: true})
	root := t.TempDir()

	opts := IncrementalFetchOptions{
		Enabled:      true,
		InitialDepth: 5,
		BatchDepth:   5,
		MaxDepth:     10,
		Budget:       DefaultIncrementalFetchBudget,
		OnExhaustion: ExhaustionError,
	}
	restoreTempDirPermissions(t, root)
	service := NewService(metrics.NewMetricsServer(), nil, RepoServerInitConstants{IncrementalFetch: opts}, &git.NoopCredsStore{}, root)
	request := realGitRequest(origin, shas[0], shas[38])
	_, err := service.GetChangeRevision(context.Background(), request)
	require.Error(t, err, "the search has to run out of its depth limit")

	require.True(t, localRepoIsShallow(t, service, request.Repo.Repo))

	// The same local repository, now used with the feature turned off.
	service.initConstants.IncrementalFetch = IncrementalFetchOptions{}
	response, err := service.GetChangeRevision(context.Background(), realGitRequest(origin, shas[0], shas[38]))
	require.NoError(t, err)
	assert.Equal(t, shas[35], response.Revision)
	assert.False(t, localRepoIsShallow(t, service, request.Repo.Repo),
		"the truncated history has to be restored")
}

// localRepoIsShallow reports whether the local copy the service keeps of the
// repository holds a truncated history. The service takes the permissions of
// that directory away after every request, so they are restored first.
func localRepoIsShallow(t *testing.T, service *Service, repoURL string) bool {
	t.Helper()
	path, err := service.gitRepoPaths.GetPath(git.NormalizeGitURL(repoURL))
	require.NoError(t, err)
	require.NoError(t, os.Chmod(path, 0o700))
	return runGit(t, path, "rev-parse", "--is-shallow-repository") == "true"
}

// TestGetChangeRevision_RealRepo_Concurrent covers several applications of one
// repository whose searches deepen the same local history at the same time.
// git refuses two fetches that change the window of a repository at once, so
// the fetches have to be serialized.
func TestGetChangeRevision_RealRepo_Concurrent(t *testing.T) {
	origin, shas := createGitRepo(t, 60, map[int]bool{3: true, 40: true})
	service := newRealGitService(t, IncrementalFetchOptions{
		Enabled:      true,
		InitialDepth: 2,
		BatchDepth:   2,
		MaxDepth:     DefaultIncrementalFetchMaxDepth,
		Budget:       DefaultIncrementalFetchBudget,
		OnExhaustion: ExhaustionError,
	})

	previous := []int{20, 30, 50, 55}
	responses := make([]*apiclient.ChangeRevisionResponse, len(previous))
	errs := make([]error, len(previous))
	var wg sync.WaitGroup
	for i, distance := range previous {
		wg.Add(1)
		go func() {
			defer wg.Done()
			responses[i], errs[i] = service.GetChangeRevision(context.Background(),
				realGitRequest(origin, shas[0], shas[distance]))
		}()
	}
	wg.Wait()

	for i := range previous {
		require.NoError(t, errs[i])
		assert.Equal(t, shas[3], responses[i].Revision)
	}
}

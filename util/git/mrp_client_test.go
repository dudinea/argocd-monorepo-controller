package git

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestRepo creates a repository with the given number of commits, each
// of which changes a file in one of three application directories.
func createTestRepo(t *testing.T, commits int) string {
	t.Helper()
	origin := t.TempDir()
	require.NoError(t, runCmd(origin, "git", "init", "--initial-branch=master"))
	require.NoError(t, runCmd(origin, "git", "config", "user.email", "test@example.com"))
	require.NoError(t, runCmd(origin, "git", "config", "user.name", "test"))
	for i := 1; i <= commits; i++ {
		dir := filepath.Join(origin, "app"+strconv.Itoa(i%3))
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "f.txt"), []byte(strings.Repeat("x", i)), 0o644))
		require.NoError(t, runCmd(origin, "git", "add", "-A"))
		require.NoError(t, runCmd(origin, "git", "commit", "-m", "commit "+strconv.Itoa(i)))
	}
	return origin
}

// revisionAt returns the SHA of the commit the given number of commits below
// the tip of the repository.
func revisionAt(t *testing.T, repo string, below int) string {
	t.Helper()
	out, err := outputCmd(repo, "git", "rev-parse", "HEAD~"+strconv.Itoa(below))
	require.NoError(t, err)
	return strings.TrimSpace(string(out))
}

func newTestClient(t *testing.T, origin string) Client {
	t.Helper()
	client, err := NewClientExt("file://"+origin, t.TempDir(), NopCreds{}, true, false, "", "")
	require.NoError(t, err)
	require.NoError(t, client.Init())
	return client
}

func Test_nativeGitClient_HasCommits(t *testing.T) {
	origin := createTestRepo(t, 3)
	client := newTestClient(t, origin)

	hasCommits, err := client.HasCommits()
	require.NoError(t, err)
	assert.False(t, hasCommits, "a freshly initialized repository holds no commit")

	require.NoError(t, client.FetchShallow(1))

	hasCommits, err = client.HasCommits()
	require.NoError(t, err)
	assert.True(t, hasCommits)
}

func Test_nativeGitClient_FetchShallow(t *testing.T) {
	origin := createTestRepo(t, 9)
	client := newTestClient(t, origin)

	shallow, err := client.IsShallow()
	require.NoError(t, err)
	assert.False(t, shallow, "a freshly initialized repository is not shallow")

	boundaries, err := client.ShallowBoundaries()
	require.NoError(t, err)
	assert.Empty(t, boundaries, "a repository without a shallow file has no boundary")

	require.NoError(t, client.FetchShallow(3))

	shallow, err = client.IsShallow()
	require.NoError(t, err)
	assert.True(t, shallow)

	boundaries, err = client.ShallowBoundaries()
	require.NoError(t, err)
	require.Len(t, boundaries, 1)

	target := revisionAt(t, origin, 0)
	window, err := client.ListRevisionsFrom(target, 0)
	require.NoError(t, err)
	assert.Len(t, window, 3, "only the fetched portion of the history is available")
	assert.Equal(t, target, window[0], "the walk starts at the revision it is given")
	assert.Equal(t, boundaries[0], window[len(window)-1], "the oldest available commit is the boundary")
}

func Test_nativeGitClient_FetchShallow_InvalidDepth(t *testing.T) {
	origin := createTestRepo(t, 2)
	client := newTestClient(t, origin)

	require.ErrorContains(t, client.FetchShallow(0), "must be positive")
	require.ErrorContains(t, client.FetchShallow(-1), "must be positive")
}

// Test_nativeGitClient_FetchShallow_Deepens covers extending a window with a
// greater depth, the way the incremental search deepens the history it walks.
func Test_nativeGitClient_FetchShallow_Deepens(t *testing.T) {
	origin := createTestRepo(t, 9)
	client := newTestClient(t, origin)
	target := revisionAt(t, origin, 0)

	require.NoError(t, client.FetchShallow(3))
	require.NoError(t, client.FetchShallow(7))

	window, err := client.ListRevisionsFrom(target, 0)
	require.NoError(t, err)
	assert.Len(t, window, 7, "a greater depth adds commits to the window instead of replacing it")

	shallow, err := client.IsShallow()
	require.NoError(t, err)
	assert.True(t, shallow)

	// A depth that reaches past the root of the history completes the
	// repository.
	require.NoError(t, client.FetchShallow(100))
	shallow, err = client.IsShallow()
	require.NoError(t, err)
	assert.False(t, shallow)

	// Fetching a complete repository again is a no-op rather than an error.
	require.NoError(t, client.FetchShallow(100))
	window, err = client.ListRevisionsFrom(target, 0)
	require.NoError(t, err)
	assert.Len(t, window, 9)
	shallow, err = client.IsShallow()
	require.NoError(t, err)
	assert.False(t, shallow)
}

// Test_nativeGitClient_FetchShallow_BoundsANewBranch pins the reason the
// window is extended with an absolute depth instead of git's --deepen: a ref
// that was created since the last fetch has no boundary of its own, and
// --deepen leaves its history untruncated, so that it arrives complete.
func Test_nativeGitClient_FetchShallow_BoundsANewBranch(t *testing.T) {
	origin := createTestRepo(t, 12)
	client := newTestClient(t, origin)

	require.NoError(t, client.FetchShallow(3))

	// A branch that forks below the boundary of the window, as a release
	// branch cut from an old commit does.
	root, err := outputCmd(origin, "git", "rev-list", "--max-parents=0", "HEAD")
	require.NoError(t, err)
	require.NoError(t, runCmd(origin, "git", "checkout", "-b", "old-fork", strings.TrimSpace(string(root))))
	for i := 1; i <= 15; i++ {
		require.NoError(t, os.WriteFile(filepath.Join(origin, "fork.txt"), []byte(strconv.Itoa(i)), 0o644))
		require.NoError(t, runCmd(origin, "git", "add", "-A"))
		require.NoError(t, runCmd(origin, "git", "commit", "-m", "fork "+strconv.Itoa(i)))
	}
	forkTip := revisionAt(t, origin, 0)
	require.NoError(t, runCmd(origin, "git", "checkout", "master"))

	require.NoError(t, client.FetchShallow(5))

	window, err := client.ListRevisionsFrom(forkTip, 0)
	require.NoError(t, err)
	assert.Len(t, window, 5, "the history of the new branch has to be bounded by the depth of the fetch")
}

func Test_nativeGitClient_Unshallow(t *testing.T) {
	origin := createTestRepo(t, 9)
	client := newTestClient(t, origin)
	target := revisionAt(t, origin, 0)

	require.NoError(t, client.FetchShallow(2))
	require.NoError(t, client.Unshallow())

	shallow, err := client.IsShallow()
	require.NoError(t, err)
	assert.False(t, shallow)

	boundaries, err := client.ShallowBoundaries()
	require.NoError(t, err)
	assert.Empty(t, boundaries)

	window, err := client.ListRevisionsFrom(target, 0)
	require.NoError(t, err)
	assert.Len(t, window, 9)
}

// Test_nativeGitClient_FetchShallow_Tags covers a revision that only a tag
// points at: the incremental fetch keeps fetching tags, so an application
// whose target revision is a tag is still found.
func Test_nativeGitClient_FetchShallow_Tags(t *testing.T) {
	origin := createTestRepo(t, 9)
	tagged := revisionAt(t, origin, 7)
	require.NoError(t, runCmd(origin, "git", "tag", "-a", "v1", "-m", "release", tagged))
	// The tagged commit is far below the tip, so no window of this size would
	// reach it through the branch.
	client := newTestClient(t, origin)
	require.NoError(t, client.FetchShallow(2))

	assert.True(t, client.IsRevisionPresent(tagged), "the commit of the tag is fetched with its own window")
}

// Test_nativeGitClient_DiffTree_ShallowBoundary pins the reason why a commit
// at the boundary of a shallow history must never be examined: its parents are
// missing, so git reports no changed files for it even though it changed one.
func Test_nativeGitClient_DiffTree_ShallowBoundary(t *testing.T) {
	origin := createTestRepo(t, 9)
	client := newTestClient(t, origin)
	target := revisionAt(t, origin, 0)

	require.NoError(t, client.FetchShallow(3))

	boundaries, err := client.ShallowBoundaries()
	require.NoError(t, err)
	require.Len(t, boundaries, 1)

	files, err := client.DiffTree(boundaries[0])
	require.NoError(t, err)
	assert.Empty(t, files, "the boundary commit reports no changed files although it changed one")

	// The same commit reports its changed files once its parent is available.
	require.NoError(t, client.FetchShallow(6))
	files, err = client.DiffTree(boundaries[0])
	require.NoError(t, err)
	assert.NotEmpty(t, files)

	// A commit above the boundary is examined correctly from the start.
	window, err := client.ListRevisionsFrom(target, 1)
	require.NoError(t, err)
	require.Len(t, window, 1)
	files, err = client.DiffTree(window[0])
	require.NoError(t, err)
	assert.NotEmpty(t, files)
}

func Test_nativeGitClient_ListRevisionsFrom(t *testing.T) {
	origin := createTestRepo(t, 9)
	client := newTestClient(t, origin)
	target := revisionAt(t, origin, 0)
	require.NoError(t, client.FetchShallow(5))

	t.Run("limit", func(t *testing.T) {
		window, err := client.ListRevisionsFrom(target, 2)
		require.NoError(t, err)
		assert.Len(t, window, 2)
	})

	t.Run("no limit", func(t *testing.T) {
		window, err := client.ListRevisionsFrom(target, 0)
		require.NoError(t, err)
		assert.Len(t, window, 5)
	})

	t.Run("not a SHA", func(t *testing.T) {
		_, err := client.ListRevisionsFrom("master", 0)
		require.ErrorContains(t, err, "must be SHA")
	})

	t.Run("unknown revision", func(t *testing.T) {
		_, err := client.ListRevisionsFrom("0000000000000000000000000000000000000000", 0)
		require.Error(t, err)
	})
}

// Test_nativeGitClient_ShallowBoundaries_SharedHistory pins the assumption the
// incremental change revision search rests on: the boundary of a shallow
// history is a property of the whole repository, not of a single branch. A
// commit that lies at the end of the window of one branch, but whose parents
// were fetched through another branch, is not a boundary and its changed files
// can be read.
func Test_nativeGitClient_ShallowBoundaries_SharedHistory(t *testing.T) {
	origin := t.TempDir()
	require.NoError(t, runCmd(origin, "git", "init", "--initial-branch=main"))
	require.NoError(t, runCmd(origin, "git", "config", "user.email", "test@example.com"))
	require.NoError(t, runCmd(origin, "git", "config", "user.name", "test"))
	for i := 1; i <= 10; i++ {
		require.NoError(t, os.WriteFile(filepath.Join(origin, "main.txt"), []byte(strconv.Itoa(i)), 0o644))
		require.NoError(t, runCmd(origin, "git", "add", "-A"))
		require.NoError(t, runCmd(origin, "git", "commit", "-m", "C"+strconv.Itoa(i)))
	}
	// A branch that forked five commits below the tip, so that both branches
	// share the older part of the history.
	require.NoError(t, runCmd(origin, "git", "checkout", "-b", "feature", "main~5"))
	for i := 1; i <= 3; i++ {
		require.NoError(t, os.WriteFile(filepath.Join(origin, "feature.txt"), []byte(strconv.Itoa(i)), 0o644))
		require.NoError(t, runCmd(origin, "git", "add", "-A"))
		require.NoError(t, runCmd(origin, "git", "commit", "-m", "F"+strconv.Itoa(i)))
	}
	require.NoError(t, runCmd(origin, "git", "checkout", "main"))

	client := newTestClient(t, origin)
	// Eight commits of every branch: the window of main ends at its eighth
	// commit, while the window of the branch reaches the root of the history
	// through the commits the two share.
	require.NoError(t, client.FetchShallow(8))

	boundaries, err := client.ShallowBoundaries()
	require.NoError(t, err)
	require.NotEmpty(t, boundaries, "the history is truncated, so it has a boundary")

	// Every boundary is a commit whose parents are really missing.
	for _, boundary := range boundaries {
		out, err := outputCmd(client.Root(), "git", "rev-parse", boundary+"^@")
		if err != nil {
			continue // a root commit has no parent to look for
		}
		for parent := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
			if parent = strings.TrimSpace(parent); parent != "" {
				assert.False(t, client.IsRevisionPresent(parent),
					"commit %s is a boundary although its parent %s is available", boundary, parent)
			}
		}
	}

	// The eighth commit of main is as deep as its own window reaches, and its
	// parents came with the other branch, so it is examinable like any other.
	deepOnMain := revisionAt(t, origin, 7)
	assert.NotContains(t, boundaries, deepOnMain,
		"a commit whose parents arrived through another branch must not be a boundary")
	files, err := client.DiffTree(deepOnMain)
	require.NoError(t, err)
	assert.Equal(t, []string{"main.txt"}, files)
}

package git

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// WithListRevisionsCache sets list revisions cacher
func WithListRevisionsCache(cache listRevisionsCache) ClientOpts {
	return func(c *nativeGitClient) {
		c.listRevisionsCache = cache
		c.useListRevisionsCache = true
	}
}

// WithListRevisionsCache sets list revisions cacher
func WithDiffTreeCache(cache diffTreeCache) ClientOpts {
	return func(c *nativeGitClient) {
		c.diffTreeCache = cache
		c.useDiffTreeCache = true
	}
}

type listRevisionsCache interface {
	GetListRevisions(repoURL, previousRevision, targetRevision string) ([]string, error)
	SetListRevisions(repoURL, previousRevision, targetRevision string, revisions []string) error
}

type diffTreeCache interface {
	GetDiffTree(repoURL, revision string) ([]string, error)
	SetDiffTree(repoURL, revision string, files []string) error
}

func (m *nativeGitClient) ListRevisions(revision string, targetRevision string) ([]string, error) {
	cached := false
	var err error
	if revision == "" {
		return []string{targetRevision}, nil
	}

	if !IsCommitSHA(revision) || !IsCommitSHA(targetRevision) {
		err = errors.New("invalid revision provided, must be SHA")
		return nil, err
	}

	if m.OnRevList != nil {
		done := m.OnRevList(m.repoURL)
		defer func() {
			done(cached, err == nil)
		}()
	}
	if revision == targetRevision {
		return []string{revision}, nil
	}

	var revisions []string
	// Try to get from cache first
	if m.useListRevisionsCache && m.listRevisionsCache != nil {
		if revisions, err = m.listRevisionsCache.GetListRevisions(m.repoURL, revision, targetRevision); err == nil {
			cached = true
			return revisions, nil
		}
	}

	var out string
	out, err = m.runCmd("rev-list", "--ancestry-path", fmt.Sprintf("%s..%s", revision, targetRevision))
	if err != nil {
		return nil, err
	}

	if out == "" {
		revisions = []string{}
	} else {
		revisions = strings.Split(out, "\n")
	}

	// Cache successful results with non-zero length
	if err == nil && len(revisions) > 0 && m.useListRevisionsCache && m.listRevisionsCache != nil {
		_ = m.listRevisionsCache.SetListRevisions(m.repoURL, revision, targetRevision, revisions)
	}

	return revisions, nil
}

func (m *nativeGitClient) DiffTree(targetRevision string) ([]string, error) {
	cached := false
	var err error

	if !IsCommitSHA(targetRevision) {
		return []string{}, errors.New("invalid revision provided, must be SHA")
	}
	if m.OnDiffTree != nil {
		done := m.OnDiffTree(m.repoURL)
		defer func() {
			done(cached, err == nil)
		}()
	}

	var result []string
	if m.useDiffTreeCache && m.diffTreeCache != nil {
		if result, err = m.diffTreeCache.GetDiffTree(m.repoURL, targetRevision); err == nil {
			cached = true
			return result, nil
		}
	}

	var out string
	out, err = m.runCmd("diff-tree", "--no-commit-id", "--name-only", "-r", targetRevision)
	if err != nil {
		return nil, fmt.Errorf("failed to diff %s: %w", targetRevision, err)
	}

	if out == "" {
		return []string{}, nil
	}

	files := strings.Split(out, "\n")

	// Cache successful results with non-zero length
	if err == nil && len(files) > 0 && m.useDiffTreeCache && m.diffTreeCache != nil {
		_ = m.diffTreeCache.SetDiffTree(m.repoURL, targetRevision, files)
	}
	return files, nil
}

// IsShallow reports whether the local repository holds a truncated history.
func (m *nativeGitClient) IsShallow() (bool, error) {
	out, err := m.runCmd("rev-parse", "--is-shallow-repository")
	if err != nil {
		return false, fmt.Errorf("failed to check whether the repository is shallow: %w", err)
	}
	return strings.TrimSpace(out) == "true", nil
}

// HasCommits reports whether the local repository holds any commit. It tells a
// freshly initialized repository, which is not shallow only because it has no
// history at all, from one that has already been fetched.
func (m *nativeGitClient) HasCommits() (bool, error) {
	out, err := m.runCmd("rev-list", "-n", "1", "--all")
	if err != nil {
		return false, fmt.Errorf("failed to list the commits of the repository: %w", err)
	}
	return strings.TrimSpace(out) != "", nil
}

// ShallowBoundaries returns the grafted commits at the bottom of a shallow
// history. Their parents are not available locally, so `git diff-tree` reports
// no changed files for them, and they must never be taken for commits that
// changed nothing.
//
// a .shallow file entry is just a line with commit SHA
func (m *nativeGitClient) ShallowBoundaries() ([]string, error) {
	out, err := m.runCmd("rev-parse", "--git-path", "shallow")
	if err != nil {
		return nil, fmt.Errorf("failed to resolve the path of the shallow file: %w", err)
	}
	path := strings.TrimSpace(out)
	if !filepath.IsAbs(path) {
		path = filepath.Join(m.root, path)
	}
	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		// A repository that holds the full history has no shallow file.
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read the shallow file %s: %w", path, err)
	}
	var boundaries []string
	for line := range strings.SplitSeq(string(content), "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			boundaries = append(boundaries, trimmed)
		}
	}
	return boundaries, nil
}

// FetchShallow fetches the same refs as Fetch(""), truncating the history of
// every one of them to depth commits, counted from the tip of the ref. Tags
// are fetched as well, so that a revision only a tag points at becomes
// reachable as soon as the window is deep enough. A depth that reaches past
// the root of the history restores the repository to its complete state.
//
// Depth is absolute, and git shortens a window that is deeper than it,
// discarding history that earlier calls have paid for, so a window is
// extended by calling this again with a greater depth. git also offers
// --deepen to extend a window by a relative number of commits, but it leaves
// the history of a ref that has no boundary yet, a ref that was created since
// the last fetch, untruncated: such a ref arrives with its complete history,
// which is the very cost this is here to avoid.
func (m *nativeGitClient) FetchShallow(depth int) error {
	if depth <= 0 {
		return fmt.Errorf("fetch depth is %d, must be positive", depth)
	}
	return m.fetchWindow(fmt.Sprintf("--depth=%d", depth))
}

// Unshallow restores the full history of a repository that an incremental
// fetch has truncated. A plain fetch does not undo the truncation on its own.
func (m *nativeGitClient) Unshallow() error {
	return m.fetchWindow("--unshallow")
}

// fetchWindow runs the fetch of Fetch("") with an extra history argument. LFS
// objects are deliberately left out: the incremental change revision search
// reads commits and trees only, and never checks the work tree out.
func (m *nativeGitClient) fetchWindow(historyArg string) error {
	if m.OnFetch != nil {
		done := m.OnFetch(m.repoURL)
		defer done()
	}
	return m.runCredentialedCmd("fetch", "origin", historyArg, "--tags", "--force", "--prune")
}

// ListRevisionsFrom lists the commits of the history of revision, newest
// first, at most limit of them when limit is positive. In a shallow repository
// the walk stops at the boundary of the fetched window, so the result is the
// part of the history that is available locally.
//
// The result is deliberately not cached: unlike the range of ListRevisions, it
// depends on how deep the local history happens to be.
func (m *nativeGitClient) ListRevisionsFrom(revision string, limit int) ([]string, error) {
	cached := false
	var err error

	if !IsCommitSHA(revision) {
		return nil, errors.New("invalid revision provided, must be SHA")
	}
	if m.OnRevList != nil {
		done := m.OnRevList(m.repoURL)
		defer func() {
			done(cached, err == nil)
		}()
	}

	args := []string{"rev-list", "--date-order"}
	if limit > 0 {
		args = append(args, "-n", strconv.Itoa(limit))
	}
	args = append(args, revision)

	var out string
	out, err = m.runCmd(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list the revisions of %s: %w", revision, err)
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

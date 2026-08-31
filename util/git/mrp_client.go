package git

import (
	"errors"
	"fmt"
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

package repository

// import (
// 	"bytes"
// 	"context"
// 	"encoding/json"
// 	"errors"
// 	"fmt"
// 	goio "io"
// 	"io/fs"
// 	"net/mail"
// 	"os"
// 	"os/exec"
// 	"path"
// 	"path/filepath"
// 	"regexp"
// 	"slices"
// 	"sort"
// 	"strings"
// 	"sync"
// 	"testing"
// 	"time"

// 	log "github.com/sirupsen/logrus"
// 	"k8s.io/apimachinery/pkg/api/resource"
// 	"k8s.io/apimachinery/pkg/util/intstr"

// 	"github.com/argoproj/argo-cd/v3/util/oci"

// 	cacheutil "github.com/argoproj/argo-cd/v3/util/cache"

// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/mock"
// 	"github.com/stretchr/testify/require"
// 	appsv1 "k8s.io/api/apps/v1"
// 	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
// 	"k8s.io/apimachinery/pkg/runtime"
// 	"sigs.k8s.io/yaml"

// 	"github.com/argoproj/argo-cd/v3/common"
// 	"github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
// 	"github.com/argoproj/argo-cd/v3/reposerver/apiclient"
// 	"github.com/argoproj/argo-cd/v3/reposerver/cache"
// 	repositorymocks "github.com/argoproj/argo-cd/v3/reposerver/cache/mocks"
// 	"github.com/argoproj/argo-cd/v3/reposerver/metrics"
// 	fileutil "github.com/argoproj/argo-cd/v3/test/fixture/path"
// 	//"github.com/argoproj/argo-cd/v3/util/argo"
// 	"github.com/argoproj/argo-cd/v3/util/git"
// 	gitmocks "github.com/argoproj/argo-cd/v3/util/git/mocks"
// 	"github.com/argoproj/argo-cd/v3/util/helm"
// 	helmmocks "github.com/argoproj/argo-cd/v3/util/helm/mocks"
// 	utilio "github.com/argoproj/argo-cd/v3/util/io"
// 	iomocks "github.com/argoproj/argo-cd/v3/util/io/mocks"
// 	ocimocks "github.com/argoproj/argo-cd/v3/util/oci/mocks"
// )

// const testSignature = `gpg: Signature made Wed Feb 26 23:22:34 2020 CET
// gpg:                using RSA key 4AEE18F83AFDEB23
// gpg: Good signature from "GitHub (web-flow commit signing) <noreply@github.com>" [ultimate]
// `

// type clientFunc func(*gitmocks.Client, *helmmocks.Client, *ocimocks.Client, *iomocks.TempPaths)

// type repoCacheMocks struct {
// 	mock.Mock
// 	cacheutilCache *cacheutil.Cache
// 	cache          *cache.Cache
// 	mockCache      *repositorymocks.MockRepoCache
// }

// type newGitRepoHelmChartOptions struct {
// 	chartName string
// 	// valuesFiles is a map of the values file name to the key/value pairs to be written to the file
// 	valuesFiles map[string]map[string]string
// }

// type newGitRepoOptions struct {
// 	path             string
// 	createPath       bool
// 	remote           string
// 	addEmptyCommit   bool
// 	helmChartOptions newGitRepoHelmChartOptions
// }

// func newCacheMocks() *repoCacheMocks {
// 	return newCacheMocksWithOpts(1*time.Minute, 1*time.Minute, 10*time.Second)
// }

// func newCacheMocksWithOpts(repoCacheExpiration, revisionCacheExpiration, revisionCacheLockTimeout time.Duration) *repoCacheMocks {
// 	mockRepoCache := repositorymocks.NewMockRepoCache(&repositorymocks.MockCacheOptions{
// 		RepoCacheExpiration:     1 * time.Minute,
// 		RevisionCacheExpiration: 1 * time.Minute,
// 		ReadDelay:               0,
// 		WriteDelay:              0,
// 	})
// 	cacheutilCache := cacheutil.NewCache(mockRepoCache.RedisClient)
// 	return &repoCacheMocks{
// 		cacheutilCache: cacheutilCache,
// 		cache:          cache.NewCache(cacheutilCache, repoCacheExpiration, revisionCacheExpiration, revisionCacheLockTimeout),
// 		mockCache:      mockRepoCache,
// 	}
// }

// func newServiceWithMocks(t *testing.T, root string, signed bool) (*Service, *gitmocks.Client, *repoCacheMocks) {
// 	t.Helper()
// 	root, err := filepath.Abs(root)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return newServiceWithOpt(t, func(gitClient *gitmocks.Client, helmClient *helmmocks.Client, ociClient *ocimocks.Client, paths *iomocks.TempPaths) {
// 		gitClient.On("Init").Return(nil)
// 		gitClient.On("IsRevisionPresent", mock.Anything).Return(false)
// 		gitClient.On("Fetch", mock.Anything).Return(nil)
// 		gitClient.On("Checkout", mock.Anything, mock.Anything).Return("", nil)
// 		gitClient.On("LsRemote", mock.Anything).Return(mock.Anything, nil)
// 		gitClient.On("CommitSHA").Return(mock.Anything, nil)
// 		gitClient.On("Root").Return(root)
// 		gitClient.On("IsAnnotatedTag").Return(false)
// 		if signed {
// 			gitClient.On("VerifyCommitSignature", mock.Anything).Return(testSignature, nil)
// 		} else {
// 			gitClient.On("VerifyCommitSignature", mock.Anything).Return("", nil)
// 		}

// 		chart := "my-chart"
// 		oobChart := "out-of-bounds-chart"
// 		version := "1.1.0"
// 		helmClient.On("GetIndex", mock.AnythingOfType("bool"), mock.Anything).Return(&helm.Index{Entries: map[string]helm.Entries{
// 			chart:    {{Version: "1.0.0"}, {Version: version}},
// 			oobChart: {{Version: "1.0.0"}, {Version: version}},
// 		}}, nil)
// 		helmClient.On("GetTags", mock.Anything, mock.Anything).Return(nil, nil)
// 		helmClient.On("ExtractChart", chart, version, false, int64(0), false).Return("./testdata/my-chart", utilio.NopCloser, nil)
// 		helmClient.On("ExtractChart", oobChart, version, false, int64(0), false).Return("./testdata2/out-of-bounds-chart", utilio.NopCloser, nil)
// 		helmClient.On("CleanChartCache", chart, version).Return(nil)
// 		helmClient.On("CleanChartCache", oobChart, version).Return(nil)
// 		helmClient.On("DependencyBuild").Return(nil)

// 		ociClient.On("GetTags", mock.Anything, mock.Anything).Return(nil)
// 		ociClient.On("ResolveRevision", mock.Anything, mock.Anything, mock.Anything).Return("", nil)
// 		ociClient.On("Extract", mock.Anything, mock.Anything).Return("./testdata/my-chart", utilio.NopCloser, nil)

// 		paths.On("Add", mock.Anything, mock.Anything).Return(root, nil)
// 		paths.On("GetPath", mock.Anything).Return(root, nil)
// 		paths.On("GetPathIfExists", mock.Anything).Return(root, nil)
// 		paths.On("GetPaths").Return(map[string]string{"fake-nonce": root})
// 	}, root)
// }

// func newServiceWithOpt(t *testing.T, cf clientFunc, root string) (*Service, *gitmocks.Client, *repoCacheMocks) {
// 	t.Helper()
// 	helmClient := &helmmocks.Client{}
// 	gitClient := &gitmocks.Client{}
// 	ociClient := &ocimocks.Client{}
// 	paths := &iomocks.TempPaths{}
// 	cf(gitClient, helmClient, ociClient, paths)
// 	cacheMocks := newCacheMocks()
// 	t.Cleanup(cacheMocks.mockCache.StopRedisCallback)
// 	service := NewService(metrics.NewMetricsServer(), cacheMocks.cache, RepoServerInitConstants{ParallelismLimit: 1}, &git.NoopCredsStore{}, root)

// 	service.newGitClient = func(_ string, _ string, _ git.Creds, _ bool, _ bool, _ string, _ string, _ ...git.ClientOpts) (client git.Client, e error) {
// 		return gitClient, nil
// 	}
// 	service.newHelmClient = func(_ string, _ helm.Creds, _ bool, _ string, _ string, _ ...helm.ClientOpts) helm.Client {
// 		return helmClient
// 	}
// 	service.newOCIClient = func(_ string, _ oci.Creds, _ string, _ string, _ []string, _ ...oci.ClientOpts) (oci.Client, error) {
// 		return ociClient, nil
// 	}
// 	service.gitRepoInitializer = func(_ string) goio.Closer {
// 		return utilio.NopCloser
// 	}
// 	service.gitRepoPaths = paths
// 	return service, gitClient, cacheMocks
// }

// func newService(t *testing.T, root string) *Service {
// 	t.Helper()
// 	service, _, _ := newServiceWithMocks(t, root, false)
// 	return service
// }

// func newServiceWithSignature(t *testing.T, root string) *Service {
// 	t.Helper()
// 	service, _, _ := newServiceWithMocks(t, root, true)
// 	return service
// }

// func newServiceWithCommitSHA(t *testing.T, root, revision string) *Service {
// 	t.Helper()
// 	var revisionErr error

// 	commitSHARegex := regexp.MustCompile("^[0-9A-Fa-f]{40}$")
// 	if !commitSHARegex.MatchString(revision) {
// 		revisionErr = errors.New("not a commit SHA")
// 	}

// 	service, gitClient, _ := newServiceWithOpt(t, func(gitClient *gitmocks.Client, _ *helmmocks.Client, _ *ocimocks.Client, paths *iomocks.TempPaths) {
// 		gitClient.On("Init").Return(nil)
// 		gitClient.On("IsRevisionPresent", mock.Anything).Return(false)
// 		gitClient.On("Fetch", mock.Anything).Return(nil)
// 		gitClient.On("Checkout", mock.Anything, mock.Anything).Return("", nil)
// 		gitClient.On("LsRemote", revision).Return(revision, revisionErr)
// 		gitClient.On("CommitSHA").Return("632039659e542ed7de0c170a4fcc1c571b288fc0", nil)
// 		gitClient.On("Root").Return(root)
// 		paths.On("GetPath", mock.Anything).Return(root, nil)
// 		paths.On("GetPathIfExists", mock.Anything).Return(root, nil)
// 	}, root)

// 	service.newGitClient = func(_ string, _ string, _ git.Creds, _ bool, _ bool, _ string, _ string, _ ...git.ClientOpts) (client git.Client, e error) {
// 		return gitClient, nil
// 	}

// 	return service
// }

// func TestIdentifyAppSourceTypeByAppDirWithKustomizations(t *testing.T) {
// 	sourceType, err := GetAppSourceType(t.Context(), &v1alpha1.ApplicationSource{}, "./testdata/kustomization_yaml", "./testdata", "testapp", map[string]bool{}, []string{}, []string{})
// 	require.NoError(t, err)
// 	assert.Equal(t, v1alpha1.ApplicationSourceTypeKustomize, sourceType)

// 	sourceType, err = GetAppSourceType(t.Context(), &v1alpha1.ApplicationSource{}, "./testdata/kustomization_yml", "./testdata", "testapp", map[string]bool{}, []string{}, []string{})
// 	require.NoError(t, err)
// 	assert.Equal(t, v1alpha1.ApplicationSourceTypeKustomize, sourceType)

// 	sourceType, err = GetAppSourceType(t.Context(), &v1alpha1.ApplicationSource{}, "./testdata/Kustomization", "./testdata", "testapp", map[string]bool{}, []string{}, []string{})
// 	require.NoError(t, err)
// 	assert.Equal(t, v1alpha1.ApplicationSourceTypeKustomize, sourceType)
// }

// func TestGenerateFromUTF16(t *testing.T) {
// 	q := apiclient.ManifestRequest{
// 		Repo:               &v1alpha1.Repository{},
// 		ApplicationSource:  &v1alpha1.ApplicationSource{},
// 		ProjectName:        "something",
// 		ProjectSourceRepos: []string{"*"},
// 	}
// 	res1, err := GenerateManifests(t.Context(), "./testdata/utf-16", "/", "", &q, false, &git.NoopCredsStore{}, resource.MustParse("0"), nil)
// 	require.NoError(t, err)
// 	assert.Len(t, res1.Manifests, 2)
// }

// func Test_newEnv(t *testing.T) {
// 	assert.Equal(t, &v1alpha1.Env{
// 		&v1alpha1.EnvEntry{Name: "ARGOCD_APP_NAME", Value: "my-app-name"},
// 		&v1alpha1.EnvEntry{Name: "ARGOCD_APP_NAMESPACE", Value: "my-namespace"},
// 		&v1alpha1.EnvEntry{Name: "ARGOCD_APP_PROJECT_NAME", Value: "my-project-name"},
// 		&v1alpha1.EnvEntry{Name: "ARGOCD_APP_REVISION", Value: "my-revision"},
// 		&v1alpha1.EnvEntry{Name: "ARGOCD_APP_REVISION_SHORT", Value: "my-revi"},
// 		&v1alpha1.EnvEntry{Name: "ARGOCD_APP_REVISION_SHORT_8", Value: "my-revis"},
// 		&v1alpha1.EnvEntry{Name: "ARGOCD_APP_SOURCE_REPO_URL", Value: "https://github.com/my-org/my-repo"},
// 		&v1alpha1.EnvEntry{Name: "ARGOCD_APP_SOURCE_PATH", Value: "my-path"},
// 		&v1alpha1.EnvEntry{Name: "ARGOCD_APP_SOURCE_TARGET_REVISION", Value: "my-target-revision"},
// 	}, newEnv(&apiclient.ManifestRequest{
// 		AppName:     "my-app-name",
// 		Namespace:   "my-namespace",
// 		ProjectName: "my-project-name",
// 		Repo:        &v1alpha1.Repository{Repo: "https://github.com/my-org/my-repo"},
// 		ApplicationSource: &v1alpha1.ApplicationSource{
// 			Path:           "my-path",
// 			TargetRevision: "my-target-revision",
// 		},
// 	}, "my-revision"))
// }

// func TestService_newHelmClientResolveRevision(t *testing.T) {
// 	service := newService(t, ".")

// 	t.Run("EmptyRevision", func(t *testing.T) {
// 		_, _, err := service.newHelmClientResolveRevision(&v1alpha1.Repository{}, "", "my-chart", true)
// 		assert.EqualError(t, err, "invalid revision: failed to determine semver constraint: improper constraint: ")
// 	})
// 	t.Run("InvalidRevision", func(t *testing.T) {
// 		_, _, err := service.newHelmClientResolveRevision(&v1alpha1.Repository{}, "???", "my-chart", true)
// 		assert.EqualError(t, err, "invalid revision: failed to determine semver constraint: improper constraint: ???")
// 	})
// }

// // There are unit test that will use kustomize set and by that modify the
// // kustomization.yaml. For proper testing, we need to copy the testdata to a
// // temporary path, run the tests, and then throw the copy away again.
// func mkTempParameters(source string) string {
// 	tempDir, err := os.MkdirTemp("./testdata", "app-parameters")
// 	if err != nil {
// 		panic(err)
// 	}
// 	cmd := exec.Command("cp", "-R", source, tempDir)
// 	err = cmd.Run()
// 	if err != nil {
// 		os.RemoveAll(tempDir)
// 		panic(err)
// 	}
// 	return tempDir
// }

// // Simple wrapper run a test with a temporary copy of the testdata, because
// // the test would modify the data when run.
// func runWithTempTestdata(t *testing.T, path string, runner func(t *testing.T, path string)) {
// 	t.Helper()
// 	tempDir := mkTempParameters("./testdata/app-parameters")
// 	runner(t, filepath.Join(tempDir, "app-parameters", path))
// 	os.RemoveAll(tempDir)
// }

// func TestFindManifests_Exclude(t *testing.T) {
// 	objs, err := findManifests(&log.Entry{}, "testdata/app-include-exclude", ".", nil, v1alpha1.ApplicationSourceDirectory{
// 		Recurse: true,
// 		Exclude: "subdir/deploymentSub.yaml",
// 	}, map[string]bool{}, resource.MustParse("0"))

// 	require.NoError(t, err)
// 	require.Len(t, objs, 1)

// 	assert.Equal(t, "nginx-deployment", objs[0].GetName())
// }

// func TestFindManifests_Exclude_NothingMatches(t *testing.T) {
// 	objs, err := findManifests(&log.Entry{}, "testdata/app-include-exclude", ".", nil, v1alpha1.ApplicationSourceDirectory{
// 		Recurse: true,
// 		Exclude: "nothing.yaml",
// 	}, map[string]bool{}, resource.MustParse("0"))

// 	require.NoError(t, err)
// 	require.Len(t, objs, 2)

// 	assert.ElementsMatch(t,
// 		[]string{"nginx-deployment", "nginx-deployment-sub"}, []string{objs[0].GetName(), objs[1].GetName()})
// }

// func tempDir(t *testing.T) string {
// 	t.Helper()
// 	dir, err := os.MkdirTemp(".", "")
// 	require.NoError(t, err)
// 	t.Cleanup(func() {
// 		err = os.RemoveAll(dir)
// 		if err != nil {
// 			panic(err)
// 		}
// 	})
// 	absDir, err := filepath.Abs(dir)
// 	require.NoError(t, err)
// 	return absDir
// }

// func walkFor(t *testing.T, root string, testPath string, run func(info fs.FileInfo)) {
// 	t.Helper()
// 	hitExpectedPath := false
// 	err := filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
// 		if path == testPath {
// 			require.NoError(t, err)
// 			hitExpectedPath = true
// 			run(info)
// 		}
// 		return nil
// 	})
// 	require.NoError(t, err)
// 	assert.True(t, hitExpectedPath, "did not hit expected path when walking directory")
// }

// func Test_getPotentiallyValidManifestFile(t *testing.T) {
// 	// These tests use filepath.Walk instead of os.Stat to get file info, because FileInfo from os.Stat does not return
// 	// true for IsSymlink like os.Walk does.

// 	// These tests do not use t.TempDir() because those directories can contain symlinks which cause test to fail
// 	// InBound checks.

// 	t.Run("non-JSON/YAML is skipped with an empty ignore message", func(t *testing.T) {
// 		appDir := tempDir(t)
// 		filePath := filepath.Join(appDir, "not-json-or-yaml")
// 		file, err := os.OpenFile(filePath, os.O_RDONLY|os.O_CREATE, 0o644)
// 		require.NoError(t, err)
// 		err = file.Close()
// 		require.NoError(t, err)

// 		walkFor(t, appDir, filePath, func(info fs.FileInfo) {
// 			realFileInfo, ignoreMessage, err := getPotentiallyValidManifestFile(filePath, info, appDir, appDir, "", "")
// 			assert.Nil(t, realFileInfo)
// 			assert.Empty(t, ignoreMessage)
// 			require.NoError(t, err)
// 		})
// 	})

// 	t.Run("circular link should throw an error", func(t *testing.T) {
// 		appDir := tempDir(t)

// 		aPath := filepath.Join(appDir, "a.json")
// 		bPath := filepath.Join(appDir, "b.json")
// 		err := os.Symlink(bPath, aPath)
// 		require.NoError(t, err)
// 		err = os.Symlink(aPath, bPath)
// 		require.NoError(t, err)

// 		walkFor(t, appDir, aPath, func(info fs.FileInfo) {
// 			realFileInfo, ignoreMessage, err := getPotentiallyValidManifestFile(aPath, info, appDir, appDir, "", "")
// 			assert.Nil(t, realFileInfo)
// 			assert.Empty(t, ignoreMessage)
// 			assert.ErrorContains(t, err, "too many links")
// 		})
// 	})

// 	t.Run("symlink with missing destination should throw an error", func(t *testing.T) {
// 		appDir := tempDir(t)

// 		aPath := filepath.Join(appDir, "a.json")
// 		bPath := filepath.Join(appDir, "b.json")
// 		err := os.Symlink(bPath, aPath)
// 		require.NoError(t, err)

// 		walkFor(t, appDir, aPath, func(info fs.FileInfo) {
// 			realFileInfo, ignoreMessage, err := getPotentiallyValidManifestFile(aPath, info, appDir, appDir, "", "")
// 			assert.Nil(t, realFileInfo)
// 			assert.NotEmpty(t, ignoreMessage)
// 			require.NoError(t, err)
// 		})
// 	})

// 	t.Run("out-of-bounds symlink should throw an error", func(t *testing.T) {
// 		appDir := tempDir(t)

// 		linkPath := filepath.Join(appDir, "a.json")
// 		err := os.Symlink("..", linkPath)
// 		require.NoError(t, err)

// 		walkFor(t, appDir, linkPath, func(info fs.FileInfo) {
// 			realFileInfo, ignoreMessage, err := getPotentiallyValidManifestFile(linkPath, info, appDir, appDir, "", "")
// 			assert.Nil(t, realFileInfo)
// 			assert.Empty(t, ignoreMessage)
// 			assert.ErrorContains(t, err, "illegal filepath in symlink")
// 		})
// 	})

// 	t.Run("symlink to a non-regular file should be skipped with warning", func(t *testing.T) {
// 		appDir := tempDir(t)

// 		dirPath := filepath.Join(appDir, "test.dir")
// 		err := os.MkdirAll(dirPath, 0o644)
// 		require.NoError(t, err)
// 		linkPath := filepath.Join(appDir, "test.json")
// 		err = os.Symlink(dirPath, linkPath)
// 		require.NoError(t, err)

// 		walkFor(t, appDir, linkPath, func(info fs.FileInfo) {
// 			realFileInfo, ignoreMessage, err := getPotentiallyValidManifestFile(linkPath, info, appDir, appDir, "", "")
// 			assert.Nil(t, realFileInfo)
// 			assert.Contains(t, ignoreMessage, "non-regular file")
// 			require.NoError(t, err)
// 		})
// 	})

// 	t.Run("non-included file should be skipped with no message", func(t *testing.T) {
// 		appDir := tempDir(t)

// 		filePath := filepath.Join(appDir, "not-included.yaml")
// 		file, err := os.OpenFile(filePath, os.O_RDONLY|os.O_CREATE, 0o644)
// 		require.NoError(t, err)
// 		err = file.Close()
// 		require.NoError(t, err)

// 		walkFor(t, appDir, filePath, func(info fs.FileInfo) {
// 			realFileInfo, ignoreMessage, err := getPotentiallyValidManifestFile(filePath, info, appDir, appDir, "*.json", "")
// 			assert.Nil(t, realFileInfo)
// 			assert.Empty(t, ignoreMessage)
// 			require.NoError(t, err)
// 		})
// 	})

// 	t.Run("excluded file should be skipped with no message", func(t *testing.T) {
// 		appDir := tempDir(t)

// 		filePath := filepath.Join(appDir, "excluded.json")
// 		file, err := os.OpenFile(filePath, os.O_RDONLY|os.O_CREATE, 0o644)
// 		require.NoError(t, err)
// 		err = file.Close()
// 		require.NoError(t, err)

// 		walkFor(t, appDir, filePath, func(info fs.FileInfo) {
// 			realFileInfo, ignoreMessage, err := getPotentiallyValidManifestFile(filePath, info, appDir, appDir, "", "excluded.*")
// 			assert.Nil(t, realFileInfo)
// 			assert.Empty(t, ignoreMessage)
// 			require.NoError(t, err)
// 		})
// 	})

// 	t.Run("symlink to a regular file is potentially valid", func(t *testing.T) {
// 		appDir := tempDir(t)

// 		filePath := filepath.Join(appDir, "regular-file")
// 		file, err := os.OpenFile(filePath, os.O_RDONLY|os.O_CREATE, 0o644)
// 		require.NoError(t, err)
// 		err = file.Close()
// 		require.NoError(t, err)

// 		linkPath := filepath.Join(appDir, "link.json")
// 		err = os.Symlink(filePath, linkPath)
// 		require.NoError(t, err)

// 		walkFor(t, appDir, linkPath, func(info fs.FileInfo) {
// 			realFileInfo, ignoreMessage, err := getPotentiallyValidManifestFile(linkPath, info, appDir, appDir, "", "")
// 			assert.NotNil(t, realFileInfo)
// 			assert.Empty(t, ignoreMessage)
// 			require.NoError(t, err)
// 		})
// 	})

// 	t.Run("a regular file is potentially valid", func(t *testing.T) {
// 		appDir := tempDir(t)

// 		filePath := filepath.Join(appDir, "regular-file.json")
// 		file, err := os.OpenFile(filePath, os.O_RDONLY|os.O_CREATE, 0o644)
// 		require.NoError(t, err)
// 		err = file.Close()
// 		require.NoError(t, err)

// 		walkFor(t, appDir, filePath, func(info fs.FileInfo) {
// 			realFileInfo, ignoreMessage, err := getPotentiallyValidManifestFile(filePath, info, appDir, appDir, "", "")
// 			assert.NotNil(t, realFileInfo)
// 			assert.Empty(t, ignoreMessage)
// 			require.NoError(t, err)
// 		})
// 	})

// 	t.Run("realFileInfo is for the destination rather than the symlink", func(t *testing.T) {
// 		appDir := tempDir(t)

// 		filePath := filepath.Join(appDir, "regular-file")
// 		file, err := os.OpenFile(filePath, os.O_RDONLY|os.O_CREATE, 0o644)
// 		require.NoError(t, err)
// 		err = file.Close()
// 		require.NoError(t, err)

// 		linkPath := filepath.Join(appDir, "link.json")
// 		err = os.Symlink(filePath, linkPath)
// 		require.NoError(t, err)

// 		walkFor(t, appDir, linkPath, func(info fs.FileInfo) {
// 			realFileInfo, ignoreMessage, err := getPotentiallyValidManifestFile(linkPath, info, appDir, appDir, "", "")
// 			assert.NotNil(t, realFileInfo)
// 			assert.Equal(t, filepath.Base(filePath), realFileInfo.Name())
// 			assert.Empty(t, ignoreMessage)
// 			require.NoError(t, err)
// 		})
// 	})
// }

// func Test_getPotentiallyValidManifests(t *testing.T) {
// 	// Tests which return no manifests and an error check to make sure the directory exists before running. A missing
// 	// directory would produce those same results.

// 	logCtx := log.WithField("test", "test")

// 	t.Run("unreadable file throws error", func(t *testing.T) {
// 		appDir := t.TempDir()
// 		unreadablePath := filepath.Join(appDir, "unreadable.json")
// 		err := os.WriteFile(unreadablePath, []byte{}, 0o666)
// 		require.NoError(t, err)
// 		err = os.Chmod(appDir, 0o000)
// 		require.NoError(t, err)

// 		manifests, err := getPotentiallyValidManifests(logCtx, appDir, appDir, false, "", "", resource.MustParse("0"))
// 		assert.Empty(t, manifests)
// 		require.Error(t, err)

// 		// allow cleanup
// 		err = os.Chmod(appDir, 0o777)
// 		if err != nil {
// 			panic(err)
// 		}
// 	})

// 	t.Run("no recursion when recursion is disabled", func(t *testing.T) {
// 		manifests, err := getPotentiallyValidManifests(logCtx, "./testdata/recurse", "./testdata/recurse", false, "", "", resource.MustParse("0"))
// 		assert.Len(t, manifests, 1)
// 		require.NoError(t, err)
// 	})

// 	t.Run("recursion when recursion is enabled", func(t *testing.T) {
// 		manifests, err := getPotentiallyValidManifests(logCtx, "./testdata/recurse", "./testdata/recurse", true, "", "", resource.MustParse("0"))
// 		assert.Len(t, manifests, 2)
// 		require.NoError(t, err)
// 	})

// 	t.Run("non-JSON/YAML is skipped", func(t *testing.T) {
// 		manifests, err := getPotentiallyValidManifests(logCtx, "./testdata/non-manifest-file", "./testdata/non-manifest-file", false, "", "", resource.MustParse("0"))
// 		assert.Empty(t, manifests)
// 		require.NoError(t, err)
// 	})

// 	t.Run("circular link should throw an error", func(t *testing.T) {
// 		const testDir = "./testdata/circular-link"
// 		require.DirExists(t, testDir)
// 		t.Cleanup(func() {
// 			os.Remove(path.Join(testDir, "a.json"))
// 			os.Remove(path.Join(testDir, "b.json"))
// 		})
// 		t.Chdir(testDir)
// 		require.NoError(t, fileutil.CreateSymlink(t, "a.json", "b.json"))
// 		require.NoError(t, fileutil.CreateSymlink(t, "b.json", "a.json"))
// 		manifests, err := getPotentiallyValidManifests(logCtx, "./testdata/circular-link", "./testdata/circular-link", false, "", "", resource.MustParse("0"))
// 		assert.Empty(t, manifests)
// 		require.Error(t, err)
// 	})

// 	t.Run("out-of-bounds symlink should throw an error", func(t *testing.T) {
// 		require.DirExists(t, "./testdata/out-of-bounds-link")
// 		manifests, err := getPotentiallyValidManifests(logCtx, "./testdata/out-of-bounds-link", "./testdata/out-of-bounds-link", false, "", "", resource.MustParse("0"))
// 		assert.Empty(t, manifests)
// 		require.Error(t, err)
// 	})

// 	t.Run("symlink to a regular file works", func(t *testing.T) {
// 		repoRoot, err := filepath.Abs("./testdata/in-bounds-link")
// 		require.NoError(t, err)
// 		appPath, err := filepath.Abs("./testdata/in-bounds-link/app")
// 		require.NoError(t, err)
// 		manifests, err := getPotentiallyValidManifests(logCtx, appPath, repoRoot, false, "", "", resource.MustParse("0"))
// 		assert.Len(t, manifests, 1)
// 		require.NoError(t, err)
// 	})

// 	t.Run("symlink to nowhere should be ignored", func(t *testing.T) {
// 		manifests, err := getPotentiallyValidManifests(logCtx, "./testdata/link-to-nowhere", "./testdata/link-to-nowhere", false, "", "", resource.MustParse("0"))
// 		assert.Empty(t, manifests)
// 		require.NoError(t, err)
// 	})

// 	t.Run("link to over-sized manifest fails", func(t *testing.T) {
// 		repoRoot, err := filepath.Abs("./testdata/in-bounds-link")
// 		require.NoError(t, err)
// 		appPath, err := filepath.Abs("./testdata/in-bounds-link/app")
// 		require.NoError(t, err)
// 		// The file is 35 bytes.
// 		manifests, err := getPotentiallyValidManifests(logCtx, appPath, repoRoot, false, "", "", resource.MustParse("34"))
// 		assert.Empty(t, manifests)
// 		assert.ErrorIs(t, err, ErrExceededMaxCombinedManifestFileSize)
// 	})

// 	t.Run("group of files should be limited at precisely the sum of their size", func(t *testing.T) {
// 		// There is a total of 10 files, ech file being 10 bytes.
// 		manifests, err := getPotentiallyValidManifests(logCtx, "./testdata/several-files", "./testdata/several-files", false, "", "", resource.MustParse("365"))
// 		assert.Len(t, manifests, 10)
// 		require.NoError(t, err)

// 		manifests, err = getPotentiallyValidManifests(logCtx, "./testdata/several-files", "./testdata/several-files", false, "", "", resource.MustParse("100"))
// 		assert.Empty(t, manifests)
// 		assert.ErrorIs(t, err, ErrExceededMaxCombinedManifestFileSize)
// 	})
// }

// func Test_findManifests(t *testing.T) {
// 	logCtx := log.WithField("test", "test")
// 	noRecurse := v1alpha1.ApplicationSourceDirectory{Recurse: false}

// 	t.Run("unreadable file throws error", func(t *testing.T) {
// 		appDir := t.TempDir()
// 		unreadablePath := filepath.Join(appDir, "unreadable.json")
// 		err := os.WriteFile(unreadablePath, []byte{}, 0o666)
// 		require.NoError(t, err)
// 		err = os.Chmod(appDir, 0o000)
// 		require.NoError(t, err)

// 		manifests, err := findManifests(logCtx, appDir, appDir, nil, noRecurse, nil, resource.MustParse("0"))
// 		assert.Empty(t, manifests)
// 		require.Error(t, err)

// 		// allow cleanup
// 		err = os.Chmod(appDir, 0o777)
// 		if err != nil {
// 			panic(err)
// 		}
// 	})

// 	t.Run("no recursion when recursion is disabled", func(t *testing.T) {
// 		manifests, err := findManifests(logCtx, "./testdata/recurse", "./testdata/recurse", nil, noRecurse, nil, resource.MustParse("0"))
// 		assert.Len(t, manifests, 2)
// 		require.NoError(t, err)
// 	})

// 	t.Run("recursion when recursion is enabled", func(t *testing.T) {
// 		recurse := v1alpha1.ApplicationSourceDirectory{Recurse: true}
// 		manifests, err := findManifests(logCtx, "./testdata/recurse", "./testdata/recurse", nil, recurse, nil, resource.MustParse("0"))
// 		assert.Len(t, manifests, 4)
// 		require.NoError(t, err)
// 	})

// 	t.Run("non-JSON/YAML is skipped", func(t *testing.T) {
// 		manifests, err := findManifests(logCtx, "./testdata/non-manifest-file", "./testdata/non-manifest-file", nil, noRecurse, nil, resource.MustParse("0"))
// 		assert.Empty(t, manifests)
// 		require.NoError(t, err)
// 	})

// 	t.Run("circular link should throw an error", func(t *testing.T) {
// 		const testDir = "./testdata/circular-link"
// 		require.DirExists(t, testDir)
// 		t.Cleanup(func() {
// 			os.Remove(path.Join(testDir, "a.json"))
// 			os.Remove(path.Join(testDir, "b.json"))
// 		})
// 		t.Chdir(testDir)
// 		require.NoError(t, fileutil.CreateSymlink(t, "a.json", "b.json"))
// 		require.NoError(t, fileutil.CreateSymlink(t, "b.json", "a.json"))
// 		manifests, err := findManifests(logCtx, "./testdata/circular-link", "./testdata/circular-link", nil, noRecurse, nil, resource.MustParse("0"))
// 		assert.Empty(t, manifests)
// 		require.Error(t, err)
// 	})

// 	t.Run("out-of-bounds symlink should throw an error", func(t *testing.T) {
// 		require.DirExists(t, "./testdata/out-of-bounds-link")
// 		manifests, err := findManifests(logCtx, "./testdata/out-of-bounds-link", "./testdata/out-of-bounds-link", nil, noRecurse, nil, resource.MustParse("0"))
// 		assert.Empty(t, manifests)
// 		require.Error(t, err)
// 	})

// 	t.Run("symlink to a regular file works", func(t *testing.T) {
// 		repoRoot, err := filepath.Abs("./testdata/in-bounds-link")
// 		require.NoError(t, err)
// 		appPath, err := filepath.Abs("./testdata/in-bounds-link/app")
// 		require.NoError(t, err)
// 		manifests, err := findManifests(logCtx, appPath, repoRoot, nil, noRecurse, nil, resource.MustParse("0"))
// 		assert.Len(t, manifests, 1)
// 		require.NoError(t, err)
// 	})

// 	t.Run("symlink to nowhere should be ignored", func(t *testing.T) {
// 		manifests, err := findManifests(logCtx, "./testdata/link-to-nowhere", "./testdata/link-to-nowhere", nil, noRecurse, nil, resource.MustParse("0"))
// 		assert.Empty(t, manifests)
// 		require.NoError(t, err)
// 	})

// 	t.Run("link to over-sized manifest fails", func(t *testing.T) {
// 		repoRoot, err := filepath.Abs("./testdata/in-bounds-link")
// 		require.NoError(t, err)
// 		appPath, err := filepath.Abs("./testdata/in-bounds-link/app")
// 		require.NoError(t, err)
// 		// The file is 35 bytes.
// 		manifests, err := findManifests(logCtx, appPath, repoRoot, nil, noRecurse, nil, resource.MustParse("34"))
// 		assert.Empty(t, manifests)
// 		assert.ErrorIs(t, err, ErrExceededMaxCombinedManifestFileSize)
// 	})

// 	t.Run("group of files should be limited at precisely the sum of their size", func(t *testing.T) {
// 		// There is a total of 10 files, each file being 10 bytes.
// 		manifests, err := findManifests(logCtx, "./testdata/several-files", "./testdata/several-files", nil, noRecurse, nil, resource.MustParse("365"))
// 		assert.Len(t, manifests, 10)
// 		require.NoError(t, err)

// 		manifests, err = findManifests(logCtx, "./testdata/several-files", "./testdata/several-files", nil, noRecurse, nil, resource.MustParse("364"))
// 		assert.Empty(t, manifests)
// 		assert.ErrorIs(t, err, ErrExceededMaxCombinedManifestFileSize)
// 	})

// 	t.Run("jsonnet isn't counted against size limit", func(t *testing.T) {
// 		// Each file is 36 bytes. Only the 36-byte json file should be counted against the limit.
// 		manifests, err := findManifests(logCtx, "./testdata/jsonnet-and-json", "./testdata/jsonnet-and-json", nil, noRecurse, nil, resource.MustParse("36"))
// 		assert.Len(t, manifests, 2)
// 		require.NoError(t, err)

// 		manifests, err = findManifests(logCtx, "./testdata/jsonnet-and-json", "./testdata/jsonnet-and-json", nil, noRecurse, nil, resource.MustParse("35"))
// 		assert.Empty(t, manifests)
// 		assert.ErrorIs(t, err, ErrExceededMaxCombinedManifestFileSize)
// 	})

// 	t.Run("partially valid YAML file throws an error", func(t *testing.T) {
// 		require.DirExists(t, "./testdata/partially-valid-yaml")
// 		manifests, err := findManifests(logCtx, "./testdata/partially-valid-yaml", "./testdata/partially-valid-yaml", nil, noRecurse, nil, resource.MustParse("0"))
// 		assert.Empty(t, manifests)
// 		require.Error(t, err)
// 	})

// 	t.Run("invalid manifest throws an error", func(t *testing.T) {
// 		require.DirExists(t, "./testdata/invalid-manifests")
// 		manifests, err := findManifests(logCtx, "./testdata/invalid-manifests", "./testdata/invalid-manifests", nil, noRecurse, nil, resource.MustParse("0"))
// 		assert.Empty(t, manifests)
// 		require.Error(t, err)
// 	})

// 	t.Run("invalid manifest containing '+argocd:skip-file-rendering' doesn't throw an error", func(t *testing.T) {
// 		require.DirExists(t, "./testdata/invalid-manifests-skipped")
// 		manifests, err := findManifests(logCtx, "./testdata/invalid-manifests-skipped", "./testdata/invalid-manifests-skipped", nil, noRecurse, nil, resource.MustParse("0"))
// 		assert.Empty(t, manifests)
// 		require.NoError(t, err)
// 	})

// 	t.Run("irrelevant YAML gets skipped, relevant YAML gets parsed", func(t *testing.T) {
// 		manifests, err := findManifests(logCtx, "./testdata/irrelevant-yaml", "./testdata/irrelevant-yaml", nil, noRecurse, nil, resource.MustParse("0"))
// 		assert.Len(t, manifests, 1)
// 		require.NoError(t, err)
// 	})

// 	t.Run("multiple JSON objects in one file throws an error", func(t *testing.T) {
// 		require.DirExists(t, "./testdata/json-list")
// 		manifests, err := findManifests(logCtx, "./testdata/json-list", "./testdata/json-list", nil, noRecurse, nil, resource.MustParse("0"))
// 		assert.Empty(t, manifests)
// 		require.Error(t, err)
// 	})

// 	t.Run("invalid JSON throws an error", func(t *testing.T) {
// 		require.DirExists(t, "./testdata/invalid-json")
// 		manifests, err := findManifests(logCtx, "./testdata/invalid-json", "./testdata/invalid-json", nil, noRecurse, nil, resource.MustParse("0"))
// 		assert.Empty(t, manifests)
// 		require.Error(t, err)
// 	})

// 	t.Run("valid JSON returns manifest and no error", func(t *testing.T) {
// 		manifests, err := findManifests(logCtx, "./testdata/valid-json", "./testdata/valid-json", nil, noRecurse, nil, resource.MustParse("0"))
// 		assert.Len(t, manifests, 1)
// 		require.NoError(t, err)
// 	})

// 	t.Run("YAML with an empty document doesn't throw an error", func(t *testing.T) {
// 		manifests, err := findManifests(logCtx, "./testdata/yaml-with-empty-document", "./testdata/yaml-with-empty-document", nil, noRecurse, nil, resource.MustParse("0"))
// 		assert.Len(t, manifests, 1)
// 		require.NoError(t, err)
// 	})
// }

// func TestTestRepoHelmOCI(t *testing.T) {
// 	service := newService(t, ".")
// 	_, err := service.TestRepository(t.Context(), &apiclient.TestRepositoryRequest{
// 		Repo: &v1alpha1.Repository{
// 			Repo:      "https://demo.goharbor.io",
// 			Type:      "helm",
// 			EnableOCI: true,
// 		},
// 	})
// 	assert.ErrorContains(t, err, "OCI Helm repository URL should include hostname and port only")
// }

// func Test_getHelmDependencyRepos(t *testing.T) {
// 	repo1 := "https://charts.bitnami.com/bitnami"
// 	repo2 := "https://eventstore.github.io/EventStore.Charts"

// 	repos, err := getHelmDependencyRepos("../../util/helm/testdata/dependency")
// 	require.NoError(t, err)
// 	assert.Len(t, repos, 2)
// 	assert.Equal(t, repos[0].Repo, repo1)
// 	assert.Equal(t, repos[1].Repo, repo2)
// }

// func TestResolveRevision(t *testing.T) {
// 	service := newService(t, ".")
// 	repo := &v1alpha1.Repository{Repo: "https://github.com/argoproj/argo-cd"}
// 	app := &v1alpha1.Application{Spec: v1alpha1.ApplicationSpec{Source: &v1alpha1.ApplicationSource{}}}
// 	resolveRevisionResponse, err := service.ResolveRevision(t.Context(), &apiclient.ResolveRevisionRequest{
// 		Repo:              repo,
// 		App:               app,
// 		AmbiguousRevision: "v2.2.2",
// 	})

// 	expectedResolveRevisionResponse := &apiclient.ResolveRevisionResponse{
// 		Revision:          "03b17e0233e64787ffb5fcf65c740cc2a20822ba",
// 		AmbiguousRevision: "v2.2.2 (03b17e0233e64787ffb5fcf65c740cc2a20822ba)",
// 	}

// 	assert.NotNil(t, resolveRevisionResponse.Revision)
// 	require.NoError(t, err)
// 	assert.Equal(t, expectedResolveRevisionResponse, resolveRevisionResponse)
// }

// func TestDirectoryPermissionInitializer(t *testing.T) {
// 	dir := t.TempDir()

// 	file, err := os.CreateTemp(dir, "")
// 	require.NoError(t, err)
// 	utilio.Close(file)

// 	// remove read permissions
// 	require.NoError(t, os.Chmod(dir, 0o000))

// 	// Remember to restore permissions when the test finishes so dir can
// 	// be removed properly.
// 	t.Cleanup(func() {
// 		require.NoError(t, os.Chmod(dir, 0o777))
// 	})

// 	// make sure permission are restored
// 	closer := directoryPermissionInitializer(dir)
// 	_, err = os.ReadFile(file.Name())
// 	require.NoError(t, err)

// 	// make sure permission are removed by closer
// 	utilio.Close(closer)
// 	_, err = os.ReadFile(file.Name())
// 	require.Error(t, err)
// }

// func addHelmToGitRepo(t *testing.T, options newGitRepoOptions) {
// 	t.Helper()
// 	err := os.WriteFile(filepath.Join(options.path, "Chart.yaml"), []byte("name: test\nversion: v1.0.0"), 0o777)
// 	require.NoError(t, err)
// 	for valuesFileName, values := range options.helmChartOptions.valuesFiles {
// 		valuesFileContents, err := yaml.Marshal(values)
// 		require.NoError(t, err)
// 		err = os.WriteFile(filepath.Join(options.path, valuesFileName), valuesFileContents, 0o777)
// 		require.NoError(t, err)
// 	}
// 	require.NoError(t, err)
// 	cmd := exec.Command("git", "add", "-A")
// 	cmd.Dir = options.path
// 	require.NoError(t, cmd.Run())
// 	cmd = exec.Command("git", "commit", "-m", "Initial commit")
// 	cmd.Dir = options.path
// 	require.NoError(t, cmd.Run())
// }

// func initGitRepo(t *testing.T, options newGitRepoOptions) (revision string) {
// 	t.Helper()
// 	if options.createPath {
// 		require.NoError(t, os.Mkdir(options.path, 0o755))
// 	}

// 	cmd := exec.Command("git", "init", "-b", "main", options.path)
// 	cmd.Dir = options.path
// 	require.NoError(t, cmd.Run())

// 	if options.remote != "" {
// 		cmd = exec.Command("git", "remote", "add", "origin", options.path)
// 		cmd.Dir = options.path
// 		require.NoError(t, cmd.Run())
// 	}

// 	commitAdded := options.addEmptyCommit || options.helmChartOptions.chartName != ""
// 	if options.addEmptyCommit {
// 		cmd = exec.Command("git", "commit", "-m", "Initial commit", "--allow-empty")
// 		cmd.Dir = options.path
// 		require.NoError(t, cmd.Run())
// 	} else if options.helmChartOptions.chartName != "" {
// 		addHelmToGitRepo(t, options)
// 	}

// 	if commitAdded {
// 		var revB bytes.Buffer
// 		cmd = exec.Command("git", "rev-parse", "HEAD", options.path)
// 		cmd.Dir = options.path
// 		cmd.Stdout = &revB
// 		require.NoError(t, cmd.Run())
// 		revision = strings.Split(revB.String(), "\n")[0]
// 	}
// 	return revision
// }

// func TestInit(t *testing.T) {
// 	dir := t.TempDir()

// 	// service.Init sets permission to 0300. Restore permissions when the test
// 	// finishes so dir can be removed properly.
// 	t.Cleanup(func() {
// 		require.NoError(t, os.Chmod(dir, 0o777))
// 	})

// 	repoPath := path.Join(dir, "repo1")
// 	initGitRepo(t, newGitRepoOptions{path: repoPath, remote: "https://github.com/argo-cd/test-repo1", createPath: true, addEmptyCommit: false})

// 	service := newService(t, ".")
// 	service.rootDir = dir

// 	require.NoError(t, service.Init())

// 	_, err := os.ReadDir(dir)
// 	require.Error(t, err)
// 	initGitRepo(t, newGitRepoOptions{path: path.Join(dir, "repo2"), remote: "https://github.com/argo-cd/test-repo2", createPath: true, addEmptyCommit: false})
// }

// // TestCheckoutRevisionCanGetNonstandardRefs shows that we can fetch a revision that points to a non-standard ref. In
// // other words, we haven't regressed and caused this issue again: https://github.com/argoproj/argo-cd/issues/4935
// func TestCheckoutRevisionCanGetNonstandardRefs(t *testing.T) {
// 	rootPath := t.TempDir()

// 	sourceRepoPath, err := os.MkdirTemp(rootPath, "")
// 	require.NoError(t, err)

// 	// Create a repo such that one commit is on a non-standard ref _and nowhere else_. This is meant to simulate, for
// 	// example, a GitHub ref for a pull into one repo from a fork of that repo.
// 	runGit(t, sourceRepoPath, "init")
// 	runGit(t, sourceRepoPath, "checkout", "-b", "main") // make sure there's a main branch to switch back to
// 	runGit(t, sourceRepoPath, "commit", "-m", "empty", "--allow-empty")
// 	runGit(t, sourceRepoPath, "checkout", "-b", "branch")
// 	runGit(t, sourceRepoPath, "commit", "-m", "empty", "--allow-empty")
// 	sha := runGit(t, sourceRepoPath, "rev-parse", "HEAD")
// 	runGit(t, sourceRepoPath, "update-ref", "refs/pull/123/head", strings.TrimSuffix(sha, "\n"))
// 	runGit(t, sourceRepoPath, "checkout", "main")
// 	runGit(t, sourceRepoPath, "branch", "-D", "branch")

// 	destRepoPath, err := os.MkdirTemp(rootPath, "")
// 	require.NoError(t, err)

// 	gitClient, err := git.NewClientExt("file://"+sourceRepoPath, destRepoPath, &git.NopCreds{}, true, false, "", "")
// 	require.NoError(t, err)

// 	pullSha, err := gitClient.LsRemote("refs/pull/123/head")
// 	require.NoError(t, err)

// 	err = checkoutRevision(gitClient, "does-not-exist", false)
// 	require.Error(t, err)

// 	err = checkoutRevision(gitClient, pullSha, false)
// 	require.NoError(t, err)
// }

// func TestCheckoutRevisionPresentSkipFetch(t *testing.T) {
// 	revision := "0123456789012345678901234567890123456789"

// 	gitClient := &gitmocks.Client{}
// 	gitClient.On("Init").Return(nil)
// 	gitClient.On("IsRevisionPresent", revision).Return(true)
// 	gitClient.On("Checkout", revision, mock.Anything).Return("", nil)

// 	err := checkoutRevision(gitClient, revision, false)
// 	require.NoError(t, err)
// }

// func TestCheckoutRevisionNotPresentCallFetch(t *testing.T) {
// 	revision := "0123456789012345678901234567890123456789"

// 	gitClient := &gitmocks.Client{}
// 	gitClient.On("Init").Return(nil)
// 	gitClient.On("IsRevisionPresent", revision).Return(false)
// 	gitClient.On("Fetch", "").Return(nil)
// 	gitClient.On("Checkout", revision, mock.Anything).Return("", nil)

// 	err := checkoutRevision(gitClient, revision, false)
// 	require.NoError(t, err)
// }

// func TestFetch(t *testing.T) {
// 	revision1 := "0123456789012345678901234567890123456789"
// 	revision2 := "abcdefabcdefabcdefabcdefabcdefabcdefabcd"

// 	gitClient := &gitmocks.Client{}
// 	gitClient.On("Init").Return(nil)
// 	gitClient.On("IsRevisionPresent", revision1).Once().Return(true)
// 	gitClient.On("IsRevisionPresent", revision2).Once().Return(false)
// 	gitClient.On("Fetch", "").Return(nil)
// 	gitClient.On("IsRevisionPresent", revision1).Once().Return(true)
// 	gitClient.On("IsRevisionPresent", revision2).Once().Return(true)

// 	err := fetch(gitClient, []string{revision1, revision2})
// 	require.NoError(t, err)
// }

// // TestFetchRevisionCanGetNonstandardRefs shows that we can fetch a revision that points to a non-standard ref. In
// func TestFetchRevisionCanGetNonstandardRefs(t *testing.T) {
// 	rootPath := t.TempDir()

// 	sourceRepoPath, err := os.MkdirTemp(rootPath, "")
// 	require.NoError(t, err)

// 	// Create a repo such that one commit is on a non-standard ref _and nowhere else_. This is meant to simulate, for
// 	// example, a GitHub ref for a pull into one repo from a fork of that repo.
// 	runGit(t, sourceRepoPath, "init")
// 	runGit(t, sourceRepoPath, "checkout", "-b", "main") // make sure there's a main branch to switch back to
// 	runGit(t, sourceRepoPath, "commit", "-m", "empty", "--allow-empty")
// 	runGit(t, sourceRepoPath, "checkout", "-b", "branch")
// 	runGit(t, sourceRepoPath, "commit", "-m", "empty", "--allow-empty")
// 	sha := runGit(t, sourceRepoPath, "rev-parse", "HEAD")
// 	runGit(t, sourceRepoPath, "update-ref", "refs/pull/123/head", strings.TrimSuffix(sha, "\n"))
// 	runGit(t, sourceRepoPath, "checkout", "main")
// 	runGit(t, sourceRepoPath, "branch", "-D", "branch")

// 	destRepoPath, err := os.MkdirTemp(rootPath, "")
// 	require.NoError(t, err)

// 	gitClient, err := git.NewClientExt("file://"+sourceRepoPath, destRepoPath, &git.NopCreds{}, true, false, "", "")
// 	require.NoError(t, err)

// 	// We should initialize repository
// 	err = gitClient.Init()
// 	require.NoError(t, err)

// 	pullSha, err := gitClient.LsRemote("refs/pull/123/head")
// 	require.NoError(t, err)

// 	err = fetch(gitClient, []string{"does-not-exist"})
// 	require.Error(t, err)

// 	err = fetch(gitClient, []string{pullSha})
// 	require.NoError(t, err)
// }

// // runGit runs a git command in the given working directory. If the command succeeds, it returns the combined standard
// // and error output. If it fails, it stops the test with a failure message.
// func runGit(t *testing.T, workDir string, args ...string) string {
// 	t.Helper()
// 	cmd := exec.Command("git", args...)
// 	cmd.Dir = workDir
// 	out, err := cmd.CombinedOutput()
// 	stringOut := string(out)
// 	require.NoError(t, err, stringOut)
// 	return stringOut
// }

// func Test_walkHelmValueFilesInPath(t *testing.T) {
// 	t.Run("does not exist", func(t *testing.T) {
// 		var files []string
// 		root := "/obviously/does/not/exist"
// 		err := filepath.Walk(root, walkHelmValueFilesInPath(root, &files))
// 		require.Error(t, err)
// 		assert.Empty(t, files)
// 	})
// 	t.Run("values files", func(t *testing.T) {
// 		var files []string
// 		root := "./testdata/values-files"
// 		err := filepath.Walk(root, walkHelmValueFilesInPath(root, &files))
// 		require.NoError(t, err)
// 		assert.Len(t, files, 5)
// 	})
// 	t.Run("unrelated root", func(t *testing.T) {
// 		var files []string
// 		root := "./testdata/values-files"
// 		unrelatedRoot := "/different/root/path"
// 		err := filepath.Walk(root, walkHelmValueFilesInPath(unrelatedRoot, &files))
// 		require.Error(t, err)
// 	})
// }

// func Test_populateHelmAppDetails(t *testing.T) {
// 	emptyTempPaths := utilio.NewRandomizedTempPaths(t.TempDir())
// 	res := apiclient.RepoAppDetailsResponse{}
// 	q := apiclient.RepoServerAppDetailsQuery{
// 		Repo: &v1alpha1.Repository{},
// 		Source: &v1alpha1.ApplicationSource{
// 			Helm: &v1alpha1.ApplicationSourceHelm{ValueFiles: []string{"exclude.yaml", "has-the-word-values.yaml"}},
// 		},
// 	}
// 	appPath, err := filepath.Abs("./testdata/values-files/")
// 	require.NoError(t, err)
// 	err = populateHelmAppDetails(&res, appPath, appPath, &q, emptyTempPaths)
// 	require.NoError(t, err)
// 	assert.Len(t, res.Helm.Parameters, 3)
// 	assert.Len(t, res.Helm.ValueFiles, 5)
// }

// func Test_populateHelmAppDetails_values_symlinks(t *testing.T) {
// 	emptyTempPaths := utilio.NewRandomizedTempPaths(t.TempDir())
// 	t.Run("inbound", func(t *testing.T) {
// 		res := apiclient.RepoAppDetailsResponse{}
// 		q := apiclient.RepoServerAppDetailsQuery{Repo: &v1alpha1.Repository{}, Source: &v1alpha1.ApplicationSource{}}
// 		err := populateHelmAppDetails(&res, "./testdata/in-bounds-values-file-link/", "./testdata/in-bounds-values-file-link/", &q, emptyTempPaths)
// 		require.NoError(t, err)
// 		assert.NotEmpty(t, res.Helm.Values)
// 		assert.NotEmpty(t, res.Helm.Parameters)
// 	})

// 	t.Run("out of bounds", func(t *testing.T) {
// 		res := apiclient.RepoAppDetailsResponse{}
// 		q := apiclient.RepoServerAppDetailsQuery{Repo: &v1alpha1.Repository{}, Source: &v1alpha1.ApplicationSource{}}
// 		err := populateHelmAppDetails(&res, "./testdata/out-of-bounds-values-file-link/", "./testdata/out-of-bounds-values-file-link/", &q, emptyTempPaths)
// 		require.NoError(t, err)
// 		assert.Empty(t, res.Helm.Values)
// 		assert.Empty(t, res.Helm.Parameters)
// 	})
// }

// func TestGetHelmRepos_OCIHelmDependenciesWithHelmRepo(t *testing.T) {
// 	q := apiclient.ManifestRequest{Repos: []*v1alpha1.Repository{}, HelmRepoCreds: []*v1alpha1.RepoCreds{
// 		{URL: "example.com", Username: "test", Password: "test", EnableOCI: true},
// 	}}

// 	helmRepos, err := getHelmRepos("./testdata/oci-dependencies", q.Repos, q.HelmRepoCreds)
// 	require.NoError(t, err)

// 	assert.Len(t, helmRepos, 1)
// 	assert.Equal(t, "test", helmRepos[0].GetUsername())
// 	assert.True(t, helmRepos[0].EnableOci)
// 	assert.Equal(t, "example.com/myrepo", helmRepos[0].Repo)
// }

// func TestGetHelmRepos_OCIHelmDependenciesWithRepo(t *testing.T) {
// 	q := apiclient.ManifestRequest{Repos: []*v1alpha1.Repository{{Repo: "example.com", Username: "test", Password: "test", EnableOCI: true}}, HelmRepoCreds: []*v1alpha1.RepoCreds{}}

// 	helmRepos, err := getHelmRepos("./testdata/oci-dependencies", q.Repos, q.HelmRepoCreds)
// 	require.NoError(t, err)

// 	assert.Len(t, helmRepos, 1)
// 	assert.Equal(t, "test", helmRepos[0].GetUsername())
// 	assert.True(t, helmRepos[0].EnableOci)
// 	assert.Equal(t, "example.com/myrepo", helmRepos[0].Repo)
// }

// func TestGetHelmRepos_OCIDependenciesWithHelmRepo(t *testing.T) {
// 	q := apiclient.ManifestRequest{Repos: []*v1alpha1.Repository{}, HelmRepoCreds: []*v1alpha1.RepoCreds{
// 		{URL: "oci://example.com", Username: "test", Password: "test", Type: "oci"},
// 	}}

// 	helmRepos, err := getHelmRepos("./testdata/oci-dependencies", q.Repos, q.HelmRepoCreds)
// 	require.NoError(t, err)

// 	assert.Len(t, helmRepos, 1)
// 	assert.Equal(t, "test", helmRepos[0].GetUsername())
// 	assert.True(t, helmRepos[0].EnableOci)
// 	assert.Equal(t, "example.com/myrepo", helmRepos[0].Repo)
// }

// func TestGetHelmRepos_OCIDependenciesWithRepo(t *testing.T) {
// 	q := apiclient.ManifestRequest{Repos: []*v1alpha1.Repository{{Repo: "oci://example.com", Username: "test", Password: "test", Type: "oci"}}, HelmRepoCreds: []*v1alpha1.RepoCreds{}}

// 	helmRepos, err := getHelmRepos("./testdata/oci-dependencies", q.Repos, q.HelmRepoCreds)
// 	require.NoError(t, err)

// 	assert.Len(t, helmRepos, 1)
// 	assert.Equal(t, "test", helmRepos[0].GetUsername())
// 	assert.True(t, helmRepos[0].EnableOci)
// 	assert.Equal(t, "example.com/myrepo", helmRepos[0].Repo)
// }

// func TestGetHelmRepo_NamedRepos(t *testing.T) {
// 	q := apiclient.ManifestRequest{
// 		Repos: []*v1alpha1.Repository{{
// 			Name:     "custom-repo",
// 			Repo:     "https://example.com",
// 			Username: "test",
// 		}},
// 	}

// 	helmRepos, err := getHelmRepos("./testdata/helm-with-dependencies", q.Repos, q.HelmRepoCreds)
// 	require.NoError(t, err)

// 	assert.Len(t, helmRepos, 1)
// 	assert.Equal(t, "test", helmRepos[0].GetUsername())
// 	assert.Equal(t, "https://example.com", helmRepos[0].Repo)
// }

// func TestGetHelmRepo_NamedReposAlias(t *testing.T) {
// 	q := apiclient.ManifestRequest{
// 		Repos: []*v1alpha1.Repository{{
// 			Name:     "custom-repo-alias",
// 			Repo:     "https://example.com",
// 			Username: "test-alias",
// 		}},
// 	}

// 	helmRepos, err := getHelmRepos("./testdata/helm-with-dependencies-alias", q.Repos, q.HelmRepoCreds)
// 	require.NoError(t, err)

// 	assert.Len(t, helmRepos, 1)
// 	assert.Equal(t, "test-alias", helmRepos[0].GetUsername())
// 	assert.Equal(t, "https://example.com", helmRepos[0].Repo)
// }

// func Test_getResolvedValueFiles(t *testing.T) {
// 	t.Parallel()

// 	tempDir := t.TempDir()
// 	paths := utilio.NewRandomizedTempPaths(tempDir)

// 	paths.Add(git.NormalizeGitURL("https://github.com/org/repo1"), path.Join(tempDir, "repo1"))

// 	testCases := []struct {
// 		name         string
// 		rawPath      string
// 		env          *v1alpha1.Env
// 		refSources   map[string]*v1alpha1.RefTarget
// 		expectedPath string
// 		expectedErr  bool
// 	}{
// 		{
// 			name:         "simple path",
// 			rawPath:      "values.yaml",
// 			env:          &v1alpha1.Env{},
// 			refSources:   map[string]*v1alpha1.RefTarget{},
// 			expectedPath: path.Join(tempDir, "main-repo", "values.yaml"),
// 		},
// 		{
// 			name:    "simple ref",
// 			rawPath: "$ref/values.yaml",
// 			env:     &v1alpha1.Env{},
// 			refSources: map[string]*v1alpha1.RefTarget{
// 				"$ref": {
// 					Repo: v1alpha1.Repository{
// 						Repo: "https://github.com/org/repo1",
// 					},
// 				},
// 			},
// 			expectedPath: path.Join(tempDir, "repo1", "values.yaml"),
// 		},
// 		{
// 			name:    "only ref",
// 			rawPath: "$ref",
// 			env:     &v1alpha1.Env{},
// 			refSources: map[string]*v1alpha1.RefTarget{
// 				"$ref": {
// 					Repo: v1alpha1.Repository{
// 						Repo: "https://github.com/org/repo1",
// 					},
// 				},
// 			},
// 			expectedErr: true,
// 		},
// 		{
// 			name:    "attempted traversal",
// 			rawPath: "$ref/../values.yaml",
// 			env:     &v1alpha1.Env{},
// 			refSources: map[string]*v1alpha1.RefTarget{
// 				"$ref": {
// 					Repo: v1alpha1.Repository{
// 						Repo: "https://github.com/org/repo1",
// 					},
// 				},
// 			},
// 			expectedErr: true,
// 		},
// 		{
// 			// Since $ref doesn't resolve to a ref target, we assume it's an env var. Since the env var isn't specified,
// 			// it's replaced with an empty string. This is necessary for backwards compatibility with behavior before
// 			// ref targets were introduced.
// 			name:         "ref doesn't exist",
// 			rawPath:      "$ref/values.yaml",
// 			env:          &v1alpha1.Env{},
// 			refSources:   map[string]*v1alpha1.RefTarget{},
// 			expectedPath: path.Join(tempDir, "main-repo", "values.yaml"),
// 		},
// 		{
// 			name:    "repo doesn't exist",
// 			rawPath: "$ref/values.yaml",
// 			env:     &v1alpha1.Env{},
// 			refSources: map[string]*v1alpha1.RefTarget{
// 				"$ref": {
// 					Repo: v1alpha1.Repository{
// 						Repo: "https://github.com/org/repo2",
// 					},
// 				},
// 			},
// 			expectedErr: true,
// 		},
// 		{
// 			name:    "env var is resolved",
// 			rawPath: "$ref/$APP_PATH/values.yaml",
// 			env: &v1alpha1.Env{
// 				&v1alpha1.EnvEntry{
// 					Name:  "APP_PATH",
// 					Value: "app-path",
// 				},
// 			},
// 			refSources: map[string]*v1alpha1.RefTarget{
// 				"$ref": {
// 					Repo: v1alpha1.Repository{
// 						Repo: "https://github.com/org/repo1",
// 					},
// 				},
// 			},
// 			expectedPath: path.Join(tempDir, "repo1", "app-path", "values.yaml"),
// 		},
// 		{
// 			name:    "traversal in env var is blocked",
// 			rawPath: "$ref/$APP_PATH/values.yaml",
// 			env: &v1alpha1.Env{
// 				&v1alpha1.EnvEntry{
// 					Name:  "APP_PATH",
// 					Value: "..",
// 				},
// 			},
// 			refSources: map[string]*v1alpha1.RefTarget{
// 				"$ref": {
// 					Repo: v1alpha1.Repository{
// 						Repo: "https://github.com/org/repo1",
// 					},
// 				},
// 			},
// 			expectedErr: true,
// 		},
// 		{
// 			name:    "env var prefix",
// 			rawPath: "$APP_PATH/values.yaml",
// 			env: &v1alpha1.Env{
// 				&v1alpha1.EnvEntry{
// 					Name:  "APP_PATH",
// 					Value: "app-path",
// 				},
// 			},
// 			refSources:   map[string]*v1alpha1.RefTarget{},
// 			expectedPath: path.Join(tempDir, "main-repo", "app-path", "values.yaml"),
// 		},
// 		{
// 			name:         "unresolved env var",
// 			rawPath:      "$APP_PATH/values.yaml",
// 			env:          &v1alpha1.Env{},
// 			refSources:   map[string]*v1alpha1.RefTarget{},
// 			expectedPath: path.Join(tempDir, "main-repo", "values.yaml"),
// 		},
// 	}

// 	for _, tc := range testCases {
// 		tcc := tc
// 		t.Run(tcc.name, func(t *testing.T) {
// 			t.Parallel()
// 			resolvedPaths, err := getResolvedValueFiles(path.Join(tempDir, "main-repo"), path.Join(tempDir, "main-repo"), tcc.env, []string{}, []string{tcc.rawPath}, tcc.refSources, paths, false)
// 			if !tcc.expectedErr {
// 				require.NoError(t, err)
// 				require.Len(t, resolvedPaths, 1)
// 				assert.Equal(t, tcc.expectedPath, string(resolvedPaths[0]))
// 			} else {
// 				require.Error(t, err)
// 				assert.Empty(t, resolvedPaths)
// 			}
// 		})
// 	}
// }

// func TestGetRefs_CacheWithLockDisabled(t *testing.T) {
// 	// Test that when the lock is disabled the default behavior still works correctly
// 	// Also shows the current issue with the git requests due to cache misses
// 	dir := t.TempDir()
// 	initGitRepo(t, newGitRepoOptions{
// 		path:           dir,
// 		createPath:     false,
// 		remote:         "",
// 		addEmptyCommit: true,
// 	})
// 	// Test in-memory and redis
// 	cacheMocks := newCacheMocksWithOpts(1*time.Minute, 1*time.Minute, 0)
// 	t.Cleanup(cacheMocks.mockCache.StopRedisCallback)
// 	var wg sync.WaitGroup
// 	numberOfCallers := 10
// 	for i := 0; i < numberOfCallers; i++ {
// 		wg.Add(1)
// 		go func() {
// 			defer wg.Done()
// 			client, err := git.NewClient("file://"+dir, git.NopCreds{}, true, false, "", "", git.WithCache(cacheMocks.cache, true))
// 			require.NoError(t, err)
// 			refs, err := client.LsRefs()
// 			require.NoError(t, err)
// 			assert.NotNil(t, refs)
// 			assert.NotEmpty(t, refs.Branches, "Expected branches to be populated")
// 			assert.NotEmpty(t, refs.Branches[0])
// 		}()
// 	}
// 	wg.Wait()
// 	// Unlock should not have been called
// 	cacheMocks.mockCache.AssertNumberOfCalls(t, "UnlockGitReferences", 0)
// 	// Lock should not have been called
// 	cacheMocks.mockCache.AssertNumberOfCalls(t, "TryLockGitRefCache", 0)
// }

// func TestGetRefs_CacheDisabled(t *testing.T) {
// 	// Test that default get refs with cache disabled does not call GetOrLockGitReferences
// 	dir := t.TempDir()
// 	initGitRepo(t, newGitRepoOptions{
// 		path:           dir,
// 		createPath:     false,
// 		remote:         "",
// 		addEmptyCommit: true,
// 	})
// 	cacheMocks := newCacheMocks()
// 	t.Cleanup(cacheMocks.mockCache.StopRedisCallback)
// 	client, err := git.NewClient("file://"+dir, git.NopCreds{}, true, false, "", "", git.WithCache(cacheMocks.cache, false))
// 	require.NoError(t, err)
// 	refs, err := client.LsRefs()
// 	require.NoError(t, err)
// 	assert.NotNil(t, refs)
// 	assert.NotEmpty(t, refs.Branches, "Expected branches to be populated")
// 	assert.NotEmpty(t, refs.Branches[0])
// 	// Unlock should not have been called
// 	cacheMocks.mockCache.AssertNumberOfCalls(t, "UnlockGitReferences", 0)
// 	cacheMocks.mockCache.AssertNumberOfCalls(t, "GetOrLockGitReferences", 0)
// }

// func TestGetRefs_CacheWithLock(t *testing.T) {
// 	// Test that there is only one call to SetGitReferences for the same repo which is done after the ls-remote
// 	dir := t.TempDir()
// 	initGitRepo(t, newGitRepoOptions{
// 		path:           dir,
// 		createPath:     false,
// 		remote:         "",
// 		addEmptyCommit: true,
// 	})
// 	cacheMocks := newCacheMocks()
// 	t.Cleanup(cacheMocks.mockCache.StopRedisCallback)
// 	var wg sync.WaitGroup
// 	numberOfCallers := 10
// 	for i := 0; i < numberOfCallers; i++ {
// 		wg.Add(1)
// 		go func() {
// 			defer wg.Done()
// 			client, err := git.NewClient("file://"+dir, git.NopCreds{}, true, false, "", "", git.WithCache(cacheMocks.cache, true))
// 			require.NoError(t, err)
// 			refs, err := client.LsRefs()
// 			require.NoError(t, err)
// 			assert.NotNil(t, refs)
// 			assert.NotEmpty(t, refs.Branches, "Expected branches to be populated")
// 			assert.NotEmpty(t, refs.Branches[0])
// 		}()
// 	}
// 	wg.Wait()
// 	// Unlock should not have been called
// 	cacheMocks.mockCache.AssertNumberOfCalls(t, "UnlockGitReferences", 0)
// 	cacheMocks.mockCache.AssertNumberOfCalls(t, "GetOrLockGitReferences", 0)
// }

// func TestGetRefs_CacheUnlockedOnUpdateFailed(t *testing.T) {
// 	// Worst case the ttl on the lock expires and the lock is removed
// 	// however if the holder of the lock fails to update the cache the caller should remove the lock
// 	// to allow other callers to attempt to update the cache as quickly as possible
// 	dir := t.TempDir()
// 	initGitRepo(t, newGitRepoOptions{
// 		path:           dir,
// 		createPath:     false,
// 		remote:         "",
// 		addEmptyCommit: true,
// 	})
// 	cacheMocks := newCacheMocks()
// 	t.Cleanup(cacheMocks.mockCache.StopRedisCallback)
// 	repoURL := "file://" + dir
// 	client, err := git.NewClient(repoURL, git.NopCreds{}, true, false, "", "", git.WithCache(cacheMocks.cache, true))
// 	require.NoError(t, err)
// 	refs, err := client.LsRefs()
// 	require.NoError(t, err)
// 	assert.NotNil(t, refs)
// 	assert.NotEmpty(t, refs.Branches, "Expected branches to be populated")
// 	assert.NotEmpty(t, refs.Branches[0])
// 	var output [][2]string
// 	err = cacheMocks.cacheutilCache.GetItem(fmt.Sprintf("git-refs|%s|%s", repoURL, common.CacheVersion), &output)
// 	require.Error(t, err, "Should be a cache miss")
// 	assert.Empty(t, output, "Expected cache to be empty for key")
// 	cacheMocks.mockCache.AssertNumberOfCalls(t, "UnlockGitReferences", 0)
// 	cacheMocks.mockCache.AssertNumberOfCalls(t, "GetOrLockGitReferences", 0)
// }

// func TestGetRefs_CacheLockTryLockGitRefCacheError(t *testing.T) {
// 	// Worst case the ttl on the lock expires and the lock is removed
// 	// however if the holder of the lock fails to update the cache the caller should remove the lock
// 	// to allow other callers to attempt to update the cache as quickly as possible
// 	dir := t.TempDir()
// 	initGitRepo(t, newGitRepoOptions{
// 		path:           dir,
// 		createPath:     false,
// 		remote:         "",
// 		addEmptyCommit: true,
// 	})
// 	cacheMocks := newCacheMocks()
// 	t.Cleanup(cacheMocks.mockCache.StopRedisCallback)
// 	repoURL := "file://" + dir
// 	// buf := bytes.Buffer{}
// 	// log.SetOutput(&buf)
// 	client, err := git.NewClient(repoURL, git.NopCreds{}, true, false, "", "", git.WithCache(cacheMocks.cache, true))
// 	require.NoError(t, err)
// 	refs, err := client.LsRefs()
// 	require.NoError(t, err)
// 	assert.NotNil(t, refs)
// }

// func TestGetRevisionChartDetails(t *testing.T) {
// 	t.Run("Test revision semver", func(t *testing.T) {
// 		root := t.TempDir()
// 		service := newService(t, root)
// 		_, err := service.GetRevisionChartDetails(t.Context(), &apiclient.RepoServerRevisionChartDetailsRequest{
// 			Repo: &v1alpha1.Repository{
// 				Repo: "file://" + root,
// 				Name: "test-repo-name",
// 				Type: "helm",
// 			},
// 			Name:     "test-name",
// 			Revision: "test-revision",
// 		})
// 		assert.ErrorContains(t, err, "invalid revision")
// 	})

// 	t.Run("Test GetRevisionChartDetails", func(t *testing.T) {
// 		root := t.TempDir()
// 		service := newService(t, root)
// 		repoURL := "file://" + root
// 		err := service.cache.SetRevisionChartDetails(repoURL, "my-chart", "1.1.0", &v1alpha1.ChartDetails{
// 			Description: "test-description",
// 			Home:        "test-home",
// 			Maintainers: []string{"test-maintainer"},
// 		})
// 		require.NoError(t, err)
// 		chartDetails, err := service.GetRevisionChartDetails(t.Context(), &apiclient.RepoServerRevisionChartDetailsRequest{
// 			Repo: &v1alpha1.Repository{
// 				Repo: "file://" + root,
// 				Name: "test-repo-name",
// 				Type: "helm",
// 			},
// 			Name:     "my-chart",
// 			Revision: "1.1.0",
// 		})
// 		require.NoError(t, err)
// 		assert.Equal(t, "test-description", chartDetails.Description)
// 		assert.Equal(t, "test-home", chartDetails.Home)
// 		assert.Equal(t, []string{"test-maintainer"}, chartDetails.Maintainers)
// 	})
// }

// func TestVerifyCommitSignature(t *testing.T) {
// 	repo := &v1alpha1.Repository{
// 		Repo: "https://github.com/example/repo.git",
// 	}

// 	t.Run("VerifyCommitSignature with valid signature", func(t *testing.T) {
// 		t.Setenv("ARGOCD_GPG_ENABLED", "true")
// 		mockGitClient := &gitmocks.Client{}
// 		mockGitClient.On("VerifyCommitSignature", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
// 			Return(testSignature, nil)
// 		err := verifyCommitSignature(true, mockGitClient, "abcd1234", repo)
// 		require.NoError(t, err)
// 	})

// 	t.Run("VerifyCommitSignature with invalid signature", func(t *testing.T) {
// 		t.Setenv("ARGOCD_GPG_ENABLED", "true")
// 		mockGitClient := &gitmocks.Client{}
// 		mockGitClient.On("VerifyCommitSignature", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
// 			Return("", nil)
// 		err := verifyCommitSignature(true, mockGitClient, "abcd1234", repo)
// 		assert.EqualError(t, err, "revision abcd1234 is not signed")
// 	})

// 	t.Run("VerifyCommitSignature with unknown signature", func(t *testing.T) {
// 		t.Setenv("ARGOCD_GPG_ENABLED", "true")
// 		mockGitClient := &gitmocks.Client{}
// 		mockGitClient.On("VerifyCommitSignature", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
// 			Return("", errors.New("UNKNOWN signature: gpg: Unknown signature from ABCDEFGH"))
// 		err := verifyCommitSignature(true, mockGitClient, "abcd1234", repo)
// 		assert.EqualError(t, err, "UNKNOWN signature: gpg: Unknown signature from ABCDEFGH")
// 	})

// 	t.Run("VerifyCommitSignature with error verifying signature", func(t *testing.T) {
// 		t.Setenv("ARGOCD_GPG_ENABLED", "true")
// 		mockGitClient := &gitmocks.Client{}
// 		mockGitClient.On("VerifyCommitSignature", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
// 			Return("", errors.New("error verifying signature of commit 'abcd1234' in repo 'https://github.com/example/repo.git': failed to verify signature"))
// 		err := verifyCommitSignature(true, mockGitClient, "abcd1234", repo)
// 		assert.EqualError(t, err, "error verifying signature of commit 'abcd1234' in repo 'https://github.com/example/repo.git': failed to verify signature")
// 	})

// 	t.Run("VerifyCommitSignature with signature verification disabled", func(t *testing.T) {
// 		t.Setenv("ARGOCD_GPG_ENABLED", "false")
// 		mockGitClient := &gitmocks.Client{}
// 		err := verifyCommitSignature(false, mockGitClient, "abcd1234", repo)
// 		require.NoError(t, err)
// 	})
// }

// func Test_GenerateManifests_Commands(t *testing.T) {
// 	t.Run("helm", func(t *testing.T) {
// 		service := newService(t, "testdata/my-chart")

// 		// Fill the manifest request with as many parameters affecting Helm commands as possible.
// 		q := apiclient.ManifestRequest{
// 			AppName:     "test-app",
// 			Namespace:   "test-namespace",
// 			KubeVersion: "1.2.3",
// 			ApiVersions: []string{"v1/Test", "v2/Test"},
// 			Repo:        &v1alpha1.Repository{},
// 			ApplicationSource: &v1alpha1.ApplicationSource{
// 				Path: ".",
// 				Helm: &v1alpha1.ApplicationSourceHelm{
// 					FileParameters: []v1alpha1.HelmFileParameter{
// 						{
// 							Name: "test-file-param-name",
// 							Path: "test-file-param.yaml",
// 						},
// 					},
// 					Parameters: []v1alpha1.HelmParameter{
// 						{
// 							Name: "test-param-name",
// 							// Use build env var to test substitution.
// 							Value:       "test-value-$ARGOCD_APP_NAME",
// 							ForceString: true,
// 						},
// 						{
// 							Name: "test-param-bool-name",
// 							// Use build env var to test substitution.
// 							Value: "false",
// 						},
// 					},
// 					PassCredentials:      true,
// 					SkipCrds:             true,
// 					SkipSchemaValidation: false,
// 					ValueFiles: []string{
// 						"my-chart-values.yaml",
// 					},
// 					Values: "test: values",
// 				},
// 			},
// 			ProjectName:        "something",
// 			ProjectSourceRepos: []string{"*"},
// 		}

// 		res, err := service.GenerateManifest(t.Context(), &q)

// 		require.NoError(t, err)
// 		assert.Equal(t, []string{"helm template . --name-template test-app --namespace test-namespace --kube-version 1.2.3 --set test-param-bool-name=false --set-string test-param-name=test-value-test-app --set-file test-file-param-name=./test-file-param.yaml --values ./my-chart-values.yaml --values <temp file with values from source.helm.values/valuesObject> --api-versions v1/Test --api-versions v2/Test"}, res.Commands)

// 		t.Run("with overrides", func(t *testing.T) {
// 			// These can be set explicitly instead of using inferred values. Make sure the overrides apply.
// 			q.ApplicationSource.Helm.APIVersions = []string{"v3", "v4"}
// 			q.ApplicationSource.Helm.KubeVersion = "5.6.7"
// 			q.ApplicationSource.Helm.Namespace = "different-namespace"
// 			q.ApplicationSource.Helm.ReleaseName = "different-release-name"

// 			res, err = service.GenerateManifest(t.Context(), &q)

// 			require.NoError(t, err)
// 			assert.Equal(t, []string{"helm template . --name-template different-release-name --namespace different-namespace --kube-version 5.6.7 --set test-param-bool-name=false --set-string test-param-name=test-value-test-app --set-file test-file-param-name=./test-file-param.yaml --values ./my-chart-values.yaml --values <temp file with values from source.helm.values/valuesObject> --api-versions v3 --api-versions v4"}, res.Commands)
// 		})
// 	})

// 	t.Run("helm with dependencies", func(t *testing.T) {
// 		// This test makes sure we still get commands, even if we hit the code path that has to run "helm dependency build."
// 		// We don't actually return the "helm dependency build" command, because we expect that the user is able to read
// 		// the "helm template" and figure out how to fix it.
// 		t.Cleanup(func() {
// 			err := os.Remove("testdata/helm-with-local-dependency/Chart.lock")
// 			require.NoError(t, err)
// 			err = os.RemoveAll("testdata/helm-with-local-dependency/charts")
// 			require.NoError(t, err)
// 			err = os.Remove(path.Join("testdata/helm-with-local-dependency", helmDepUpMarkerFile))
// 			require.NoError(t, err)
// 		})

// 		service := newService(t, "testdata/helm-with-local-dependency")

// 		q := apiclient.ManifestRequest{
// 			AppName:   "test-app",
// 			Namespace: "test-namespace",
// 			Repo:      &v1alpha1.Repository{},
// 			ApplicationSource: &v1alpha1.ApplicationSource{
// 				Path: ".",
// 			},
// 			ProjectName:        "something",
// 			ProjectSourceRepos: []string{"*"},
// 		}

// 		res, err := service.GenerateManifest(t.Context(), &q)

// 		require.NoError(t, err)
// 		assert.Equal(t, []string{"helm template . --name-template test-app --namespace test-namespace --include-crds"}, res.Commands)
// 	})

// 	t.Run("kustomize", func(t *testing.T) {
// 		// Write test files to a temp dir, because the test mutates kustomization.yaml in place.
// 		tempDir := t.TempDir()
// 		err := os.WriteFile(path.Join(tempDir, "kustomization.yaml"), []byte(`
// resources:
// - guestbook.yaml
// `), os.FileMode(0o600))
// 		require.NoError(t, err)
// 		err = os.WriteFile(path.Join(tempDir, "guestbook.yaml"), []byte(`
// apiVersion: apps/v1
// kind: Deployment
// metadata:
//   name: guestbook-ui
// `), os.FileMode(0o400))
// 		require.NoError(t, err)
// 		err = os.Mkdir(path.Join(tempDir, "component"), os.FileMode(0o700))
// 		require.NoError(t, err)
// 		err = os.WriteFile(path.Join(tempDir, "component", "kustomization.yaml"), []byte(`
// apiVersion: kustomize.config.k8s.io/v1alpha1
// kind: Component
// images:
// - name: old
//   newName: new
// `), os.FileMode(0o400))
// 		require.NoError(t, err)

// 		service := newService(t, tempDir)

// 		// Fill the manifest request with as many parameters affecting Kustomize commands as possible.
// 		q := apiclient.ManifestRequest{
// 			AppName:     "test-app",
// 			Namespace:   "test-namespace",
// 			KubeVersion: "1.2.3",
// 			ApiVersions: []string{"v1/Test", "v2/Test"},
// 			Repo:        &v1alpha1.Repository{},
// 			ApplicationSource: &v1alpha1.ApplicationSource{
// 				Path: ".",
// 				Kustomize: &v1alpha1.ApplicationSourceKustomize{
// 					APIVersions: []string{"v1", "v2"},
// 					CommonAnnotations: map[string]string{
// 						// Use build env var to test substitution.
// 						"test": "annotation-$ARGOCD_APP_NAME",
// 					},
// 					CommonAnnotationsEnvsubst: true,
// 					CommonLabels: map[string]string{
// 						"test": "label",
// 					},
// 					Components:             []string{"component"},
// 					ForceCommonAnnotations: true,
// 					ForceCommonLabels:      true,
// 					Images: v1alpha1.KustomizeImages{
// 						"image=override",
// 					},
// 					KubeVersion:           "5.6.7",
// 					LabelWithoutSelector:  true,
// 					LabelIncludeTemplates: true,
// 					NamePrefix:            "test-prefix",
// 					NameSuffix:            "test-suffix",
// 					Namespace:             "override-namespace",
// 					Replicas: v1alpha1.KustomizeReplicas{
// 						{
// 							Name:  "guestbook-ui",
// 							Count: intstr.Parse("1337"),
// 						},
// 					},
// 				},
// 			},
// 			ProjectName:        "something",
// 			ProjectSourceRepos: []string{"*"},
// 		}

// 		res, err := service.GenerateManifest(t.Context(), &q)

// 		require.NoError(t, err)
// 		assert.Equal(t, []string{
// 			"kustomize edit set nameprefix -- test-prefix",
// 			"kustomize edit set namesuffix -- test-suffix",
// 			"kustomize edit set image image=override",
// 			"kustomize edit set replicas guestbook-ui=1337",
// 			"kustomize edit add label --force --without-selector --include-templates test:label",
// 			"kustomize edit add annotation --force test:annotation-test-app",
// 			"kustomize edit set namespace -- override-namespace",
// 			"kustomize edit add component component",
// 			"kustomize build .",
// 		}, res.Commands)
// 	})
// }

// func Test_SkipSchemaValidation(t *testing.T) {
// 	t.Run("helm", func(t *testing.T) {
// 		service := newService(t, "testdata/broken-schema-verification")

// 		q := apiclient.ManifestRequest{
// 			AppName: "test-app",
// 			Repo:    &v1alpha1.Repository{},
// 			ApplicationSource: &v1alpha1.ApplicationSource{
// 				Path: ".",
// 				Helm: &v1alpha1.ApplicationSourceHelm{
// 					SkipSchemaValidation: true,
// 				},
// 			},
// 		}

// 		res, err := service.GenerateManifest(t.Context(), &q)

// 		require.NoError(t, err)
// 		assert.Equal(t, []string{"helm template . --name-template test-app --include-crds --skip-schema-validation"}, res.Commands)
// 	})
// 	t.Run("helm", func(t *testing.T) {
// 		service := newService(t, "testdata/broken-schema-verification")

// 		q := apiclient.ManifestRequest{
// 			AppName: "test-app",
// 			Repo:    &v1alpha1.Repository{},
// 			ApplicationSource: &v1alpha1.ApplicationSource{
// 				Path: ".",
// 				Helm: &v1alpha1.ApplicationSourceHelm{
// 					SkipSchemaValidation: false,
// 				},
// 			},
// 		}

// 		_, err := service.GenerateManifest(t.Context(), &q)

// 		require.ErrorContains(t, err, "values don't meet the specifications of the schema(s)")
// 	})
// }

// func TestGenerateManifest_OCISourceSkipsGitClient(t *testing.T) {
// 	svc := newService(t, t.TempDir())

// 	gitCalled := false
// 	svc.newGitClient = func(_, _ string, _ git.Creds, _, _ bool, _, _ string, _ ...git.ClientOpts) (git.Client, error) {
// 		gitCalled = true
// 		return nil, errors.New("git should not be called for OCI")
// 	}

// 	req := &apiclient.ManifestRequest{
// 		HasMultipleSources: true,
// 		Repo: &v1alpha1.Repository{
// 			Repo: "oci://example.com/foo",
// 		},
// 		ApplicationSource: &v1alpha1.ApplicationSource{
// 			Path:           "",
// 			TargetRevision: "v1",
// 			Ref:            "foo",
// 			RepoURL:        "oci://example.com/foo",
// 		},
// 		ProjectName:        "foo-project",
// 		ProjectSourceRepos: []string{"*"},
// 	}

// 	_, err := svc.GenerateManifest(t.Context(), req)
// 	require.NoError(t, err)

// 	// verify that newGitClient was never invoked
// 	assert.False(t, gitCalled, "GenerateManifest should not invoke Git for OCI sources")
// }

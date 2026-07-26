//go:build unit

package service

import (
	"archive/zip"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type updateServiceCacheStub struct {
	data string
}

func (s *updateServiceCacheStub) GetUpdateInfo(context.Context) (string, error) {
	if s.data == "" {
		return "", errors.New("cache miss")
	}
	return s.data, nil
}

func (s *updateServiceCacheStub) SetUpdateInfo(_ context.Context, data string, _ time.Duration) error {
	s.data = data
	return nil
}

type updateServiceGitHubClientStub struct {
	release        *GitHubRelease
	recentReleases []*GitHubRelease
	recentErr      error
	latestRepo     string
	recentRepo     string
}

func (s *updateServiceGitHubClientStub) FetchLatestRelease(_ context.Context, repo string) (*GitHubRelease, error) {
	s.latestRepo = repo
	return s.release, nil
}

func (s *updateServiceGitHubClientStub) FetchRecentReleases(_ context.Context, repo string, _ int) ([]*GitHubRelease, error) {
	s.recentRepo = repo
	return s.recentReleases, s.recentErr
}

func (s *updateServiceGitHubClientStub) DownloadFile(context.Context, string, string, int64) error {
	panic("DownloadFile should not be called when no update is available")
}

func (s *updateServiceGitHubClientStub) FetchChecksumFile(context.Context, string) ([]byte, error) {
	panic("FetchChecksumFile should not be called when no update is available")
}

func TestUpdateServicePerformUpdateNoUpdateReturnsSentinel(t *testing.T) {
	client := &updateServiceGitHubClientStub{
		recentReleases: []*GitHubRelease{{
			TagName: "v-custom.20260726.1",
			Name:    "v-custom.20260726.1",
		}},
	}
	svc := NewUpdateService(
		&updateServiceCacheStub{},
		client,
		"custom.20260726.1",
		"release",
	)

	err := svc.PerformUpdate(context.Background())

	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNoUpdateAvailable))
	require.ErrorIs(t, err, ErrNoUpdateAvailable)
	require.Equal(t, "YeTianXingShi/sub2api", client.recentRepo)
}

func TestUpdateServiceSkipsNonCustomLatestRelease(t *testing.T) {
	client := &updateServiceGitHubClientStub{recentReleases: []*GitHubRelease{
		{TagName: "v9.9.9"},
		{TagName: "v-custom.20260726.2"},
	}}
	svc := NewUpdateService(&updateServiceCacheStub{}, client, "custom.20260726.1", "release")

	info, err := svc.CheckUpdate(context.Background(), true)

	require.NoError(t, err)
	require.Equal(t, "custom.20260726.2", info.LatestVersion)
	require.True(t, info.HasUpdate)
}

func TestCompareCustomVersionsByDateAndSequence(t *testing.T) {
	require.Less(t, compareVersions("custom.20260725.9", "custom.20260726.1"), 0)
	require.Less(t, compareVersions("v-custom.20260726.1", "custom.20260726.2"), 0)
	require.Zero(t, compareVersions("v-custom.20260726.2", "custom.20260726.2"))
	require.Greater(t, compareVersions("custom.20260726.10", "custom.20260726.2"), 0)
	_, _, ok := parseCustomVersion("custom.20261340.1")
	require.False(t, ok)
}

func TestExtractBinaryFromWindowsArchive(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "sub2api_custom.20260726.1_windows_amd64.zip")
	archive, err := os.Create(archivePath)
	require.NoError(t, err)
	zw := zip.NewWriter(archive)
	entry, err := zw.Create("sub2api.exe")
	require.NoError(t, err)
	_, err = entry.Write([]byte("windows-binary"))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	require.NoError(t, archive.Close())

	dest := filepath.Join(dir, "installed.exe")
	svc := &UpdateService{}
	require.NoError(t, svc.extractBinary(archivePath, dest))
	content, err := os.ReadFile(dest)
	require.NoError(t, err)
	require.Equal(t, []byte("windows-binary"), content)
}

func newRollbackTestService(current string, releases []*GitHubRelease) *UpdateService {
	return NewUpdateService(
		&updateServiceCacheStub{},
		&updateServiceGitHubClientStub{recentReleases: releases},
		current,
		"release",
	)
}

func TestUpdateServiceListRollbackVersionsFiltersAndCaps(t *testing.T) {
	releases := []*GitHubRelease{
		{TagName: "v-custom.20260709.1", PublishedAt: "2026-07-09T00:00:00Z"},                   // newer than current: excluded
		{TagName: "v-custom.20260708.1", PublishedAt: "2026-07-08T00:00:00Z"},                   // current: excluded
		{TagName: "v-custom.20260707.2", PublishedAt: "2026-07-07T12:00:00Z", Prerelease: true}, // prerelease: excluded
		{TagName: "v-custom.20260707.1", PublishedAt: "2026-07-07T00:00:00Z"},
		{TagName: "v-custom.20260706.1", PublishedAt: "2026-07-06T00:00:00Z", Draft: true}, // draft: excluded
		{TagName: "v-custom.20260705.1", PublishedAt: "2026-07-05T00:00:00Z"},
		{TagName: "v-custom.20260705.1", PublishedAt: "2026-07-05T00:00:00Z"}, // duplicate: excluded
		{TagName: "v-custom.20260704.1", PublishedAt: "2026-07-04T00:00:00Z"},
		{TagName: "v-custom.20260703.1", PublishedAt: "2026-07-03T00:00:00Z"}, // beyond cap of 3: excluded
	}
	svc := newRollbackTestService("custom.20260708.1", releases)

	versions, err := svc.ListRollbackVersions(context.Background())

	require.NoError(t, err)
	require.Len(t, versions, 3)
	require.Equal(t, "custom.20260707.1", versions[0].Version)
	require.Equal(t, "custom.20260705.1", versions[1].Version)
	require.Equal(t, "custom.20260704.1", versions[2].Version)
}

func TestUpdateServiceListRollbackVersionsSortsUnorderedInput(t *testing.T) {
	releases := []*GitHubRelease{
		{TagName: "v-custom.20260705.1"},
		{TagName: "v-custom.20260707.1"},
		{TagName: "v-custom.20260706.1"},
	}
	svc := newRollbackTestService("custom.20260708.1", releases)

	versions, err := svc.ListRollbackVersions(context.Background())

	require.NoError(t, err)
	require.Len(t, versions, 3)
	require.Equal(t, "custom.20260707.1", versions[0].Version)
	require.Equal(t, "custom.20260706.1", versions[1].Version)
	require.Equal(t, "custom.20260705.1", versions[2].Version)
}

func TestUpdateServiceListRollbackVersionsEmptyWhenNoneOlder(t *testing.T) {
	releases := []*GitHubRelease{
		{TagName: "v-custom.20260708.1"},
		{TagName: "v-custom.20260709.1"},
	}
	svc := newRollbackTestService("custom.20260708.1", releases)

	versions, err := svc.ListRollbackVersions(context.Background())

	require.NoError(t, err)
	require.Empty(t, versions)
}

func TestUpdateServiceListRollbackVersionsPropagatesFetchError(t *testing.T) {
	svc := NewUpdateService(
		&updateServiceCacheStub{},
		&updateServiceGitHubClientStub{recentErr: errors.New("github unavailable")},
		"custom.20260708.1",
		"release",
	)

	_, err := svc.ListRollbackVersions(context.Background())

	require.Error(t, err)
	require.Contains(t, err.Error(), "github unavailable")
}

func TestUpdateServiceRollbackToVersionRejectsDisallowedTargets(t *testing.T) {
	releases := []*GitHubRelease{
		{TagName: "v-custom.20260709.1"},
		{TagName: "v-custom.20260708.1"},
		{TagName: "v-custom.20260707.1"},
		{TagName: "v-custom.20260706.1"},
		{TagName: "v-custom.20260705.1"},
		{TagName: "v-custom.20260704.1"},
		{TagName: "v-custom.20260703.1"},
	}
	svc := newRollbackTestService("custom.20260708.1", releases)

	for _, target := range []string{
		"",                    // empty
		"custom.20260708.1",   // current version
		"v-custom.20260708.1", // current version with prefix
		"custom.20260709.1",   // newer than current
		"custom.20260703.1",   // older than the 3 most recent
		"9.9.9",               // nonexistent
	} {
		err := svc.RollbackToVersion(context.Background(), target)
		require.ErrorIs(t, err, ErrRollbackVersionNotAllowed, "target %q should be rejected", target)
	}
}

func TestUpdateServiceRollbackToVersionAcceptsVPrefix(t *testing.T) {
	// No platform asset in the release: the target passes the allowlist check
	// and fails later at asset lookup, proving the version itself was accepted.
	releases := []*GitHubRelease{
		{TagName: "v-custom.20260708.1"},
		{TagName: "v-custom.20260707.1"},
	}
	svc := newRollbackTestService("custom.20260708.1", releases)

	err := svc.RollbackToVersion(context.Background(), "v-custom.20260707.1")

	require.Error(t, err)
	require.NotErrorIs(t, err, ErrRollbackVersionNotAllowed)
	require.Contains(t, err.Error(), "no compatible release found")
}

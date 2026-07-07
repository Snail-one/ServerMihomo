package install

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"snailproxy/internal/domain/mihomo"
	"snailproxy/internal/infra/downloader"
	"snailproxy/internal/infra/github"
	"snailproxy/internal/infra/platform"
	"snailproxy/internal/terminal"
)

func TestBundledMihomoPackagePathMissingPackageErrorIsActionable(t *testing.T) {
	baseDir := t.TempDir()

	_, err := bundledMihomoPackagePath(baseDir)
	if err == nil {
		t.Fatal("bundledMihomoPackagePath() error = nil, want missing package error")
	}

	var missingPackage missingBundledMihomoPackageError
	if !errors.As(err, &missingPackage) {
		t.Fatalf("error = %T, want missingBundledMihomoPackageError", err)
	}
	for _, want := range []string{
		filepath.Join(baseDir, "packages", bundledMihomoPackagePattern()),
		"go generate ./internal/assets",
		"安装与更新 -> 在线下载并安装 mihomo",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("missing error should contain %q:\n%v", want, err)
		}
	}
}

func TestInstallBundledMihomoBinaryKeepsExistingBinaryWhenPackageMissing(t *testing.T) {
	baseDir := t.TempDir()
	binaryPath := filepath.Join(baseDir, mihomoBinaryName())
	if err := os.WriteFile(binaryPath, []byte("existing"), 0o755); err != nil {
		t.Fatalf("WriteFile(existing binary) error = %v", err)
	}

	installedPath, err := installBundledMihomoBinary(baseDir, true)
	if err != nil {
		t.Fatalf("installBundledMihomoBinary() error = %v", err)
	}
	if installedPath != "" {
		t.Fatalf("installedPath = %q, want empty because existing binary was kept", installedPath)
	}
}

func TestDownloadAndPrepareMihomoKeepsExistingBinaryWithoutVersionOrGitHub(t *testing.T) {
	binaryPath := installTestBinary(t)
	useInstalledBinaryPath(t, binaryPath)
	restoreInstallHooks(t)
	withInput(t, "n\n")

	var versionCalls int
	var fetchCalls int
	var cleanupCalls int
	printLocalMihomoVersionFunc = func(context.Context) {
		versionCalls++
	}
	fetchMihomoAssetsFunc = func(context.Context) ([]github.Asset, error) {
		fetchCalls++
		return nil, errors.New("unexpected GitHub fetch")
	}
	cleanupTemporaryFilesFunc = func() error {
		cleanupCalls++
		return nil
	}

	var err error
	output := captureStdout(t, func() {
		err = downloadAndPrepareMihomo(context.Background(), stubRuntime{})
	})
	if err != nil {
		t.Fatalf("downloadAndPrepareMihomo() error = %v", err)
	}
	if versionCalls != 0 {
		t.Fatalf("printLocalMihomoVersion calls = %d, want 0", versionCalls)
	}
	if fetchCalls != 0 {
		t.Fatalf("fetchMihomoAssets calls = %d, want 0", fetchCalls)
	}
	if cleanupCalls != 0 {
		t.Fatalf("cleanup calls = %d, want 0", cleanupCalls)
	}
	if !strings.Contains(output, "保留现有程序文件") {
		t.Fatalf("output should keep existing binary:\n%s", output)
	}
}

func TestDownloadAndPrepareMihomoOverwriteExistingBinaryContinuesToReleaseAndAssetSelection(t *testing.T) {
	binaryPath := installTestBinary(t)
	useInstalledBinaryPath(t, binaryPath)
	restoreInstallHooks(t)
	withInput(t, "y\n")

	stopAfterSelection := errors.New("stop after asset selection")
	var events []string
	printLocalMihomoVersionFunc = func(context.Context) {
		events = append(events, "version")
	}
	fetchMihomoAssetsFunc = func(context.Context) ([]github.Asset, error) {
		events = append(events, "fetch")
		return []github.Asset{{
			Name:               "mihomo-linux-amd64-v3-v1.0.0.gz",
			BrowserDownloadURL: "https://example.invalid/mihomo.gz",
		}}, nil
	}
	selectAssetFunc = func(assets []github.Asset) (github.Asset, error) {
		events = append(events, "select")
		if len(assets) != 1 {
			t.Fatalf("assets length = %d, want 1", len(assets))
		}
		return github.Asset{}, stopAfterSelection
	}
	cleanupTemporaryFilesFunc = func() error {
		events = append(events, "cleanup")
		return nil
	}

	err := downloadAndPrepareMihomo(context.Background(), stubRuntime{})
	if !errors.Is(err, stopAfterSelection) {
		t.Fatalf("downloadAndPrepareMihomo() error = %v, want %v", err, stopAfterSelection)
	}
	if got, want := strings.Join(events, ","), "version,fetch,select,cleanup"; got != want {
		t.Fatalf("events = %q, want %q", got, want)
	}
}

func TestDownloadAndPrepareMihomoCleansDownloadAndInstallTempFilesAfterInstall(t *testing.T) {
	tempRoot := t.TempDir()
	t.Setenv("TMPDIR", tempRoot)
	useInstalledBinaryPath(t, filepath.Join(tempRoot, "not-installed"))
	restoreInstallHooks(t)
	withInput(t, "n\n")

	asset := github.Asset{
		Name:               "mihomo-linux-amd64-v3-v1.0.0.gz",
		BrowserDownloadURL: "https://example.invalid/mihomo.gz",
	}
	downloadPath := downloader.AssetPath(asset)
	if err := os.MkdirAll(filepath.Dir(downloadPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(download dir) error = %v", err)
	}
	if err := os.WriteFile(downloadPath, []byte("existing archive"), 0o644); err != nil {
		t.Fatalf("WriteFile(download archive) error = %v", err)
	}
	installTempPath := filepath.Join(tempRoot, "mihomo-install", "mihomo")
	if err := os.MkdirAll(filepath.Dir(installTempPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(install temp dir) error = %v", err)
	}
	if err := os.WriteFile(installTempPath, []byte("extracted binary"), 0o755); err != nil {
		t.Fatalf("WriteFile(install temp binary) error = %v", err)
	}

	printLocalMihomoVersionFunc = func(context.Context) {}
	fetchMihomoAssetsFunc = func(context.Context) ([]github.Asset, error) {
		return []github.Asset{asset}, nil
	}
	selectAssetFunc = func(assets []github.Asset) (github.Asset, error) {
		return assets[0], nil
	}

	manager := stubPlatformManager{
		prepareBinary: func(_ context.Context, archivePath string, assetName string, overwrite bool) error {
			if archivePath != downloadPath {
				t.Fatalf("archivePath = %q, want %q", archivePath, downloadPath)
			}
			if assetName != asset.Name {
				t.Fatalf("assetName = %q, want %q", assetName, asset.Name)
			}
			if overwrite {
				t.Fatal("overwrite = true, want false")
			}
			return nil
		},
	}

	if err := downloadAndPrepareMihomo(context.Background(), stubRuntime{manager: manager}); err != nil {
		t.Fatalf("downloadAndPrepareMihomo() error = %v", err)
	}
	for _, path := range []string{
		filepath.Join(tempRoot, "mihomo"),
		filepath.Join(tempRoot, "mihomo-install"),
	} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("temp path %s exists after cleanup or stat failed with unexpected error: %v", path, err)
		}
	}
}

func installTestBinary(t *testing.T) string {
	t.Helper()

	binaryPath := filepath.Join(t.TempDir(), "mihomo")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("WriteFile(test binary) error = %v", err)
	}
	return binaryPath
}

func useInstalledBinaryPath(t *testing.T, path string) {
	t.Helper()

	originalPath := installedBinaryPath
	installedBinaryPath = path
	t.Cleanup(func() {
		installedBinaryPath = originalPath
	})
}

func restoreInstallHooks(t *testing.T) {
	t.Helper()

	originalPrintLocalMihomoVersionFunc := printLocalMihomoVersionFunc
	originalFetchMihomoAssetsFunc := fetchMihomoAssetsFunc
	originalSelectAssetFunc := selectAssetFunc
	originalCleanupTemporaryFilesFunc := cleanupTemporaryFilesFunc
	t.Cleanup(func() {
		printLocalMihomoVersionFunc = originalPrintLocalMihomoVersionFunc
		fetchMihomoAssetsFunc = originalFetchMihomoAssetsFunc
		selectAssetFunc = originalSelectAssetFunc
		cleanupTemporaryFilesFunc = originalCleanupTemporaryFilesFunc
	})
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = originalStdout
	}()

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("close stdout writer error = %v", err)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read stdout error = %v", err)
	}
	return string(data)
}

type stubRuntime struct {
	manager    platform.Manager
	managerErr error
}

func (stubRuntime) Terminal() terminal.Terminal {
	return nil
}

func (stubRuntime) NewMihomoStore() mihomo.Store {
	return mihomo.Store{}
}

func (r stubRuntime) NewPlatformManager() (platform.Manager, error) {
	if r.manager != nil || r.managerErr != nil {
		return r.manager, r.managerErr
	}
	return nil, errors.New("unexpected platform manager creation")
}

type stubPlatformManager struct {
	prepareBinary func(ctx context.Context, archivePath string, assetName string, overwrite bool) error
}

func (m stubPlatformManager) PrepareBinary(ctx context.Context, archivePath string, assetName string, overwrite bool) error {
	if m.prepareBinary != nil {
		return m.prepareBinary(ctx, archivePath, assetName, overwrite)
	}
	return nil
}

func (stubPlatformManager) InstallService(context.Context) error {
	return nil
}

func (stubPlatformManager) StartService(context.Context) error {
	return nil
}

func (stubPlatformManager) RestartService(context.Context) error {
	return nil
}

func (stubPlatformManager) StopService(context.Context) error {
	return nil
}

func (stubPlatformManager) WriteProxyEnvironment(context.Context) error {
	return nil
}

func (stubPlatformManager) ClearProxyEnvironment(context.Context) error {
	return nil
}

func (stubPlatformManager) Uninstall(context.Context) error {
	return nil
}

package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunVersionArgReturnsBeforeSudoOrMenu(t *testing.T) {
	for _, arg := range []string{"--version", "-v", "version"} {
		t.Run(arg, func(t *testing.T) {
			if err := Run(context.Background(), []string{arg}); err != nil {
				t.Fatalf("Run(%q) error = %v", arg, err)
			}
		})
	}
}

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
		"go generate ./resources",
		"安装 -> 在线安装",
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

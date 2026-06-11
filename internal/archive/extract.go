package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func ExtractMihomoBinary(archivePath, assetName, targetDir string) (string, error) {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", err
	}

	lower := strings.ToLower(assetName)
	switch {
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return extractTarGzip(archivePath, targetDir)
	case strings.HasSuffix(lower, ".gz"):
		return extractGzip(archivePath, targetDir)
	case strings.HasSuffix(lower, ".zip"):
		return extractZip(archivePath, targetDir)
	default:
		return copyPlainBinary(archivePath, targetDir)
	}
}

func extractGzip(archivePath, targetDir string) (string, error) {
	in, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer in.Close()

	gz, err := gzip.NewReader(in)
	if err != nil {
		return "", err
	}
	defer gz.Close()

	return writeExecutable(gz, filepath.Join(targetDir, binaryName()))
}

func extractTarGzip(archivePath, targetDir string) (string, error) {
	in, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer in.Close()

	gz, err := gzip.NewReader(in)
	if err != nil {
		return "", err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		if isMihomoBinary(header.Name) {
			return writeExecutable(tr, filepath.Join(targetDir, binaryName()))
		}
	}

	return "", fmt.Errorf("压缩包里没有找到 mihomo 可执行文件")
}

func extractZip(archivePath, targetDir string) (string, error) {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer zr.Close()

	for _, file := range zr.File {
		if file.FileInfo().IsDir() || !isMihomoBinary(file.Name) {
			continue
		}

		in, err := file.Open()
		if err != nil {
			return "", err
		}
		defer in.Close()

		return writeExecutable(in, filepath.Join(targetDir, binaryName()))
	}

	return "", fmt.Errorf("压缩包里没有找到 mihomo 可执行文件")
}

func copyPlainBinary(sourcePath, targetDir string) (string, error) {
	in, err := os.Open(sourcePath)
	if err != nil {
		return "", err
	}
	defer in.Close()

	return writeExecutable(in, filepath.Join(targetDir, binaryName()))
}

func writeExecutable(in io.Reader, targetPath string) (string, error) {
	out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return "", err
	}
	return targetPath, nil
}

func isMihomoBinary(name string) bool {
	base := strings.ToLower(filepath.Base(name))
	return base == "mihomo" || base == "mihomo.exe" || strings.HasPrefix(base, "mihomo-")
}

func binaryName() string {
	if runtime.GOOS == "windows" {
		return "mihomo.exe"
	}
	return "mihomo"
}

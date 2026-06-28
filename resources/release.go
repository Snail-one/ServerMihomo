package resources

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type ReleaseOptions struct {
	TargetDir string
	Overwrite bool
}

type ReleaseResult struct {
	TargetDir string
	Released  []string
	Skipped   []string
}

func ReleaseMihomoBundle(options ReleaseOptions) (ReleaseResult, error) {
	return releaseMihomoBundleFromFS(embeddedFiles, options)
}

func releaseMihomoBundleFromFS(source fs.FS, options ReleaseOptions) (ReleaseResult, error) {
	result := ReleaseResult{TargetDir: options.TargetDir}
	if strings.TrimSpace(options.TargetDir) == "" {
		return result, fmt.Errorf("释放目录不能为空")
	}

	if err := os.MkdirAll(options.TargetDir, 0o755); err != nil {
		return result, fmt.Errorf("创建 mihomo 目录失败: %w", err)
	}

	err := fs.WalkDir(source, mihomoBundleRoot, func(sourcePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if sourcePath == mihomoBundleRoot {
			return nil
		}
		switch path.Base(sourcePath) {
		case ".gitkeep", ".gitignore":
			return nil
		}

		relativePath := strings.TrimPrefix(sourcePath, mihomoBundleRoot+"/")
		if relativePath == sourcePath {
			return fmt.Errorf("资源路径不在 %s 目录下: %s", mihomoBundleRoot, sourcePath)
		}
		targetPath := filepath.Join(options.TargetDir, filepath.FromSlash(relativePath))

		if entry.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}
		if entry.Type()&fs.ModeType != 0 {
			return nil
		}
		if fileExists(targetPath) && !options.Overwrite {
			result.Skipped = append(result.Skipped, targetPath)
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("创建目录 %s 失败: %w", filepath.Dir(targetPath), err)
		}
		if err := copyEmbeddedFile(source, sourcePath, targetPath, releaseFileMode(relativePath)); err != nil {
			return err
		}

		result.Released = append(result.Released, targetPath)
		return nil
	})
	if err != nil {
		return result, fmt.Errorf("释放本地资源包失败: %w", err)
	}

	return result, nil
}

func copyEmbeddedFile(source fs.FS, sourcePath string, targetPath string, mode fs.FileMode) error {
	input, err := source.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("读取资源文件 %s 失败: %w", sourcePath, err)
	}
	defer input.Close()

	output, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("写入资源文件 %s 失败: %w", targetPath, err)
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		return fmt.Errorf("写入资源文件 %s 失败: %w", targetPath, copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("关闭资源文件 %s 失败: %w", targetPath, closeErr)
	}
	if err := os.Chmod(targetPath, mode); err != nil {
		return fmt.Errorf("设置资源文件权限 %s 失败: %w", targetPath, err)
	}
	return nil
}

func releaseFileMode(relativePath string) fs.FileMode {
	base := path.Base(relativePath)
	switch {
	case base == "mihomo":
		return 0o770
	case strings.HasSuffix(base, ".sh"):
		return 0o755
	default:
		return 0o644
	}
}

func fileExists(targetPath string) bool {
	info, err := os.Stat(targetPath)
	return err == nil && !info.IsDir()
}

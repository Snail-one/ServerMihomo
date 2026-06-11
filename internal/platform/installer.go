package platform

import "context"

type Installer interface {
	PrepareBinary(ctx context.Context, archivePath string, assetName string, overwrite bool) error
	InstallService(ctx context.Context) error
}

type Uninstaller interface {
	Uninstall(ctx context.Context) error
}

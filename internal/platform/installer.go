package platform

import "context"

type Installer interface {
	PrepareBinary(ctx context.Context, archivePath string, assetName string, overwrite bool) error
	InstallService(ctx context.Context) error
	StartService(ctx context.Context) error
	RestartService(ctx context.Context) error
	StopService(ctx context.Context) error
	WriteProxyEnvironment(ctx context.Context) error
	ClearProxyEnvironment(ctx context.Context) error
}

package platform

import "context"

type Installer interface {
	Install(ctx context.Context, archivePath string, assetName string) error
}

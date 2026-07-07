package assets

import "embed"

//go:generate go run ../../cmd/resourcegen -target mihomo

//go:embed all:mihomo
var embeddedFiles embed.FS

const mihomoBundleRoot = "mihomo"

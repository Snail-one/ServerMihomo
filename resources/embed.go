package resources

import "embed"

//go:generate go run ../cmd/resourcegen

//go:embed all:mihomo
var embeddedFiles embed.FS

const mihomoBundleRoot = "mihomo"

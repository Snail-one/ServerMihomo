package resources

import "embed"

//go:generate go run ./download.go

//go:embed all:mihomo
var embeddedFiles embed.FS

const mihomoBundleRoot = "mihomo"

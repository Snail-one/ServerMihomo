//go:build linux

package platform

const (
	installDir      = "/opt/mihomo"
	installedBinary = "/opt/mihomo/mihomo"
	serviceFile     = "/etc/systemd/system/mihomo.service"
	serviceName     = "mihomo.service"
)

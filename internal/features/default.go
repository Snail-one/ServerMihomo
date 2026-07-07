package features

import (
	"snailproxy/internal/feature"
	installfeature "snailproxy/internal/features/install"
	servicefeature "snailproxy/internal/features/service"
	subscriptionfeature "snailproxy/internal/features/subscription"
	uninstallfeature "snailproxy/internal/features/uninstall"
)

func Default() feature.Registry {
	return feature.MustRegistry(
		installfeature.Feature{},
		subscriptionfeature.Feature{},
		servicefeature.Feature{},
		uninstallfeature.Feature{},
	)
}

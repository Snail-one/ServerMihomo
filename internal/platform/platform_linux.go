//go:build linux

package platform

func NewManager() (Manager, error) {
	return &linuxManager{}, nil
}

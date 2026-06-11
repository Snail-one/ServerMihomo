//go:build !linux

package platform

func RequireSudo() error {
	return nil
}

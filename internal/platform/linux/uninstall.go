//go:build linux

package linux

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

func (m *Manager) Uninstall(ctx context.Context) error {
	runCommandAllowFailure(ctx, "systemctl", "stop", serviceName)
	runCommandAllowFailure(ctx, "systemctl", "disable", serviceName)

	if err := runCommand(ctx, "rm", "-f", serviceFile); err != nil {
		return err
	}
	if err := runCommand(ctx, "systemctl", "daemon-reload"); err != nil {
		return err
	}
	runCommandAllowFailure(ctx, "systemctl", "reset-failed", serviceName)

	if err := runCommand(ctx, "rm", "-rf", installDir); err != nil {
		return err
	}
	if err := runCommand(
		ctx,
		"rm",
		"-rf",
		filepath.Join(os.TempDir(), "mihomo"),
		filepath.Join(os.TempDir(), "mihomo-install"),
	); err != nil {
		return err
	}
	runCommandAllowFailure(ctx, "userdel", "mihomo")

	fmt.Println("mihomo 已卸载并清理完成。")
	return nil
}

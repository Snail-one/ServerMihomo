//go:build linux

package platform

import (
	"fmt"
	"os"
)

func RequireSudo() error {
	if os.Geteuid() == 0 {
		return nil
	}

	executable, err := os.Executable()
	if err != nil || executable == "" {
		executable = "snailproxy"
	}

	return fmt.Errorf("请使用 sudo 运行本程序: sudo %s", executable)
}

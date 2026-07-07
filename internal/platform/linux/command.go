//go:build linux

package linux

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
)

func runCommand(ctx context.Context, name string, args ...string) error {
	command := append([]string{name}, args...)

	fmt.Printf("\n执行命令: %s\n", joinCommand(command))
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Stdin = os.Stdin

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	err := cmd.Run()
	fmt.Print("命令输出:\n")
	if output.Len() == 0 {
		fmt.Println("(无输出)")
	} else {
		fmt.Print(output.String())
	}

	if err != nil {
		return fmt.Errorf("命令执行失败 %q: %w", joinCommand(command), err)
	}
	return nil
}

func runCommandAllowFailure(ctx context.Context, name string, args ...string) {
	if err := runCommand(ctx, name, args...); err != nil {
		fmt.Printf("命令失败，继续执行卸载清理: %v\n", err)
	}
}

func joinCommand(parts []string) string {
	var buf bytes.Buffer
	for i, part := range parts {
		if i > 0 {
			buf.WriteByte(' ')
		}
		buf.WriteString(part)
	}
	return buf.String()
}

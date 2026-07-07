package app

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"snailproxy/internal/feature"
	"snailproxy/internal/features"
	"snailproxy/internal/terminal"
)

func TestRunVersionArgReturnsBeforeSudoOrMenu(t *testing.T) {
	for _, arg := range []string{"--version", "-v", "version"} {
		t.Run(arg, func(t *testing.T) {
			if err := Run(context.Background(), []string{arg}, feature.Registry{}); err != nil {
				t.Fatalf("Run(%q) error = %v", arg, err)
			}
		})
	}
}

func TestSelectMainMenuUsesDefaultRegistryOrder(t *testing.T) {
	tests := []struct {
		input  string
		wantID string
	}{
		{input: "1\n", wantID: "install"},
		{input: "2\n", wantID: "subscription"},
		{input: "3\n", wantID: "service"},
		{input: "4\n", wantID: "uninstall"},
	}

	for _, tt := range tests {
		t.Run(strings.TrimSpace(tt.input), func(t *testing.T) {
			withInput(t, tt.input)

			action, err := selectMainMenu(features.Default())
			if err != nil {
				t.Fatalf("selectMainMenu() error = %v", err)
			}
			if action.exit {
				t.Fatal("action.exit = true, want feature")
			}
			if action.feature.ID() != tt.wantID {
				t.Fatalf("feature ID = %q, want %q", action.feature.ID(), tt.wantID)
			}
		})
	}

	withInput(t, "0\n")
	action, err := selectMainMenu(features.Default())
	if err != nil {
		t.Fatalf("selectMainMenu() exit error = %v", err)
	}
	if !action.exit {
		t.Fatal("action.exit = false, want true")
	}
}

func TestSelectMainMenuIsGeneratedFromRegistry(t *testing.T) {
	registry := feature.MustRegistry(
		testFeature{id: "install", label: "安装与更新", order: 10},
		testFeature{id: "service", label: "mihomo 服务与代理", order: 30},
	)
	withInput(t, "2\n")

	var action mainMenuAction
	var err error
	output := captureStdout(t, func() {
		action, err = selectMainMenu(registry)
	})
	if err != nil {
		t.Fatalf("selectMainMenu() error = %v", err)
	}
	if action.feature.ID() != "service" {
		t.Fatalf("feature ID = %q, want service", action.feature.ID())
	}
	if strings.Contains(output, "订阅管理") {
		t.Fatalf("output contains unregistered feature:\n%s", output)
	}
	if !strings.Contains(output, "请输入操作编号 [0-2]:") {
		t.Fatalf("output prompt range should follow registry size:\n%s", output)
	}
}

type testFeature struct {
	id    string
	label string
	order int
}

func (f testFeature) ID() string {
	return f.id
}

func (f testFeature) Label() string {
	return f.label
}

func (f testFeature) Order() int {
	return f.order
}

func (f testFeature) Run(context.Context, feature.Runtime) error {
	return nil
}

func withInput(t *testing.T, input string) {
	t.Helper()
	restore := terminal.SetInput(strings.NewReader(input))
	t.Cleanup(restore)
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = originalStdout
	}()

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("close stdout writer error = %v", err)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read stdout error = %v", err)
	}
	return string(data)
}

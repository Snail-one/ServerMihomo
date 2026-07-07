package ui

import (
	"bufio"
	"io"
	"os"
	"strings"
	"testing"
)

func TestSelectMenuReprintsAfterEmptyAndInvalidInput(t *testing.T) {
	withInput(t, "\n9\n1\n")

	var action int
	var err error
	output := captureStdout(t, func() {
		action, err = SelectMenu("测试菜单:", "[0-1]", []MenuOption[int]{
			{Number: 1, Label: "继续", Value: 42},
			{Number: 0, Label: "返回", Value: 0},
		})
	})
	if err != nil {
		t.Fatalf("SelectMenu() error = %v", err)
	}
	if action != 42 {
		t.Fatalf("action = %d, want 42", action)
	}
	if count := strings.Count(output, "测试菜单:"); count != 3 {
		t.Fatalf("menu prints = %d, want 3\n%s", count, output)
	}
	for _, want := range []string{"输入不能为空，请输入菜单编号。", "输入无效，请重新输入。"} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestSelectMainActionMapping(t *testing.T) {
	tests := []struct {
		input string
		want  Action
	}{
		{input: "1\n", want: ActionInstall},
		{input: "2\n", want: ActionManageSubscription},
		{input: "3\n", want: ActionManageMihomoService},
		{input: "4\n", want: ActionUninstall},
		{input: "0\n", want: ActionExit},
	}

	for _, tt := range tests {
		t.Run(strings.TrimSpace(tt.input), func(t *testing.T) {
			withInput(t, tt.input)

			action, err := SelectMainAction()
			if err != nil {
				t.Fatalf("SelectMainAction() error = %v", err)
			}
			if action != tt.want {
				t.Fatalf("action = %d, want %d", action, tt.want)
			}
		})
	}
}

func TestSelectInstallActionMapping(t *testing.T) {
	tests := []struct {
		input string
		want  InstallAction
	}{
		{input: "1\n", want: InstallLocal},
		{input: "2\n", want: InstallOnline},
		{input: "3\n", want: InstallService},
		{input: "0\n", want: InstallReturn},
	}

	for _, tt := range tests {
		t.Run(strings.TrimSpace(tt.input), func(t *testing.T) {
			withInput(t, tt.input)

			action, err := SelectInstallAction()
			if err != nil {
				t.Fatalf("SelectInstallAction() error = %v", err)
			}
			if action != tt.want {
				t.Fatalf("action = %d, want %d", action, tt.want)
			}
		})
	}
}

func TestSelectServiceActionMapping(t *testing.T) {
	tests := []struct {
		input string
		want  ServiceAction
	}{
		{input: "1\n", want: ServiceStart},
		{input: "2\n", want: ServiceRestart},
		{input: "3\n", want: ServiceStop},
		{input: "4\n", want: ServiceWriteProxyEnv},
		{input: "5\n", want: ServiceClearProxyEnv},
		{input: "0\n", want: ServiceReturn},
	}

	for _, tt := range tests {
		t.Run(strings.TrimSpace(tt.input), func(t *testing.T) {
			withInput(t, tt.input)

			action, err := SelectServiceAction()
			if err != nil {
				t.Fatalf("SelectServiceAction() error = %v", err)
			}
			if action != tt.want {
				t.Fatalf("action = %d, want %d", action, tt.want)
			}
		})
	}
}

func TestSelectSubscriptionActionWithoutSubscriptionsOnlyAllowsCreateAndReturn(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  SubscriptionAction
	}{
		{name: "create", input: "1\n", want: SubscriptionCreate},
		{name: "return", input: "0\n", want: SubscriptionReturn},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withInput(t, tt.input)

			action, err := SelectSubscriptionAction(nil)
			if err != nil {
				t.Fatalf("SelectSubscriptionAction() error = %v", err)
			}
			if action != tt.want {
				t.Fatalf("action = %d, want %d", action, tt.want)
			}
		})
	}

	withInput(t, "2\n0\n")
	output := captureStdout(t, func() {
		action, err := SelectSubscriptionAction(nil)
		if err != nil {
			t.Fatalf("SelectSubscriptionAction() error = %v", err)
		}
		if action != SubscriptionReturn {
			t.Fatalf("action = %d, want SubscriptionReturn", action)
		}
	})
	if !strings.Contains(output, "输入无效，请重新输入。") {
		t.Fatalf("output should reject unavailable existing-subscription actions:\n%s", output)
	}
}

func TestSelectSubscriptionActionWithSubscriptionsMapping(t *testing.T) {
	tests := []struct {
		input string
		want  SubscriptionAction
	}{
		{input: "1\n", want: SubscriptionCreate},
		{input: "2\n", want: SubscriptionUpdate},
		{input: "3\n", want: SubscriptionModify},
		{input: "4\n", want: SubscriptionDelete},
		{input: "5\n", want: SubscriptionApply},
		{input: "0\n", want: SubscriptionReturn},
	}

	for _, tt := range tests {
		t.Run(strings.TrimSpace(tt.input), func(t *testing.T) {
			withInput(t, tt.input)

			action, err := SelectSubscriptionAction([]string{"测试订阅（subscription.yaml）"})
			if err != nil {
				t.Fatalf("SelectSubscriptionAction() error = %v", err)
			}
			if action != tt.want {
				t.Fatalf("action = %d, want %d", action, tt.want)
			}
		})
	}
}

func TestSelectSubscriptionTargetMapping(t *testing.T) {
	withInput(t, "2\n")

	index, err := SelectSubscription([]string{"订阅 1（one.yaml）", "订阅 2（two.yaml）"})
	if err != nil {
		t.Fatalf("SelectSubscription() error = %v", err)
	}
	if index != 1 {
		t.Fatalf("index = %d, want 1", index)
	}
}

func TestPromptSubscriptionURLDefaultUsesDefaultForEmptyInput(t *testing.T) {
	withInput(t, "\n")

	url, err := PromptSubscriptionURLDefault("https://example.com/sub")
	if err != nil {
		t.Fatalf("PromptSubscriptionURLDefault() error = %v", err)
	}
	if url != "https://example.com/sub" {
		t.Fatalf("url = %q, want default", url)
	}
}

func withInput(t *testing.T, input string) {
	t.Helper()
	originalReader := stdinReader
	stdinReader = bufio.NewReader(strings.NewReader(input))
	t.Cleanup(func() {
		stdinReader = originalReader
	})
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

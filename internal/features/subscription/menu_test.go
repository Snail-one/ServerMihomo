package subscription

import (
	"io"
	"os"
	"strings"
	"testing"

	"snailproxy/internal/domain/mihomo"
	"snailproxy/internal/terminal"
)

func TestSelectWithoutSubscriptionsOnlyAllowsCreateAndReturn(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Action
	}{
		{name: "create", input: "1\n", want: ActionCreate},
		{name: "return", input: "0\n", want: ActionReturn},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withInput(t, tt.input)

			action, err := Select(nil)
			if err != nil {
				t.Fatalf("Select() error = %v", err)
			}
			if action != tt.want {
				t.Fatalf("action = %d, want %d", action, tt.want)
			}
		})
	}

	withInput(t, "2\n0\n")
	output := captureStdout(t, func() {
		action, err := Select(nil)
		if err != nil {
			t.Fatalf("Select() error = %v", err)
		}
		if action != ActionReturn {
			t.Fatalf("action = %d, want ActionReturn", action)
		}
	})
	if !strings.Contains(output, "输入无效，请重新输入。") {
		t.Fatalf("output should reject unavailable existing-subscription actions:\n%s", output)
	}
}

func TestSelectWithSubscriptionsMapping(t *testing.T) {
	tests := []struct {
		input string
		want  Action
	}{
		{input: "1\n", want: ActionCreate},
		{input: "2\n", want: ActionUpdate},
		{input: "3\n", want: ActionModify},
		{input: "4\n", want: ActionDelete},
		{input: "5\n", want: ActionApply},
		{input: "0\n", want: ActionReturn},
	}

	for _, tt := range tests {
		t.Run(strings.TrimSpace(tt.input), func(t *testing.T) {
			withInput(t, tt.input)

			action, err := Select([]string{"测试订阅（subscription.yaml）"})
			if err != nil {
				t.Fatalf("Select() error = %v", err)
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

func TestSelectSubscriptionTargetReturnStaysInSubscriptionMenu(t *testing.T) {
	withInput(t, "0\n")

	index, ok, err := selectSubscriptionTarget("更新", []mihomo.Subscription{
		{Name: "订阅 1", File: "one.yaml"},
	})
	if err != nil {
		t.Fatalf("selectSubscriptionTarget() error = %v, want nil", err)
	}
	if ok {
		t.Fatal("ok = true, want false")
	}
	if index != -1 {
		t.Fatalf("index = %d, want -1", index)
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

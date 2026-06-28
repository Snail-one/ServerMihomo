package ui

import (
	"bufio"
	"strings"
	"testing"
)

func TestSelectActionReprintsMenuAfterInvalidInput(t *testing.T) {
	originalReader := stdinReader
	stdinReader = bufio.NewReader(strings.NewReader("\n9\n0\n"))
	t.Cleanup(func() {
		stdinReader = originalReader
	})

	menuPrints := 0
	action, err := selectAction("[0-1]", func() {
		menuPrints++
	}, map[string]Action{
		"0": ActionExit,
		"1": ActionDownload,
	})
	if err != nil {
		t.Fatalf("selectAction() error = %v", err)
	}
	if action != ActionExit {
		t.Fatalf("action = %d, want ActionExit", action)
	}
	if menuPrints != 3 {
		t.Fatalf("menuPrints = %d, want 3", menuPrints)
	}
}

func TestSelectSubscriptionDownloadTargetSupportsModifyOption(t *testing.T) {
	originalReader := stdinReader
	stdinReader = bufio.NewReader(strings.NewReader("3\n"))
	t.Cleanup(func() {
		stdinReader = originalReader
	})

	index, action, err := SelectSubscriptionDownloadTarget([]string{"测试订阅（subscription.yaml）"})
	if err != nil {
		t.Fatalf("SelectSubscriptionDownloadTarget() error = %v", err)
	}
	if index != -1 {
		t.Fatalf("index = %d, want -1", index)
	}
	if action != SubscriptionDownloadModify {
		t.Fatalf("action = %d, want SubscriptionDownloadModify", action)
	}
}

func TestPromptSubscriptionURLDefaultUsesDefaultForEmptyInput(t *testing.T) {
	originalReader := stdinReader
	stdinReader = bufio.NewReader(strings.NewReader("\n"))
	t.Cleanup(func() {
		stdinReader = originalReader
	})

	url, err := PromptSubscriptionURLDefault("https://example.com/sub")
	if err != nil {
		t.Fatalf("PromptSubscriptionURLDefault() error = %v", err)
	}
	if url != "https://example.com/sub" {
		t.Fatalf("url = %q, want default", url)
	}
}

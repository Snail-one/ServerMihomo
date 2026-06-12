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

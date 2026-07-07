package app

import (
	"context"
	"testing"
)

func TestRunVersionArgReturnsBeforeSudoOrMenu(t *testing.T) {
	for _, arg := range []string{"--version", "-v", "version"} {
		t.Run(arg, func(t *testing.T) {
			if err := Run(context.Background(), []string{arg}); err != nil {
				t.Fatalf("Run(%q) error = %v", arg, err)
			}
		})
	}
}

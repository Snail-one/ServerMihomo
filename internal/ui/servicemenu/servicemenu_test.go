package servicemenu

import (
	"strings"
	"testing"

	"snailproxy/internal/ui/menu"
)

func TestSelectMapping(t *testing.T) {
	tests := []struct {
		input string
		want  Action
	}{
		{input: "1\n", want: ActionStart},
		{input: "2\n", want: ActionRestart},
		{input: "3\n", want: ActionStop},
		{input: "4\n", want: ActionWriteProxyEnv},
		{input: "5\n", want: ActionClearProxyEnv},
		{input: "0\n", want: ActionReturn},
	}

	for _, tt := range tests {
		t.Run(strings.TrimSpace(tt.input), func(t *testing.T) {
			withInput(t, tt.input)

			action, err := Select()
			if err != nil {
				t.Fatalf("Select() error = %v", err)
			}
			if action != tt.want {
				t.Fatalf("action = %d, want %d", action, tt.want)
			}
		})
	}
}

func withInput(t *testing.T, input string) {
	t.Helper()
	restore := menu.SetInput(strings.NewReader(input))
	t.Cleanup(restore)
}

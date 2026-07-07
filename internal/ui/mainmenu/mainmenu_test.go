package mainmenu

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
		{input: "1\n", want: ActionInstall},
		{input: "2\n", want: ActionManageSubscription},
		{input: "3\n", want: ActionManageMihomoService},
		{input: "4\n", want: ActionUninstall},
		{input: "0\n", want: ActionExit},
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

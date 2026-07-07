package mihomo

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopySubscriptionToFinalConfigPreservesContent(t *testing.T) {
	store := Store{BaseDir: t.TempDir()}
	if err := store.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs() error = %v", err)
	}

	subscriptionFile := "subscription-test.yaml"
	subscriptionContent := `mixed-port: 7890
proxies:
  - {name: Node A, server: example.com, port: 443, type: ss, cipher: chacha20-ietf-poly1305, password: secret}
proxy-groups:
  - name: Proxies
    type: select
    proxies:
      - Node A
rules:
  - MATCH,Proxies
`
	writeTestFile(t, store.ProfilePath(subscriptionFile), subscriptionContent)

	finalPath, err := store.CopySubscriptionToFinalConfig(Subscription{
		Name: "测试订阅",
		File: subscriptionFile,
		URL:  "https://example.com/sub",
	})
	if err != nil {
		t.Fatalf("CopySubscriptionToFinalConfig() error = %v", err)
	}
	if finalPath != store.FinalConfigPath() {
		t.Fatalf("finalPath = %q, want %q", finalPath, store.FinalConfigPath())
	}
	assertTestFileContent(t, store.FinalConfigPath(), subscriptionContent)
	assertTestFileContent(t, store.ProfilePath(subscriptionFile), subscriptionContent)
}

func TestCopySubscriptionToFinalConfigRejectsInvalidYAMLWithoutReplacingExistingConfig(t *testing.T) {
	store := Store{BaseDir: t.TempDir()}
	if err := store.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs() error = %v", err)
	}

	subscriptionFile := "subscription-test.yaml"
	writeTestFile(t, store.ProfilePath(subscriptionFile), "proxies:\n  - type: ss\n")
	writeTestFile(t, store.FinalConfigPath(), "existing\n")

	_, err := store.CopySubscriptionToFinalConfig(Subscription{
		Name: "测试订阅",
		File: subscriptionFile,
		URL:  "https://example.com/sub",
	})
	if err == nil || !strings.Contains(err.Error(), "没有可用 name") {
		t.Fatalf("CopySubscriptionToFinalConfig() error = %v, want missing name error", err)
	}
	assertTestFileContent(t, store.FinalConfigPath(), "existing\n")
}

func TestValidateSubscriptionDataRequiresValidYAMLWithNamedProxies(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name: "valid block",
			content: `proxies:
  - name: Node A
    type: ss
    server: example.com
    port: 443
`,
		},
		{
			name: "valid inline",
			content: `proxies:
  - {name: Node A, server: example.com, port: 443, type: ss, cipher: chacha20-ietf-poly1305, password: secret}
`,
		},
		{
			name: "invalid yaml",
			content: `proxies:
  - name: Node A
    port: [`,
			wantErr: "订阅 YAML 无效",
		},
		{
			name: "missing proxies",
			content: `proxy-groups:
  - name: Proxies
`,
			wantErr: "缺少 proxies",
		},
		{
			name: "proxies is not list",
			content: `proxies:
  name: Node A
`,
			wantErr: "proxies 必须是列表",
		},
		{
			name: "empty proxies",
			content: `proxies: []
`,
			wantErr: "proxies 不能为空",
		},
		{
			name: "proxy without name",
			content: `proxies:
  - type: ss
    server: example.com
    port: 443
`,
			wantErr: "没有可用 name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSubscriptionData([]byte(tt.content))
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("ValidateSubscriptionData() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ValidateSubscriptionData() error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestDownloadSubscriptionRejectsInvalidYAMLWithoutReplacingExistingFile(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "subscription.yaml")
	if err := os.WriteFile(targetPath, []byte("existing\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(existing) error = %v", err)
	}

	body := "not: [valid"
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode:    http.StatusOK,
				Status:        "200 OK",
				Body:          io.NopCloser(strings.NewReader(body)),
				ContentLength: int64(len(body)),
				Header:        make(http.Header),
				Request:       req,
			}, nil
		}),
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com/sub", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext() error = %v", err)
	}

	err = downloadSubscriptionWithClient(client, req, targetPath)
	if err == nil || !strings.Contains(err.Error(), "订阅 YAML 无效") {
		t.Fatalf("downloadSubscriptionWithClient() error = %v, want invalid YAML error", err)
	}

	assertTestFileContent(t, targetPath, "existing\n")
	if _, err := os.Stat(targetPath + ".download"); !os.IsNotExist(err) {
		t.Fatalf("temporary download file should be removed, stat err = %v", err)
	}
}

func TestDownloadSubscriptionRejectsHTTPErrorWithoutReplacingExistingFile(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "subscription.yaml")
	if err := os.WriteFile(targetPath, []byte("existing\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(existing) error = %v", err)
	}

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadGateway,
				Status:     "502 Bad Gateway",
				Body:       io.NopCloser(strings.NewReader("bad gateway")),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		}),
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com/sub", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext() error = %v", err)
	}

	err = downloadSubscriptionWithClient(client, req, targetPath)
	if err == nil || !strings.Contains(err.Error(), "502 Bad Gateway") {
		t.Fatalf("downloadSubscriptionWithClient() error = %v, want HTTP status error", err)
	}
	assertTestFileContent(t, targetPath, "existing\n")
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func writeTestFile(t *testing.T, targetPath string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(targetPath), err)
	}
	if err := os.WriteFile(targetPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", targetPath, err)
	}
}

func assertTestFileContent(t *testing.T, targetPath string, want string) {
	t.Helper()
	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", targetPath, err)
	}
	if string(data) != want {
		t.Fatalf("ReadFile(%s) = %q, want %q", targetPath, data, want)
	}
}

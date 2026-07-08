//go:build linux

package platform

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseTopLevelInt(t *testing.T) {
	data := []byte(`mode: rule
mixed-port: 57913 # local proxy
tun:
  mixed-port: 1
`)

	port, ok := parseTopLevelInt(data, "mixed-port")
	if !ok {
		t.Fatal("parseTopLevelInt() did not find mixed-port")
	}
	if port != 57913 {
		t.Fatalf("port = %d, want 57913", port)
	}
}

func TestProxyEnvironmentContent(t *testing.T) {
	content := proxyEnvironmentContent(proxyEnvironmentSettingsFromMixedPort(57913))
	want := `# Managed by snailproxy. Use snailproxy to clear this file.
export http_proxy="http://127.0.0.1:57913"
export HTTP_PROXY="http://127.0.0.1:57913"
export https_proxy="http://127.0.0.1:57913"
export HTTPS_PROXY="http://127.0.0.1:57913"
export no_proxy="` + defaultNoProxy + `"
export NO_PROXY="` + defaultNoProxy + `"
`
	if content != want {
		t.Fatalf("proxyEnvironmentContent() = %q, want %q", content, want)
	}
}

func TestProxyEnvironmentSettingsFromConfigPrefersMixedPort(t *testing.T) {
	settings, ok := proxyEnvironmentSettingsFromConfig([]byte(`port: 7890
socks-port: 7891
mixed-port: 57913
`))
	if !ok {
		t.Fatal("proxyEnvironmentSettingsFromConfig() did not find proxy settings")
	}
	if settings.HTTPProxy != "http://127.0.0.1:57913" {
		t.Fatalf("HTTPProxy = %q, want mixed-port HTTP proxy", settings.HTTPProxy)
	}
}

func TestProxyEnvironmentSettingsFromConfigSupportsLegacyPorts(t *testing.T) {
	settings, ok := proxyEnvironmentSettingsFromConfig([]byte(`port: 7890
socks-port: 7891
`))
	if !ok {
		t.Fatal("proxyEnvironmentSettingsFromConfig() did not find proxy settings")
	}
	if settings.HTTPProxy != "http://127.0.0.1:7890" {
		t.Fatalf("HTTPProxy = %q, want port HTTP proxy", settings.HTTPProxy)
	}
}

func TestAppendProxyEnvironmentBlockToBashrcAppendsAtBottom(t *testing.T) {
	block := proxyEnvironmentBashrcBlock(proxyEnvironmentSettings{
		HTTPProxy: "http://127.0.0.1:57913",
	})
	content := appendProxyEnvironmentBlockToBashrc("export PATH=$PATH\n", block)

	if !strings.HasPrefix(content, "export PATH=$PATH\n\n") {
		t.Fatalf("content should preserve existing bashrc content before block:\n%s", content)
	}
	if !strings.HasSuffix(content, block) {
		t.Fatalf("content should append proxy block at bottom:\n%s", content)
	}
}

func TestAppendProxyEnvironmentBlockToBashrcReplacesExistingBlock(t *testing.T) {
	oldBlock := proxyEnvironmentBashrcBlock(proxyEnvironmentSettings{
		HTTPProxy: "http://127.0.0.1:7890",
	})
	newBlock := proxyEnvironmentBashrcBlock(proxyEnvironmentSettingsFromMixedPort(57913))
	content := appendProxyEnvironmentBlockToBashrc("alias ll='ls -l'\n\n"+oldBlock, newBlock)

	if strings.Contains(content, "7890") {
		t.Fatalf("content still contains old proxy block:\n%s", content)
	}
	if strings.Count(content, proxyEnvironmentBlockBegin) != 1 {
		t.Fatalf("content should contain exactly one proxy block:\n%s", content)
	}
	if !strings.HasSuffix(content, newBlock) {
		t.Fatalf("content should end with new proxy block:\n%s", content)
	}
}

func TestRemoveProxyEnvironmentBlockFromBashrc(t *testing.T) {
	block := proxyEnvironmentBashrcBlock(proxyEnvironmentSettingsFromMixedPort(57913))
	content, removed := removeProxyEnvironmentBlockFromBashrc("before\n\n" + block + "\nafter\n")

	if !removed {
		t.Fatal("removeProxyEnvironmentBlockFromBashrc() did not remove block")
	}
	if strings.Contains(content, proxyEnvironmentBlockBegin) || strings.Contains(content, "57913") {
		t.Fatalf("content still contains proxy block:\n%s", content)
	}
	if content != "before\n\nafter\n" {
		t.Fatalf("content = %q, want existing bashrc content joined around removed block", content)
	}
}

func TestUnmanagedProxyEnvironmentLinesFindsManualProxyConfig(t *testing.T) {
	content := `# existing shell config
export http_proxy="http://192.168.31.108:52013"
HTTPS_PROXY=http://192.168.31.108:52013
declare -x no_proxy="localhost,127.0.0.1"
`
	got := unmanagedProxyEnvironmentLines(content)
	want := []string{
		`export http_proxy="http://192.168.31.108:52013"`,
		`HTTPS_PROXY=http://192.168.31.108:52013`,
		`declare -x no_proxy="localhost,127.0.0.1"`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unmanagedProxyEnvironmentLines() = %#v, want %#v", got, want)
	}
}

func TestUnmanagedProxyEnvironmentLinesIgnoresSnailproxyManagedBlock(t *testing.T) {
	managedBlock := proxyEnvironmentBashrcBlock(proxyEnvironmentSettingsFromMixedPort(57913))
	content := "before\n\n" + managedBlock + "\nafter\n"
	got := unmanagedProxyEnvironmentLines(content)
	if len(got) != 0 {
		t.Fatalf("unmanagedProxyEnvironmentLines() = %#v, want empty", got)
	}
}

func TestProxyEnvironmentTargetFromEnvironmentUsesSudoUser(t *testing.T) {
	target, err := proxyEnvironmentTargetFromEnvironment(
		func(key string) string {
			if key == "SUDO_USER" {
				return "alice"
			}
			return ""
		},
		func(username string) (systemUser, error) {
			if username != "alice" {
				t.Fatalf("lookup username = %q, want alice", username)
			}
			return systemUser{
				HomeDir: "/home/alice",
				UID:     "1000",
				GID:     "1000",
			}, nil
		},
		func() (systemUser, error) {
			t.Fatal("current user should not be used when SUDO_USER is set")
			return systemUser{}, nil
		},
	)
	if err != nil {
		t.Fatalf("proxyEnvironmentTargetFromEnvironment() error = %v", err)
	}
	if target.Path != "/home/alice/.bashrc" {
		t.Fatalf("Path = %q, want sudo user's .bashrc path", target.Path)
	}
	if target.UID != 1000 || target.GID != 1000 {
		t.Fatalf("UID/GID = %d/%d, want 1000/1000", target.UID, target.GID)
	}
}

func TestProxyEnvironmentTargetFromEnvironmentFallsBackToCurrentUser(t *testing.T) {
	for _, sudoUser := range []string{"", "root"} {
		target, err := proxyEnvironmentTargetFromEnvironment(
			func(key string) string {
				if key == "SUDO_USER" {
					return sudoUser
				}
				return ""
			},
			func(username string) (systemUser, error) {
				t.Fatalf("lookup should not be called for SUDO_USER=%q", sudoUser)
				return systemUser{}, nil
			},
			func() (systemUser, error) {
				return systemUser{
					HomeDir: "/root",
					UID:     "0",
					GID:     "0",
				}, nil
			},
		)
		if err != nil {
			t.Fatalf("proxyEnvironmentTargetFromEnvironment(%q) error = %v", sudoUser, err)
		}
		if target.Path != "/root/.bashrc" {
			t.Fatalf("Path = %q, want current user's .bashrc path", target.Path)
		}
	}
}

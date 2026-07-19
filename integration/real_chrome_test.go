//go:build integration

package integration

import (
	"bufio"
	"context"
	"encoding/json"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/masahide/tabcli/internal/buildinfo"
	"github.com/masahide/tabcli/internal/discovery"
	"github.com/masahide/tabcli/internal/install"
	"github.com/masahide/tabcli/internal/mcpclient"
	"github.com/masahide/tabcli/internal/tools"
)

func TestRealChromeNativeHTTPCLIAndStdioMCP(t *testing.T) {
	if os.Getenv("CHROME_REAL_INTEGRATION") != "1" {
		t.Skip("set CHROME_REAL_INTEGRATION=1")
	}
	if runtime.GOOS != "darwin" {
		t.Skip("MVP integration target is macOS")
	}
	binary := requirePath(t, "TABCLI_INTEGRATION_BINARY")
	extension := requirePath(t, "TABCLI_INTEGRATION_EXTENSION")
	chrome := os.Getenv("CHROME_INTEGRATION_EXECUTABLE")
	if chrome == "" {
		chrome = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
	}
	if _, err := os.Stat(chrome); err != nil {
		t.Skipf("Chrome Stable unavailable: %v", err)
	}

	run(t, binary, "install")
	t.Cleanup(func() { _ = exec.Command(binary, "uninstall").Run() })
	profile := filepath.Join(t.TempDir(), "profile")
	installProfileManifest(t, profile)
	ctx, cancel := context.WithCancel(context.Background())
	launcher, err := filepath.Abs("../extension/scripts/launch-integration-chrome.mjs")
	if err != nil {
		t.Fatal(err)
	}
	process := exec.CommandContext(ctx, "node", launcher, chrome, extension, profile)
	stdin, err := process.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := process.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	process.Stderr = os.Stderr
	if err := process.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- process.Wait() }()
	ready := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		if scanner.Scan() {
			ready <- scanner.Text()
		} else {
			ready <- ""
		}
	}()
	select {
	case line := <-ready:
		if !strings.HasPrefix(line, "READY chrome-extension://"+buildinfo.ExtensionID+"/") {
			t.Fatalf("Puppeteer extension launcher stopped before ready")
		}
	case err := <-done:
		t.Fatalf("Puppeteer extension launcher exited: %v", err)
	case <-time.After(40 * time.Second):
		t.Fatal("Puppeteer extension launcher timed out")
	}
	t.Log("extension service worker ready")
	nativeStart := time.Now()
	t.Cleanup(func() {
		_ = stdin.Close()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			cancel()
			<-done
		}
	})

	discoveryPath, err := discovery.DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	client := mcpclient.New(discoveryPath)
	deadline := time.Now().Add(30 * time.Second)
	for {
		file, resolveErr := client.Resolve()
		if resolveErr == nil {
			endpoint, parseErr := url.Parse(file.Endpoint)
			if parseErr != nil || endpoint.Hostname() != "127.0.0.1" {
				t.Fatalf("non-loopback MCP endpoint: %q", file.Endpoint)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("Native Messaging did not start the host: %v", resolveErr)
		}
		time.Sleep(200 * time.Millisecond)
	}
	nativeReadyDuration := time.Since(nativeStart)
	t.Logf("Native Messaging host and loopback discovery ready in %s", nativeReadyDuration)
	if nativeReadyDuration > time.Second {
		t.Fatalf("Native Host readiness exceeded 1s target: %s", nativeReadyDuration)
	}

	var direct tools.TabsListResult
	if err := client.Call(context.Background(), tools.ToolChromeTabsList, tools.TabsListInput{}, &direct); err != nil {
		t.Fatalf("direct HTTP MCP: %v", err)
	}
	t.Log("direct HTTP MCP call succeeded")
	cliOutput := run(t, binary, "--json", "tabs", "list")
	var cliResult tools.TabsListResult
	if err := json.Unmarshal(cliOutput, &cliResult); err != nil {
		t.Fatalf("CLI JSON: %v: %s", err, cliOutput)
	}
	t.Log("CLI call succeeded")

	command := exec.Command(binary, "mcp", "serve")
	sdkClient := mcp.NewClient(&mcp.Implementation{Name: "integration-test", Version: "1"}, nil)
	proxyStart := time.Now()
	session, err := sdkClient.Connect(context.Background(), &mcp.CommandTransport{Command: command}, nil)
	if err != nil {
		t.Fatalf("stdio MCP connect: %v", err)
	}
	proxyReadyDuration := time.Since(proxyStart)
	t.Logf("stdio MCP initialized in %s", proxyReadyDuration)
	if proxyReadyDuration > 500*time.Millisecond {
		t.Fatalf("stdio MCP initialization exceeded 500ms target: %s", proxyReadyDuration)
	}
	defer session.Close()
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: tools.ToolChromeTabsList, Arguments: tools.TabsListInput{}})
	if err != nil || result.IsError {
		t.Fatalf("stdio MCP call: result=%#v err=%v", result, err)
	}
	t.Log("stdio MCP proxy call succeeded")
}

func installProfileManifest(t *testing.T, profile string) {
	t.Helper()
	config, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	source := install.NativeMessagingManifestIn(config)
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	destinations := []string{
		filepath.Join(profile, "NativeMessagingHosts", filepath.Base(source)),
		filepath.Join(config, "Google", "ChromeForTesting", "NativeMessagingHosts", filepath.Base(source)),
	}
	for _, destination := range destinations {
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(destination, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for _, destination := range destinations {
			_ = os.Remove(destination)
		}
	})
}

func requirePath(t *testing.T, name string) string {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		t.Fatalf("%s is required", name)
	}
	absolute, err := filepath.Abs(value)
	if err != nil {
		t.Fatal(err)
	}
	return absolute
}

func run(t *testing.T, name string, args ...string) []byte {
	t.Helper()
	command := exec.Command(name, args...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, output)
	}
	return output
}

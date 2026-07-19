package mcpserver

import (
	"context"
	"encoding/base64"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/masahide/tabcli/internal/discovery"
	"github.com/masahide/tabcli/internal/tools"
)

func TestStartPublishesAuthenticatedRandomLoopbackServerAndCleansUp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "discovery.json")
	mcpService := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0.1.0"}, nil)
	running, err := Start(Config{
		ListenAddress:   "127.0.0.1:0",
		DiscoveryPath:   path,
		ProfileID:       "default",
		ProtocolVersion: tools.ProtocolVersion,
		MCPServer:       mcpService,
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() { _ = running.Close(context.Background()) })

	decodedToken, err := base64.RawURLEncoding.DecodeString(running.Token)
	if err != nil || len(decodedToken) != 32 {
		t.Fatalf("token has %d decoded bytes, want 32 (error %v)", len(decodedToken), err)
	}
	file, err := discovery.Read(path, discovery.ReadOptions{ProtocolVersion: tools.ProtocolVersion, ProcessAlive: func(int) bool { return true }})
	if err != nil {
		t.Fatalf("Read(discovery) error = %v", err)
	}
	if file.Endpoint != running.Endpoint || file.Token != running.Token || file.InstanceID != running.InstanceID {
		t.Fatalf("discovery = %#v, running = %#v", file, running)
	}

	request, err := http.NewRequest(http.MethodGet, running.Endpoint, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want 401", response.StatusCode)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := running.Close(ctx); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("discovery file remains after close: %v", err)
	}
}

package mcpserver

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/masahide/tabcli/internal/discovery"
)

type Config struct {
	ListenAddress   string
	DiscoveryPath   string
	ProfileID       string
	ProtocolVersion int
	MCPServer       *mcp.Server
}

type RunningServer struct {
	Endpoint   string
	InstanceID string
	Token      string

	discoveryPath string
	listener      net.Listener
	httpServer    *http.Server
	closeOnce     sync.Once
	serveError    chan error
}

func Start(config Config) (*RunningServer, error) {
	if config.ListenAddress == "" {
		config.ListenAddress = "127.0.0.1:0"
	}
	if err := ValidateListenAddress(config.ListenAddress); err != nil {
		return nil, err
	}
	if config.MCPServer == nil {
		return nil, errors.New("MCP server is required")
	}
	listener, err := net.Listen("tcp", config.ListenAddress)
	if err != nil {
		return nil, err
	}
	cleanupListener := true
	defer func() {
		if cleanupListener {
			_ = listener.Close()
		}
	}()
	token, err := randomToken(32)
	if err != nil {
		return nil, err
	}
	instanceID, err := randomHex(16)
	if err != nil {
		return nil, err
	}
	host := listener.Addr().String()
	endpoint := "http://" + host + "/mcp"
	streamable := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return config.MCPServer },
		&mcp.StreamableHTTPOptions{JSONResponse: true},
	)
	httpServer := &http.Server{
		Handler:           SecurityMiddleware(token, host, streamable),
		ReadHeaderTimeout: 5 * time.Second,
	}
	running := &RunningServer{
		Endpoint:      endpoint,
		InstanceID:    instanceID,
		Token:         token,
		discoveryPath: config.DiscoveryPath,
		listener:      listener,
		httpServer:    httpServer,
		serveError:    make(chan error, 1),
	}
	if err := discovery.Write(config.DiscoveryPath, discovery.File{
		Endpoint:        endpoint,
		PID:             os.Getpid(),
		InstanceID:      instanceID,
		ProfileID:       config.ProfileID,
		ProtocolVersion: config.ProtocolVersion,
		CreatedAt:       time.Now().UTC(),
		Token:           token,
	}); err != nil {
		return nil, err
	}
	cleanupListener = false
	go func() {
		err := httpServer.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		running.serveError <- err
	}()
	return running, nil
}

func (server *RunningServer) Close(ctx context.Context) error {
	var closeErr error
	server.closeOnce.Do(func() {
		shutdownErr := server.httpServer.Shutdown(ctx)
		removeErr := discovery.RemoveIfInstance(server.discoveryPath, server.InstanceID)
		closeErr = errors.Join(shutdownErr, removeErr)
	})
	return closeErr
}

func (server *RunningServer) Wait() error {
	return <-server.serveError
}

func randomToken(bytes int) (string, error) {
	buffer := make([]byte, bytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate bearer token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func randomHex(bytes int) (string, error) {
	buffer := make([]byte, bytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate instance ID: %w", err)
	}
	return hex.EncodeToString(buffer), nil
}

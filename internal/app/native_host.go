package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/masahide/tabcli/internal/buildinfo"
	"github.com/masahide/tabcli/internal/discovery"
	"github.com/masahide/tabcli/internal/mcpserver"
	"github.com/masahide/tabcli/internal/nativemsg"
	"github.com/masahide/tabcli/internal/plan"
	"github.com/masahide/tabcli/internal/tools"
)

func RunNativeHost(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, discoveryPath string) error {
	bridge := nativemsg.NewBridge(stdin, stdout)
	chromeBridge := nativemsg.ChromeBridge{Native: bridge}
	mcpService := mcp.NewServer(
		&mcp.Implementation{Name: "tabcli", Version: buildinfo.Version},
		&mcp.ServerOptions{Instructions: ProxyInstructions},
	)
	tools.RegisterListTools(mcpService, chromeBridge)
	tools.RegisterContentTools(mcpService, chromeBridge)
	tools.RegisterCloseTool(mcpService, chromeBridge)
	previewManager := plan.NewPreviewManager(chromeBridge, plan.PreviewOptions{
		ContentRevisionValid: func(int, string) bool { return true },
	})
	tools.RegisterPreviewTool(mcpService, validatedPreviewEngine{bridge: chromeBridge, manager: previewManager})
	applyManager := plan.NewApplyManager(previewManager, chromeBridge)
	tools.RegisterApplyTools(mcpService, applyManager)
	running, err := mcpserver.Start(mcpserver.Config{
		ListenAddress:   "127.0.0.1:0",
		DiscoveryPath:   discoveryPath,
		ProfileID:       discovery.DefaultProfileID,
		ProtocolVersion: tools.ProtocolVersion,
		MCPServer:       mcpService,
	})
	if err != nil {
		return err
	}
	logger := slog.New(slog.NewJSONHandler(stderr, nil))
	logger.Info("native host started", "event", "native_host_started")
	defer func() {
		shutdownContext, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := running.Close(shutdownContext); err != nil {
			logger.Error("native host cleanup failed", "event", "native_host_cleanup_failed", "errorType", "cleanup")
		}
	}()

	err = bridge.Run(ctx)
	if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
		logger.Info("native host stopped", "event", "native_host_stopped")
		return nil
	}
	logger.Error("native host connection ended", "event", "native_host_connection_ended", "errorType", "native_messaging")
	return err
}

type validatedPreviewEngine struct {
	bridge  nativemsg.ChromeBridge
	manager *plan.PreviewManager
}

func (engine validatedPreviewEngine) Preview(ctx context.Context, classification tools.ClassificationPlan) (tools.PreviewResult, error) {
	if err := engine.bridge.ValidateContentRevisions(ctx, classification.ContentRevisions); err != nil {
		return tools.PreviewResult{}, err
	}
	return engine.manager.Preview(ctx, classification)
}

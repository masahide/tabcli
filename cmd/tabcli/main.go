package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/masahide/tabcli/internal/app"
	"github.com/masahide/tabcli/internal/buildinfo"
	"github.com/masahide/tabcli/internal/cli"
	"github.com/masahide/tabcli/internal/discovery"
	"github.com/masahide/tabcli/internal/install"
	"github.com/masahide/tabcli/internal/mcpclient"
	"github.com/masahide/tabcli/internal/tools"
)

var (
	allowedExtensionOrigin             = buildinfo.AllowedExtensionOrigin
	cliHandler             app.Handler = runCLI
	nativeHostHandler      app.Handler = runNativeHost
)

func main() {
	err := app.Run(
		context.Background(),
		os.Args[1:],
		allowedExtensionOrigin,
		app.Handlers{CLI: cliHandler, NativeHost: nativeHostHandler},
		os.Stdin,
		os.Stdout,
		os.Stderr,
	)
	if err != nil {
		var exit app.ExitError
		if errors.As(err, &exit) {
			os.Exit(exit.Code)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runNativeHost(ctx context.Context, _ []string, stdin io.Reader, stdout, stderr io.Writer) error {
	discoveryPath, err := discovery.DefaultPath()
	if err != nil {
		return err
	}
	return app.RunNativeHost(ctx, stdin, stdout, stderr, discoveryPath)
}

func runCLI(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	discoveryPath, err := discovery.DefaultPath()
	if err != nil {
		return err
	}
	client := mcpclient.New(discoveryPath)
	if len(args) == 2 && args[0] == "mcp" && args[1] == "serve" {
		return app.RunStdioProxy(ctx, stdin, stdout, stderr, client)
	}
	command := cli.Command{
		Caller: client,
		Version: map[string]any{
			"version": buildinfo.Version, "commit": buildinfo.Commit, "builtAt": buildinfo.BuiltAt,
			"goVersion": runtime.Version(), "os": runtime.GOOS, "arch": runtime.GOARCH,
			"protocolVersion":        buildinfo.ProtocolVersion,
			"minimumProtocolVersion": buildinfo.MinimumProtocolVersion,
			"maximumProtocolVersion": buildinfo.MaximumProtocolVersion,
			"extensionId":            buildinfo.ExtensionID,
		},
		Install: func() (string, error) {
			executable, err := install.CurrentExecutable()
			if err != nil {
				return "", err
			}
			return install.Install(executable)
		},
		Uninstall: func() (any, error) {
			manifestPath, err := install.NativeMessagingManifest()
			if err != nil {
				return nil, err
			}
			productDirectory, err := install.ProductDirectory()
			if err != nil {
				return nil, err
			}
			return install.Uninstall(install.UninstallOptions{ManifestPath: manifestPath, ProductDirectory: productDirectory})
		},
		Status: func() (any, error) {
			file, err := discovery.Read(discoveryPath, discovery.ReadOptions{ProtocolVersion: tools.ProtocolVersion})
			if err != nil {
				return nil, tools.NewError(tools.CodeBrowserDisconnected, "Chrome is not connected. Start Chrome with the extension enabled, then retry.")
			}
			return cli.StatusResult{
				Connected: true, Endpoint: file.Endpoint, PID: file.PID, InstanceID: file.InstanceID,
				ProfileID: file.ProfileID, ProtocolVersion: file.ProtocolVersion, CreatedAt: file.CreatedAt,
			}, nil
		},
		Doctor: func() (any, error) {
			executable, err := install.CurrentExecutable()
			if err != nil {
				return nil, err
			}
			manifestPath, err := install.NativeMessagingManifest()
			if err != nil {
				return nil, err
			}
			return install.Diagnose(install.DoctorOptions{
				ExecutablePath: executable,
				ManifestPath:   manifestPath,
				DiscoveryPath:  discoveryPath,
				CheckChrome: func() error {
					_, err := discovery.Read(discoveryPath, discovery.ReadOptions{ProtocolVersion: tools.ProtocolVersion})
					return err
				},
				CheckMCP: func() error {
					var result tools.TabsListResult
					return client.Call(ctx, tools.ToolChromeTabsList, tools.TabsListInput{}, &result)
				},
			}), nil
		},
	}
	if code := command.Run(ctx, args, stdout, stderr); code != cli.ExitOK {
		return app.ExitError{Code: code}
	}
	return nil
}

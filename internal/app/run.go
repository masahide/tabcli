package app

import (
	"context"
	"fmt"
	"io"
)

type Handler func(context.Context, []string, io.Reader, io.Writer, io.Writer) error

type Handlers struct {
	CLI        Handler
	NativeHost Handler
}

type ExitError struct {
	Code int
}

func (err ExitError) Error() string { return fmt.Sprintf("exit status %d", err.Code) }

func Run(ctx context.Context, args []string, allowedOrigin string, handlers Handlers, stdin io.Reader, stdout, stderr io.Writer) error {
	mode, err := DetectMode(args, allowedOrigin)
	if err != nil {
		return err
	}
	switch mode {
	case ModeCLI:
		if handlers.CLI == nil {
			return fmt.Errorf("CLI handler is not configured")
		}
		return handlers.CLI(ctx, args, stdin, stdout, stderr)
	case ModeNativeHost:
		if handlers.NativeHost == nil {
			return fmt.Errorf("native host handler is not configured")
		}
		return handlers.NativeHost(ctx, args[1:], stdin, stdout, stderr)
	default:
		return fmt.Errorf("unsupported mode %q", mode)
	}
}

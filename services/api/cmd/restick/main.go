package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/eykh/restick/api_svc/internal/infrastructure/config"
)

// main runs one restick process; every command shares the same data dir and
// app lock rules.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:]); err != nil {
		slog.Error("restick", "err", err)
		os.Exit(1)
	}
}

// run dispatches the CLI. `restick` and `restick serve` start the server.
func run(ctx context.Context, args []string) error {
	command := "serve"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		command, args = args[0], args[1:]
	}
	switch command {
	case "serve":
		if len(args) != 0 {
			return fmt.Errorf("serve: unexpected arguments %v", args)
		}
		cfg, err := config.LoadFactory()
		if err != nil {
			return err
		}
		return serve(ctx, cfg)
	case "healthcheck":
		cfg, err := config.LoadFactory()
		if err != nil {
			return err
		}
		return healthcheck(cfg)
	case "admin":
		return runAdmin(ctx, args)
	case "backup":
		output, err := parseOneFlag("backup", args, "output")
		if err != nil {
			return err
		}
		return backup(ctx, config.LoadDataDir(), output)
	case "restore":
		flags, err := parseFlags("restore", args, "input", "password-file")
		if err != nil {
			return err
		}
		return restore(ctx, config.LoadDataDir(), flags["input"], flags["password-file"])
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}

func runAdmin(ctx context.Context, args []string) error {
	if len(args) == 0 || args[0] != "set-password" {
		return fmt.Errorf("usage: restick admin set-password --password-file FILE")
	}
	passwordFile, err := parseOneFlag("admin set-password", args[1:], "password-file")
	if err != nil {
		return err
	}
	return setPassword(ctx, config.LoadDataDir(), passwordFile)
}

// parseFlags parses the named string flags and rejects missing values.
func parseFlags(name string, args []string, names ...string) (map[string]string, error) {
	set := flag.NewFlagSet(name, flag.ContinueOnError)
	set.SetOutput(io.Discard)
	values := make(map[string]*string, len(names))
	for _, n := range names {
		values[n] = set.String(n, "", "")
	}
	if err := set.Parse(args); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	if set.NArg() != 0 {
		return nil, fmt.Errorf("%s: unexpected arguments %v", name, set.Args())
	}
	out := make(map[string]string, len(names))
	for _, n := range names {
		if strings.TrimSpace(*values[n]) == "" {
			return nil, fmt.Errorf("%s: --%s is required", name, n)
		}
		out[n] = *values[n]
	}
	return out, nil
}

func parseOneFlag(name string, args []string, flagName string) (string, error) {
	values, err := parseFlags(name, args, flagName)
	if err != nil {
		return "", err
	}
	return values[flagName], nil
}

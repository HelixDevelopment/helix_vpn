// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Vasic Digital. All rights reserved.

// Package main provides the standalone LLMOrchestrator CLI.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"digital.vasic.llmorchestrator/pkg/agent"
	"digital.vasic.llmorchestrator/pkg/adapter"
	"digital.vasic.llmorchestrator/pkg/config"
)

const version = "0.1.0"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		// CONST-046 round-387: version banner routed through i18n.
		fmt.Print(cliMsgf(
			"llmorchestrator_cli_version_banner",
			map[string]any{"version": version},
			"LLMOrchestrator v%s\n", version,
		))
		os.Exit(0)
	}

	// Load configuration.
	cfg := config.LoadFromEnvironment()
	if envFile := os.Getenv("HELIX_ENV_FILE"); envFile != "" {
		var err error
		cfg, err = config.LoadFromEnv(envFile)
		if err != nil {
			// CONST-046 round-387: config-load error routed through i18n.
			fmt.Fprint(os.Stderr, cliMsgf(
				"llmorchestrator_cli_config_load_error",
				map[string]any{"error": err.Error()},
				"Error loading config: %v\n", err,
			))
			os.Exit(1)
		}
	}

	if err := cfg.Validate(); err != nil {
		// CONST-046 round-387: invalid-config error routed through i18n.
		fmt.Fprint(os.Stderr, cliMsgf(
			"llmorchestrator_cli_config_invalid",
			map[string]any{"error": err.Error()},
			"Invalid config: %v\n", err,
		))
		os.Exit(1)
	}

	// Create agent pool.
	pool := agent.NewPool()

	// Register enabled agents.
	for _, name := range cfg.EnabledAgents {
		path, err := cfg.AgentBinaryPath(name)
		if err != nil {
			// CONST-046 round-387: agent-skipped warning routed through i18n.
			fmt.Fprint(os.Stderr, cliMsgf(
				"llmorchestrator_cli_agent_skipped",
				map[string]any{"agent": name, "error": err.Error()},
				"Warning: skipping agent %s: %v\n", name, err,
			))
			continue
		}

		adapterCfg := adapter.AdapterConfig{
			BinaryPath: path,
			Timeout:    cfg.AgentTimeout,
			MaxRetries: cfg.MaxRetries,
			OutputDir:  cfg.SessionDir("default") + "/" + name,
		}

		var a agent.Agent
		switch name {
		case "opencode":
			a = adapter.NewOpenCodeAgent(name+"-0", adapterCfg)
		case "claude-code":
			a = adapter.NewClaudeCodeAgent(name+"-0", adapterCfg)
		case "gemini":
			a = adapter.NewGeminiAgent(name+"-0", adapterCfg)
		case "junie":
			a = adapter.NewJunieAgent(name+"-0", adapterCfg)
		case "qwen-code":
			a = adapter.NewQwenCodeAgent(name+"-0", adapterCfg)
		default:
			// CONST-046 round-387: unknown-agent error routed through i18n.
			fmt.Fprint(os.Stderr, cliMsgf(
				"llmorchestrator_cli_agent_unknown",
				map[string]any{"agent": name},
				"Unknown agent: %s\n", name,
			))
			continue
		}

		if err := pool.Register(a); err != nil {
			// CONST-046 round-387: agent-register error routed through i18n.
			fmt.Fprint(os.Stderr, cliMsgf(
				"llmorchestrator_cli_agent_register_error",
				map[string]any{"agent": name, "error": err.Error()},
				"Error registering agent %s: %v\n", name, err,
			))
			continue
		}
		// CONST-046 round-387: agent-registered notice routed through i18n.
		fmt.Print(cliMsgf(
			"llmorchestrator_cli_agent_registered",
			map[string]any{"name": a.Name(), "id": a.ID()},
			"Registered agent: %s (%s)\n", a.Name(), a.ID(),
		))
	}

	// CONST-046 round-387: ready banner routed through i18n.
	fmt.Print(cliMsgf(
		"llmorchestrator_cli_ready_banner",
		map[string]any{"version": version, "count": len(pool.Available())},
		"LLMOrchestrator v%s ready with %d agents\n", version, len(pool.Available()),
	))

	// Wait for interrupt.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	// CONST-046 round-387: shutting-down notice routed through i18n.
	fmt.Println(cliMsg(
		"llmorchestrator_cli_shutting_down",
		nil,
		"\nShutting down...",
	))

	if err := pool.Shutdown(ctx); err != nil {
		// CONST-046 round-387: shutdown error routed through i18n.
		fmt.Fprint(os.Stderr, cliMsgf(
			"llmorchestrator_cli_shutdown_error",
			map[string]any{"error": err.Error()},
			"Shutdown error: %v\n", err,
		))
		os.Exit(1)
	}

	// CONST-046 round-387: shutdown-complete notice routed through i18n.
	fmt.Println(cliMsg(
		"llmorchestrator_cli_shutdown_complete",
		nil,
		"Shutdown complete.",
	))
}

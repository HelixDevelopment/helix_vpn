// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package agent

import (
	"os/exec"
	"syscall"
)

// setProcessGroup configures the command to run in its own process
// group (Setpgid) so killProcessGroup can later send SIGKILL to the
// whole group rather than only to the direct child.
//
// This is what makes ctx-cancellation reliable for the OpenCode CLI:
// without it, `exec.CommandContext` only kills the direct child (e.g.
// `/bin/sh -c …` or `opencode` itself), leaving any grandchild
// (`sleep`, `tail -f`, the opencode TUI's subprocesses) orphaned and
// the Send call hanging on stdout-pipe drainage.
func setProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

// killProcessGroup sends SIGKILL to the entire process group of cmd,
// reaping the leader plus every descendant. Safe to call when the
// process has already exited (kill returns ESRCH which we ignore).
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		// Fall back to killing just the direct child.
		_ = cmd.Process.Kill()
		return nil
	}
	// Negative PID targets the process group.
	_ = syscall.Kill(-pgid, syscall.SIGKILL)
	return nil
}

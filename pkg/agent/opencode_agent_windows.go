// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

//go:build windows

package agent

import "os/exec"

// setProcessGroup is a no-op on Windows; ctx-cancellation goes through
// exec.CommandContext's default Cancel hook which calls cmd.Process.Kill
// — sufficient for most Windows CLIs.
func setProcessGroup(_ *exec.Cmd) {}

// killProcessGroup falls back to killing the direct child only on
// Windows. A proper Win32 job-object implementation is follow-up work
// (see round 65+); this stub keeps the bridge functional on Windows
// for the no-grandchild common case.
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	_ = cmd.Process.Kill()
	return nil
}

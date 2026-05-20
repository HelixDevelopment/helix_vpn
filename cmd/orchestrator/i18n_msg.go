// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Vasic Digital. All rights reserved.

package main

import (
	"context"
	"fmt"

	"digital.vasic.llmorchestrator/pkg/i18n"
)

// cliMsg renders a user-facing console message for messageID through the
// package-level i18n translator (CONST-046 round-387). When the active
// translator is the NoopTranslator (returns the message ID verbatim) it
// falls back to fallback so the wire-evidence path keeps the captured
// data visible to the operator regardless of whether a consumer has
// installed a real translator yet.
//
// fallback MUST be the exact original English literal so the standalone
// CLI behaves identically when no translator is wired.
func cliMsg(
	messageID string,
	templateData map[string]any,
	fallback string,
) string {
	msg, err := i18n.Pkg().T(context.Background(), messageID, templateData)
	if err == nil && msg != "" && msg != messageID {
		return msg
	}
	return fallback
}

// cliMsgf is the printf-style convenience wrapper: it builds the
// fallback from a Go format string and args so call sites read close
// to the original fmt.Sprintf they replace.
func cliMsgf(
	messageID string,
	templateData map[string]any,
	format string,
	args ...any,
) string {
	return cliMsg(messageID, templateData, fmt.Sprintf(format, args...))
}

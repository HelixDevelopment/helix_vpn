#!/usr/bin/env bash
# scripts/install_git_hooks.sh
# Installs the .githooks pre-commit hook by pointing core.hooksPath at .githooks/.
# Idempotent: safe to run multiple times.
# Usage: bash scripts/install_git_hooks.sh

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Verify we are inside a git repo
if ! git -C "${REPO_ROOT}" rev-parse --git-dir > /dev/null 2>&1; then
    echo "ERROR: ${REPO_ROOT} is not a git repository." >&2
    exit 1
fi

HOOKS_DIR=".githooks"
HOOKS_PATH="${REPO_ROOT}/${HOOKS_DIR}"

if [ ! -d "${HOOKS_PATH}" ]; then
    echo "ERROR: hooks directory not found: ${HOOKS_PATH}" >&2
    echo "       Run this script from the repo that contains ${HOOKS_DIR}/." >&2
    exit 1
fi

CURRENT="$(git -C "${REPO_ROOT}" config --local core.hooksPath 2>/dev/null || true)"

if [ "${CURRENT}" = "${HOOKS_DIR}" ]; then
    echo "INFO: core.hooksPath is already set to '${HOOKS_DIR}' — nothing to do."
else
    git -C "${REPO_ROOT}" config core.hooksPath "${HOOKS_DIR}"
    echo "INFO: core.hooksPath set to '${HOOKS_DIR}' for repo at ${REPO_ROOT}."
fi

# Confirm the hook file is executable
HOOK_FILE="${HOOKS_PATH}/pre-commit"
if [ -f "${HOOK_FILE}" ] && [ ! -x "${HOOK_FILE}" ]; then
    chmod +x "${HOOK_FILE}"
    echo "INFO: Made ${HOOK_FILE} executable."
fi

echo "Done. Git will now run .githooks/pre-commit before every commit in this repo."

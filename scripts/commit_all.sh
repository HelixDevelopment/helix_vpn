#!/usr/bin/env bash
# scripts/commit_all.sh
# §2/§2.1 (Constitution.md lines 168/185) canonical single commit+push entrypoint.
# §11.4.88 (Constitution.md line 7512) background-push mandate.
#
# Usage:
#   bash scripts/commit_all.sh --dry-run
#   bash scripts/commit_all.sh --sync-push -m "commit message"
#   bash scripts/commit_all.sh -m "commit message"
#
# Flags:
#   --dry-run    Run gates and show status; do NOT commit or push.
#   --sync-push  Push synchronously instead of detached background (§11.4.88 escape hatch).
#   -m <msg>     Commit message (required unless --dry-run).
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Advisory lock via mkdir atomicity (macOS-compatible; macOS flock is unreliable).
# §2: "The commit and push wrappers MUST hold an advisory flock so two
#      invocations cannot race against each other."
LOCKDIR="${REPO_ROOT}/.git/.commit_all.lock"

DRY_RUN=0
SYNC_PUSH=0
MSG=""

# ---------------------------------------------------------------------------
# Parse arguments
# ---------------------------------------------------------------------------
while [ $# -gt 0 ]; do
    case "$1" in
        --dry-run)   DRY_RUN=1 ;;
        --sync-push) SYNC_PUSH=1 ;;
        -m)
            shift
            MSG="${1:-}"
            ;;
        *)
            echo "Unknown argument: $1" >&2
            echo "Usage: $0 [--dry-run] [--sync-push] [-m <message>]" >&2
            exit 1
            ;;
    esac
    shift
done

# ---------------------------------------------------------------------------
# Acquire advisory lock
# ---------------------------------------------------------------------------
cleanup_lock() {
    rmdir "${LOCKDIR}" 2>/dev/null || true
}
trap 'cleanup_lock; exit' EXIT INT TERM

if ! mkdir "${LOCKDIR}" 2>/dev/null; then
    echo "ERROR: Another commit_all.sh is already running." >&2
    echo "       Lock directory: ${LOCKDIR}" >&2
    echo "       If the owning process has been killed, remove it with:" >&2
    echo "         rmdir '${LOCKDIR}'" >&2
    exit 1
fi

echo "=== commit_all.sh (§2/§2.1/§11.4.88) ==="
echo "Repo root: ${REPO_ROOT}"
echo ""

# ---------------------------------------------------------------------------
# §3: Refuse if constitution/ has staged changes
# (submodule commits must be done first, before the parent pointer is captured)
# ---------------------------------------------------------------------------
CONST_STAGED="$(git -C "${REPO_ROOT}" diff --cached --name-only -- constitution/ 2>/dev/null || true)"
# Also catch UNSTAGED pointer/content drift: the `git add -A` below would
# otherwise stage and commit a submodule pointer bump, defeating the §3 guard.
CONST_UNSTAGED="$(git -C "${REPO_ROOT}" diff --name-only -- constitution/ 2>/dev/null || true)"
if [ -n "${CONST_STAGED}" ] || [ -n "${CONST_UNSTAGED}" ]; then
    echo "ERROR: constitution/ has changes (staged or unstaged)." >&2
    echo "       Commit the submodule first (§3):" >&2
    echo "         cd constitution && git add -A && git commit -m '...' && cd .." >&2
    echo "       Changed constitution/ entries:" >&2
    echo "${CONST_STAGED}" >&2
    echo "${CONST_UNSTAGED}" >&2
    exit 1
fi

# ---------------------------------------------------------------------------
# Gate enforcement (§11.4.32): run pre_build_verification.sh before any commit
# ---------------------------------------------------------------------------
echo "--- Running pre_build_verification.sh gate ---"
if ! bash "${REPO_ROOT}/tests/pre_build_verification.sh"; then
    echo "" >&2
    echo "GATE FAIL: pre_build_verification.sh failed. Aborting." >&2
    exit 1
fi
echo "GATE PASS"
echo ""

# ---------------------------------------------------------------------------
# --dry-run: show what would happen, then exit cleanly
# ---------------------------------------------------------------------------
if [ "${DRY_RUN}" -eq 1 ]; then
    echo "DRY-RUN: gate passed. Would commit and push. No changes made."
    echo ""
    echo "Current git status:"
    git -C "${REPO_ROOT}" status --short
    cleanup_lock
    trap - EXIT INT TERM
    exit 0
fi

# ---------------------------------------------------------------------------
# Real commit path
# ---------------------------------------------------------------------------
if [ -z "${MSG}" ]; then
    echo "ERROR: No commit message supplied. Use -m <message>." >&2
    exit 1
fi

# Check there is something to commit
STAGED="$(git -C "${REPO_ROOT}" diff --cached --name-only 2>/dev/null || true)"
UNSTAGED="$(git -C "${REPO_ROOT}" diff --name-only 2>/dev/null || true)"
UNTRACKED="$(git -C "${REPO_ROOT}" ls-files --others --exclude-standard 2>/dev/null || true)"

if [ -z "${STAGED}" ] && [ -z "${UNSTAGED}" ] && [ -z "${UNTRACKED}" ]; then
    echo "Nothing to commit. Working tree clean."
    exit 0
fi

echo "--- Staging and committing ---"
git -C "${REPO_ROOT}" add -A
git -C "${REPO_ROOT}" commit -m "${MSG}"

# §11.4.88: Release lock immediately after git commit returns 0.
# Push MUST NOT hold the lock.
cleanup_lock
trap - EXIT INT TERM
echo "Lock released (§11.4.88)."
echo ""

# ---------------------------------------------------------------------------
# Push phase — multi-remote per §2.1
# ---------------------------------------------------------------------------
REMOTES="$(git -C "${REPO_ROOT}" remote 2>/dev/null || true)"
if [ -z "${REMOTES}" ]; then
    echo "No remotes configured. Push skipped."
    exit 0
fi

BRANCH="$(git -C "${REPO_ROOT}" rev-parse --abbrev-ref HEAD 2>/dev/null)"

do_push_all() {
    local remotes_list="$1"
    local repo="${2}"
    local branch="${3}"
    local failed=0
    for remote in ${remotes_list}; do
        echo "Pushing to ${remote}/${branch} ..."
        if git -C "${repo}" push "${remote}" "${branch}"; then
            echo "  Push to ${remote}: OK"
        else
            echo "  Push to ${remote}: FAILED" >&2
            failed=1
        fi
    done
    return "${failed}"
}

if [ "${SYNC_PUSH}" -eq 1 ]; then
    # Synchronous push (--sync-push escape hatch per §11.4.88)
    echo "--- Synchronous push (--sync-push) ---"
    do_push_all "${REMOTES}" "${REPO_ROOT}" "${BRANCH}"
else
    # §11.4.88 background push via nohup + disown
    # Per-remote failure logs land in qa-results/push_failures/
    PUSH_LOG_DIR="${REPO_ROOT}/qa-results/push_failures"
    mkdir -p "${PUSH_LOG_DIR}"
    PUSH_LOG="${PUSH_LOG_DIR}/$(date +%Y%m%dT%H%M%S)_push.log"

    # Export variables needed inside the nohup subshell
    export _CA_REMOTES="${REMOTES}"
    export _CA_REPO_ROOT="${REPO_ROOT}"
    export _CA_BRANCH="${BRANCH}"

    nohup bash -c '
        failed=0
        for remote in ${_CA_REMOTES}; do
            echo "$(date +%Y%m%dT%H%M%S) Pushing to ${remote}/${_CA_BRANCH} ..."
            if git -C "${_CA_REPO_ROOT}" push "${remote}" "${_CA_BRANCH}"; then
                echo "$(date +%Y%m%dT%H%M%S) OK: ${remote}"
            else
                echo "$(date +%Y%m%dT%H%M%S) FAIL: ${remote}"
                failed=1
            fi
        done
        exit "${failed}"
    ' > "${PUSH_LOG}" 2>&1 &
    disown

    echo "Background push started (§11.4.88)."
    echo "Log: ${PUSH_LOG}"
fi

echo ""
echo "DONE."
exit 0

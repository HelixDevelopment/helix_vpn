#!/usr/bin/env bash
# scripts/testing/meta_test_precommit_hook.sh
# §1.1 paired proof: verifies that .githooks/pre-commit is discriminating —
# it ALLOWS a commit when all constitution invariants are satisfied and
# BLOCKS a commit when an invariant is broken.
#
# Operates entirely in a throwaway git repo under $(mktemp -d).
# NEVER touches the real repo or the real submodule.
#
# Exit 0  — hook allowed good commit AND blocked bad commit (proof complete).
# Exit 1  — either the allowed commit was blocked or the bad commit was allowed.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REAL_REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# ---------------------------------------------------------------------------
# Cleanup trap — always remove the temp dir on exit
# ---------------------------------------------------------------------------
TMPDIR_WORK=""
cleanup() {
    if [ -n "${TMPDIR_WORK}" ] && [ -d "${TMPDIR_WORK}" ]; then
        rm -rf "${TMPDIR_WORK}"
    fi
}
trap cleanup EXIT

# ---------------------------------------------------------------------------
# Create isolated throwaway repo
# ---------------------------------------------------------------------------
TMPDIR_WORK="$(mktemp -d)"
FAKE_REPO="${TMPDIR_WORK}/fake_repo"
mkdir -p "${FAKE_REPO}"

echo "[meta-test] Isolated temp repo: ${FAKE_REPO}"
echo "[meta-test] Real repo root:     ${REAL_REPO_ROOT}"

git -C "${FAKE_REPO}" init -q
git -C "${FAKE_REPO}" config user.email "test@example.com"
git -C "${FAKE_REPO}" config user.name  "Meta Test"

# ---------------------------------------------------------------------------
# Lay down the directory structure the gate script expects
# ---------------------------------------------------------------------------
mkdir -p "${FAKE_REPO}/constitution"
mkdir -p "${FAKE_REPO}/tests"
mkdir -p "${FAKE_REPO}/docs/guides"

# constitution/Constitution.md — must contain §11.4 anchor (em-dash U+2014)
cat > "${FAKE_REPO}/constitution/Constitution.md" <<'EOF'
# Constitution

### §11.4 End-user quality guarantee — forensic anchor (User mandate, 2026-04-28)

This is the forensic anchor required by Inv2.
EOF

# constitution/CLAUDE.md — must contain MANDATORY ANTI-BLUFF COVENANT
cat > "${FAKE_REPO}/constitution/CLAUDE.md" <<'EOF'
# CLAUDE

## MANDATORY ANTI-BLUFF COVENANT

All agents must not bluff.
EOF

# constitution/AGENTS.md — must contain Anti-bluff covenant
cat > "${FAKE_REPO}/constitution/AGENTS.md" <<'EOF'
# AGENTS

### Anti-bluff covenant — END-USER QUALITY GUARANTEE (§11.4)

All agents must comply.
EOF

# Parent CLAUDE.md — must reference constitution/CLAUDE.md
cat > "${FAKE_REPO}/CLAUDE.md" <<'EOF'
# Project CLAUDE.md

All rules in `constitution/CLAUDE.md` apply unconditionally.
EOF

# Parent AGENTS.md — must reference constitution/AGENTS.md
cat > "${FAKE_REPO}/AGENTS.md" <<'EOF'
# Project AGENTS.md

Base agent rules: `constitution/AGENTS.md` — READ IT FIRST.
EOF

# docs/guides/HELIX_VPN_CONSTITUTION.md — must reference constitution/Constitution.md
cat > "${FAKE_REPO}/docs/guides/HELIX_VPN_CONSTITUTION.md" <<'EOF'
# Helix VPN Constitution Guide

This extends `constitution/Constitution.md`.
EOF

# Copy in the real gate script (read-only use of real repo)
cp "${REAL_REPO_ROOT}/tests/pre_build_verification.sh" "${FAKE_REPO}/tests/pre_build_verification.sh"

# Copy in the real pre-commit hook
mkdir -p "${FAKE_REPO}/.githooks"
cp "${REAL_REPO_ROOT}/.githooks/pre-commit" "${FAKE_REPO}/.githooks/pre-commit"
chmod +x "${FAKE_REPO}/.githooks/pre-commit"

# Point git at our hooks dir
git -C "${FAKE_REPO}" config core.hooksPath ".githooks"

# ---------------------------------------------------------------------------
# Helper: stage all and attempt a commit; return git's exit code
# ---------------------------------------------------------------------------
attempt_commit() {
    local msg="$1"
    git -C "${FAKE_REPO}" add -A
    git -C "${FAKE_REPO}" commit -m "${msg}"
    return $?
}

# ---------------------------------------------------------------------------
# (a) GOOD COMMIT — all invariants satisfied — hook must ALLOW
# ---------------------------------------------------------------------------
echo ""
echo "[meta-test] === (a) Good commit — all invariants satisfied ==="

attempt_commit "good: all invariants satisfied"
GOOD_EXIT=$?

if [ "${GOOD_EXIT}" -eq 0 ]; then
    echo "[meta-test] PASS (a): hook allowed the good commit (exit 0)."
else
    echo "[meta-test] FAIL (a): hook BLOCKED the good commit (exit ${GOOD_EXIT}) — should have allowed it." >&2
    exit 1
fi

# ---------------------------------------------------------------------------
# (b) BAD COMMIT — break Inv2 by removing the §11.4 anchor from Constitution.md
# ---------------------------------------------------------------------------
echo ""
echo "[meta-test] === (b) Bad commit — §11.4 anchor removed from Constitution.md ==="

# Overwrite Constitution.md without the anchor line
cat > "${FAKE_REPO}/constitution/Constitution.md" <<'EOF'
# Constitution (anchor deliberately removed for meta-test)

This file no longer contains the required forensic anchor.
EOF

attempt_commit "bad: §11.4 anchor removed"
BAD_EXIT=$?

if [ "${BAD_EXIT}" -ne 0 ]; then
    echo "[meta-test] PASS (b): hook BLOCKED the bad commit (exit ${BAD_EXIT}) — as required."
else
    echo "[meta-test] FAIL (b): hook ALLOWED the bad commit — should have blocked it." >&2
    exit 1
fi

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo ""
echo "[meta-test] Both checks passed:"
echo "  (a) Good commit was ALLOWED  (hook is not over-blocking)"
echo "  (b) Bad commit was BLOCKED   (hook is discriminating)"
echo "[meta-test] EXIT 0 — proof complete."
exit 0

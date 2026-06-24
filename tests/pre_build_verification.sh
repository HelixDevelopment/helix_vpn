#!/usr/bin/env bash
# tests/pre_build_verification.sh
# Constitution-inheritance pre-build gate.
# Checks five invariants; prints PASS/FAIL per invariant; exits 0 only if all pass.
# Usage: bash tests/pre_build_verification.sh  (resolved from SCRIPT location, not $PWD)

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

pass_count=0
fail_count=0

check() {
    local inv="$1"
    local label="$2"
    local result="$3"   # "ok" or "fail"
    local detail="$4"
    if [ "$result" = "ok" ]; then
        echo "PASS  ${inv}: ${label}"
        pass_count=$((pass_count + 1))
    else
        echo "FAIL  ${inv}: ${label} — ${detail}"
        fail_count=$((fail_count + 1))
    fi
}

# ---------------------------------------------------------------------------
# Inv1: directory constitution/ exists
# ---------------------------------------------------------------------------
if [ -d "${REPO_ROOT}/constitution" ]; then
    check "Inv1" "constitution/ directory exists" "ok" ""
else
    check "Inv1" "constitution/ directory exists" "fail" \
        "directory ${REPO_ROOT}/constitution not found"
fi

# ---------------------------------------------------------------------------
# Inv2: constitution/Constitution.md contains the §11.4 forensic anchor
# ---------------------------------------------------------------------------
CONST_MD="${REPO_ROOT}/constitution/Constitution.md"
INV2_ANCHOR='§11.4 End-user quality guarantee — forensic anchor'
if [ -f "${CONST_MD}" ] && grep -qF "${INV2_ANCHOR}" "${CONST_MD}"; then
    check "Inv2" "constitution/Constitution.md has §11.4 forensic anchor" "ok" ""
elif [ ! -f "${CONST_MD}" ]; then
    check "Inv2" "constitution/Constitution.md has §11.4 forensic anchor" "fail" \
        "file not found: ${CONST_MD}"
else
    check "Inv2" "constitution/Constitution.md has §11.4 forensic anchor" "fail" \
        "anchor literal not found in ${CONST_MD}"
fi

# ---------------------------------------------------------------------------
# Inv3: constitution/CLAUDE.md contains MANDATORY ANTI-BLUFF COVENANT
# ---------------------------------------------------------------------------
CLAUDE_SUB="${REPO_ROOT}/constitution/CLAUDE.md"
INV3_ANCHOR='MANDATORY ANTI-BLUFF COVENANT'
if [ -f "${CLAUDE_SUB}" ] && grep -qF "${INV3_ANCHOR}" "${CLAUDE_SUB}"; then
    check "Inv3" "constitution/CLAUDE.md has MANDATORY ANTI-BLUFF COVENANT" "ok" ""
elif [ ! -f "${CLAUDE_SUB}" ]; then
    check "Inv3" "constitution/CLAUDE.md has MANDATORY ANTI-BLUFF COVENANT" "fail" \
        "file not found: ${CLAUDE_SUB}"
else
    check "Inv3" "constitution/CLAUDE.md has MANDATORY ANTI-BLUFF COVENANT" "fail" \
        "anchor literal not found in ${CLAUDE_SUB}"
fi

# ---------------------------------------------------------------------------
# Inv4: constitution/AGENTS.md contains Anti-bluff covenant
# ---------------------------------------------------------------------------
AGENTS_SUB="${REPO_ROOT}/constitution/AGENTS.md"
INV4_ANCHOR='Anti-bluff covenant'
if [ -f "${AGENTS_SUB}" ] && grep -qF "${INV4_ANCHOR}" "${AGENTS_SUB}"; then
    check "Inv4" "constitution/AGENTS.md has Anti-bluff covenant" "ok" ""
elif [ ! -f "${AGENTS_SUB}" ]; then
    check "Inv4" "constitution/AGENTS.md has Anti-bluff covenant" "fail" \
        "file not found: ${AGENTS_SUB}"
else
    check "Inv4" "constitution/AGENTS.md has Anti-bluff covenant" "fail" \
        "anchor literal not found in ${AGENTS_SUB}"
fi

# ---------------------------------------------------------------------------
# Inv5: parent files reference the submodule (all three must hold)
# ---------------------------------------------------------------------------
CLAUDE_PARENT="${REPO_ROOT}/CLAUDE.md"
AGENTS_PARENT="${REPO_ROOT}/AGENTS.md"
GUIDE_PARENT="${REPO_ROOT}/docs/guides/HELIX_VPN_CONSTITUTION.md"

inv5_ok=true
inv5_detail=""

if [ ! -f "${CLAUDE_PARENT}" ]; then
    inv5_ok=false
    inv5_detail="${inv5_detail}CLAUDE.md not found: ${CLAUDE_PARENT}; "
elif ! grep -qF 'constitution/CLAUDE.md' "${CLAUDE_PARENT}"; then
    inv5_ok=false
    inv5_detail="${inv5_detail}CLAUDE.md missing 'constitution/CLAUDE.md' reference; "
fi
if [ ! -f "${AGENTS_PARENT}" ]; then
    inv5_ok=false
    inv5_detail="${inv5_detail}AGENTS.md not found: ${AGENTS_PARENT}; "
elif ! grep -qF 'constitution/AGENTS.md' "${AGENTS_PARENT}"; then
    inv5_ok=false
    inv5_detail="${inv5_detail}AGENTS.md missing 'constitution/AGENTS.md' reference; "
fi
if [ ! -f "${GUIDE_PARENT}" ]; then
    inv5_ok=false
    inv5_detail="${inv5_detail}docs/guides/HELIX_VPN_CONSTITUTION.md not found: ${GUIDE_PARENT}; "
elif ! grep -qF 'constitution/Constitution.md' "${GUIDE_PARENT}"; then
    inv5_ok=false
    inv5_detail="${inv5_detail}docs/guides/HELIX_VPN_CONSTITUTION.md missing 'constitution/Constitution.md' reference; "
fi

if $inv5_ok; then
    check "Inv5" "parent files all reference submodule" "ok" ""
else
    check "Inv5" "parent files all reference submodule" "fail" "${inv5_detail%%; }"
fi

# ---------------------------------------------------------------------------
# Inv6: CM-NO-ACTIVE-CI — no active root CI workflow files (§11.4.156 part E)
# git ls-files must return empty for *.yml/*.yaml under .github/workflows/ and
# for .gitlab-ci.yml at the repo root.
# ---------------------------------------------------------------------------
inv6_ok=true
inv6_detail=""

if git -C "${REPO_ROOT}" ls-files | grep -qE '^\.github/workflows/.*\.ya?ml$'; then
    inv6_ok=false
    inv6_detail="${inv6_detail}active GitHub Actions workflow(s) found in git index; "
fi
if git -C "${REPO_ROOT}" ls-files | grep -qE '^\.gitlab-ci\.yml$'; then
    inv6_ok=false
    inv6_detail="${inv6_detail}.gitlab-ci.yml found in git index; "
fi

if $inv6_ok; then
    check "Inv6" "CM-NO-ACTIVE-CI: no active root CI workflow" "ok" ""
else
    check "Inv6" "CM-NO-ACTIVE-CI: no active root CI workflow" "fail" "${inv6_detail%%; }"
fi

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo ""
echo "Results: ${pass_count} passed, ${fail_count} failed."

if [ "${fail_count}" -gt 0 ]; then
    echo "STATUS: FAIL"
    exit 1
fi

echo "STATUS: PASS"
exit 0

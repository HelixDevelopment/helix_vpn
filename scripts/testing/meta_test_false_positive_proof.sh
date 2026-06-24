#!/usr/bin/env bash
# scripts/testing/meta_test_false_positive_proof.sh
# Mutation case: CM-CONSTITUTION-INHERITANCE
# Proves the pre_build_verification.sh gate is NOT a bluff gate by:
#   1. Asserting gate exits 0 on clean repo (GREEN baseline)
#   2. Mutating constitution/Constitution.md to break Inv2
#   3. Asserting gate exits non-zero (RED — gate caught the mutation)
#   4. Restoring constitution/Constitution.md from backup
#   5. Asserting gate exits 0 again (GREEN restored)
#   6. Verifying restored file is byte-identical to original (checksum)
#
# Data-safety (Constitution §9):
#   - Checksum + backup are captured BEFORE any mutation.
#   - A trap on EXIT restores the file on any exit path (success/failure/interrupt).
#   - Restore uses our own backup copy, NOT git checkout/restore.
#   - After restore we verify git submodule status and checksum independently.
#
# Exits 0 only if all assertions hold.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
GATE="${REPO_ROOT}/tests/pre_build_verification.sh"
TARGET="${REPO_ROOT}/constitution/Constitution.md"
MUTATION_CASE="CM-CONSTITUTION-INHERITANCE"

# ---- scratch space OUTSIDE the submodule ----
SCRATCH_DIR="$(mktemp -d)"
BACKUP_FILE="${SCRATCH_DIR}/Constitution.md.orig"
MUTATED_FILE="${SCRATCH_DIR}/Constitution.md.mut"

# We will set this to "true" once the mutation is live, so the trap knows
# whether to restore.
MUTATION_APPLIED=false

pass_count=0
fail_count=0

# ---------------------------------------------------------------------------
# Logging helpers
# ---------------------------------------------------------------------------
log()    { echo "[${MUTATION_CASE}] $*"; }
pass()   { log "PASS: $*"; pass_count=$((pass_count + 1)); }
fail()   { log "FAIL: $*"; fail_count=$((fail_count + 1)); }
die()    { log "FATAL: $*" >&2; exit 2; }

# ---------------------------------------------------------------------------
# Trap: restore original file on ANY exit (success, failure, interrupt).
# This is a §9 safety net; real restoration happens in step 4 with explicit
# verification — the trap is just the last-resort backstop.
# ---------------------------------------------------------------------------
restore_target() {
    if $MUTATION_APPLIED; then
        log "TRAP/EXIT: restoring ${TARGET} from backup..."
        cp "${BACKUP_FILE}" "${TARGET}" || {
            echo "[${MUTATION_CASE}] FATAL §9 VIOLATION: trap restore FAILED — submodule left dirty!" >&2
            exit 3
        }
        log "TRAP/EXIT: restore complete."
    fi
    # Clean up scratch dir
    rm -rf "${SCRATCH_DIR}"
}
trap restore_target EXIT

# ---------------------------------------------------------------------------
# Pre-flight: gate script must exist and be executable.
# ---------------------------------------------------------------------------
[ -f "${GATE}" ] || die "gate script not found: ${GATE}"
[ -x "${GATE}" ] || die "gate script not executable: ${GATE}"
[ -f "${TARGET}" ] || die "target file not found: ${TARGET}"

# ---------------------------------------------------------------------------
# Capture original checksum + backup BEFORE any mutation.
# ---------------------------------------------------------------------------
log "Capturing original checksum and backup..."
ORIG_SUM="$(shasum -a 256 "${TARGET}" | awk '{print $1}')"
cp "${TARGET}" "${BACKUP_FILE}" \
    || die "§9 VIOLATION: failed to create backup of ${TARGET}"
log "  Original SHA-256: ${ORIG_SUM}"

# Quick sanity: backup must be byte-identical to original right now.
BACKUP_SUM="$(shasum -a 256 "${BACKUP_FILE}" | awk '{print $1}')"
[ "${BACKUP_SUM}" = "${ORIG_SUM}" ] \
    || die "§9 VIOLATION: backup checksum mismatch immediately after copy"

# ---------------------------------------------------------------------------
# Step 1 — Baseline GREEN: gate must pass on clean repo.
# ---------------------------------------------------------------------------
log "Step 1: baseline GREEN — running gate on clean repo..."
if bash "${GATE}" > /dev/null 2>&1; then
    pass "Step 1 baseline GREEN (gate exited 0)"
else
    fail "Step 1 baseline FAILED (gate exited non-zero on clean repo)"
fi

# ---------------------------------------------------------------------------
# Step 2 — Mutate: replace the Inv2 anchor in constitution/Constitution.md
# so grep can no longer match it. We do NOT use sed -i to stay BSD-safe;
# instead we write a new file to MUTATED_FILE then copy it over.
# ---------------------------------------------------------------------------
log "Step 2: applying mutation to ${TARGET}..."
INV2_ANCHOR='§11.4 End-user quality guarantee — forensic anchor'
REPLACEMENT='§11.4 End-user quality guarantee — MUTATED_ANCHOR_REMOVED'

# Use Python (available on macOS) for reliable UTF-8 byte-safe replacement.
# Replace ALL occurrences so grep -qF finds none (the anchor appears in both
# the TOC entry and the section heading).
python3 -c "
import sys
with open(sys.argv[1], 'r', encoding='utf-8') as f:
    content = f.read()
if sys.argv[2] not in content:
    sys.exit(10)  # anchor not found — mutation would be a no-op
mutated = content.replace(sys.argv[2], sys.argv[3])  # replace ALL occurrences
with open(sys.argv[4], 'w', encoding='utf-8') as f:
    f.write(mutated)
" "${TARGET}" "${INV2_ANCHOR}" "${REPLACEMENT}" "${MUTATED_FILE}" \
    || die "Mutation preparation failed (anchor not found or python error)"

cp "${MUTATED_FILE}" "${TARGET}" \
    || die "§9 VIOLATION: failed to write mutated content to ${TARGET}"
MUTATION_APPLIED=true
log "  Mutation applied."

# ---------------------------------------------------------------------------
# Step 3 — Assert RED: gate must fail after mutation.
# ---------------------------------------------------------------------------
log "Step 3: assert RED — gate must detect the mutation..."
if bash "${GATE}" > /dev/null 2>&1; then
    fail "Step 3 RED assertion FAILED (gate exited 0 despite mutation — THIS IS A BLUFF GATE)"
else
    pass "Step 3 RED confirmed (gate exited non-zero — mutation detected)"
fi

# ---------------------------------------------------------------------------
# Step 4 — Restore from backup (explicit, not via git).
# ---------------------------------------------------------------------------
log "Step 4: restoring ${TARGET} from backup..."
cp "${BACKUP_FILE}" "${TARGET}" \
    || die "§9 VIOLATION: explicit restore from backup FAILED"
MUTATION_APPLIED=false   # disarm the trap (restore already done)
log "  Restore complete."

# Verify git submodule sees the file as clean.
GIT_STATUS="$(git -C "${REPO_ROOT}/constitution" status --porcelain 2>&1)"
if [ -z "${GIT_STATUS}" ]; then
    log "  git submodule status: clean (OK)"
else
    die "§9 VIOLATION: git submodule not clean after restore — status: ${GIT_STATUS}"
fi

# Verify checksum matches original.
RESTORED_SUM="$(shasum -a 256 "${TARGET}" | awk '{print $1}')"
if [ "${RESTORED_SUM}" = "${ORIG_SUM}" ]; then
    log "  Checksum after restore matches original: ${RESTORED_SUM} (OK)"
else
    die "§9 VIOLATION: restored file checksum MISMATCH — orig=${ORIG_SUM} restored=${RESTORED_SUM}"
fi

# ---------------------------------------------------------------------------
# Step 5 — Assert GREEN again: gate must pass after restore.
# ---------------------------------------------------------------------------
log "Step 5: assert GREEN after restore — running gate..."
if bash "${GATE}" > /dev/null 2>&1; then
    pass "Step 5 GREEN confirmed (gate exited 0 after restore)"
else
    fail "Step 5 GREEN assertion FAILED (gate still failing after restore)"
fi

# ---------------------------------------------------------------------------
# Step 6 — Byte-identical proof: shasum comparison.
# ---------------------------------------------------------------------------
log "Step 6: byte-identical proof..."
if [ "${RESTORED_SUM}" = "${ORIG_SUM}" ]; then
    pass "Step 6 byte-identical (SHA-256 orig=${ORIG_SUM} == restored=${RESTORED_SUM})"
else
    fail "Step 6 byte-identity FAILED (orig=${ORIG_SUM} != restored=${RESTORED_SUM})"
fi

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo ""
log "Results: ${pass_count} passed, ${fail_count} failed."

if [ "${fail_count}" -gt 0 ]; then
    log "STATUS: FAIL — mutation case ${MUTATION_CASE} detected problems"
    exit 1
fi

log "STATUS: PASS — gate is non-discriminating and correctly catches ${MUTATION_CASE}"
exit 0

#!/usr/bin/env bash
#
# ============================================================================
# sync_all_markdown_exports.sh
# ============================================================================
#
# Purpose:
#   Generate synchronized .html + .pdf sibling artifacts for every Markdown
#   document under docs/research/mvp/final/ (recursive), closing the
#   §11.4.65 Universal Markdown export gap for the HelixVPN spec tree.
#   Each source .md gets:
#     - <name>.html  (pandoc, standalone, embedded CSS, title from H1)
#     - <name>.pdf   (weasyprint, rendered from the generated HTML)
#
# Usage:
#   bash scripts/testing/sync_all_markdown_exports.sh [PATH_PREFIX] [--force]
#
#   PATH_PREFIX  Optional. Limit the walk to a sub-path under the scope root
#               (relative to repo root OR absolute). Default: the whole
#               docs/research/mvp/final/ tree.
#   --force      Regenerate siblings even if they are newer than the source.
#
#   Examples:
#     bash scripts/testing/sync_all_markdown_exports.sh
#     bash scripts/testing/sync_all_markdown_exports.sh docs/research/mvp/final/v02-data-plane
#     bash scripts/testing/sync_all_markdown_exports.sh --force
#
# Inputs:
#   - *.md files under docs/research/mvp/final/ (recursive)
#   - pandoc (HTML generation), weasyprint (PDF generation) on PATH
#
# Outputs:
#   - Sibling <name>.html and <name>.pdf next to each source <name>.md
#   - A summary line on stdout: "generated=N skipped=M failed=K"
#   - Exit 0 if no failures, exit 1 if any file/format failed.
#
# Side-effects:
#   - Writes .html / .pdf files into the spec tree (siblings only; never
#     modifies any .md source). Runs sequentially (concurrency = 1) to
#     respect the §12.6 60% memory ceiling. timeout 60 per file per format.
#
# Dependencies:
#   - bash, pandoc, weasyprint, timeout (coreutils gtimeout fallback), find
#
# Cross-references:
#   - Constitution §11.4.65 (Universal Markdown export mandate)
#   - Constitution §11.4.18 (script documentation mandate)
#   - Constitution §11.4.67 (target-shell-parseability — parses under sh -n + bash -n)
#   - Constitution §12.6   (60% memory ceiling — sequential, no fan-out)
#   - Companion guide: docs/scripts/sync_all_markdown_exports.md
# ============================================================================

set -u

# --- Resolve repo root (script lives in scripts/testing/) --------------------
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)
SCOPE_ROOT="$REPO_ROOT/docs/research/mvp/final"

# --- Parse args --------------------------------------------------------------
FORCE=0
PATH_PREFIX=""
for arg in "$@"; do
    case "$arg" in
        --force)
            FORCE=1
            ;;
        -*)
            echo "WARN: unknown flag '$arg' ignored" >&2
            ;;
        *)
            PATH_PREFIX="$arg"
            ;;
    esac
done

# --- Determine the walk root -------------------------------------------------
WALK_ROOT="$SCOPE_ROOT"
if [ -n "$PATH_PREFIX" ]; then
    case "$PATH_PREFIX" in
        /*) WALK_ROOT="$PATH_PREFIX" ;;
        *)  WALK_ROOT="$REPO_ROOT/$PATH_PREFIX" ;;
    esac
fi

if [ ! -d "$WALK_ROOT" ]; then
    echo "ERROR: walk root does not exist: $WALK_ROOT" >&2
    exit 1
fi

# --- Locate a timeout binary (gtimeout on macOS via coreutils) ---------------
TIMEOUT_BIN=""
if command -v timeout >/dev/null 2>&1; then
    TIMEOUT_BIN="timeout"
elif command -v gtimeout >/dev/null 2>&1; then
    TIMEOUT_BIN="gtimeout"
fi

run_with_timeout() {
    # $1 = seconds, rest = command. Falls back to no-timeout if unavailable.
    secs="$1"
    shift
    if [ -n "$TIMEOUT_BIN" ]; then
        "$TIMEOUT_BIN" "$secs" "$@"
    else
        "$@"
    fi
}

# --- Verify tooling ----------------------------------------------------------
if ! command -v pandoc >/dev/null 2>&1; then
    echo "ERROR: pandoc not found on PATH" >&2
    exit 1
fi
if ! command -v weasyprint >/dev/null 2>&1; then
    echo "ERROR: weasyprint not found on PATH" >&2
    exit 1
fi

# --- mtime helper (newer-than test): returns 0 if $1 is newer than $2 --------
is_newer_than() {
    # $1 newer than $2 ?  (sibling newer than source => skip)
    [ -e "$1" ] || return 1
    [ "$1" -nt "$2" ]
}

# --- Extract H1 title for pandoc --metadata title ----------------------------
extract_title() {
    # First ATX H1 ("# Title") in the file, stripped; fallback to basename.
    md_file="$1"
    title_line=$(grep -m1 '^# ' "$md_file" 2>/dev/null || true)
    if [ -n "$title_line" ]; then
        # strip leading "# " and any trailing whitespace
        printf '%s' "${title_line#\# }" | sed 's/[[:space:]]*$//'
    else
        basename "$md_file" .md
    fi
}

# --- Counters ----------------------------------------------------------------
GENERATED=0
SKIPPED=0
FAILED=0
FAILED_LIST=""

# --- Walk + convert ----------------------------------------------------------
# Collect the file list into a temp file (newline-delimited). Spec doc paths
# contain no newlines, so this is safe and parses cleanly under sh -n + bash -n
# (no bash-only process substitution / NUL handling required).
FILELIST=$(mktemp 2>/dev/null || echo "/tmp/_sync_md_list.$$")
find "$WALK_ROOT" -type f -name '*.md' >"$FILELIST" 2>/dev/null

while IFS= read -r md; do
    [ -n "$md" ] || continue
    base="${md%.md}"
    html="$base.html"
    pdf="$base.pdf"

    need_html=1
    need_pdf=1
    if [ "$FORCE" -eq 0 ]; then
        if is_newer_than "$html" "$md"; then need_html=0; fi
        if is_newer_than "$pdf" "$md"; then need_pdf=0; fi
    fi

    if [ "$need_html" -eq 0 ] && [ "$need_pdf" -eq 0 ]; then
        SKIPPED=$((SKIPPED + 1))
        continue
    fi

    title=$(extract_title "$md")
    file_failed=0

    # --- HTML via pandoc -----------------------------------------------------
    if [ "$need_html" -eq 1 ]; then
        if run_with_timeout 60 pandoc "$md" \
                --standalone \
                --embed-resources \
                --metadata title="$title" \
                -f gfm \
                -o "$html" 2>/tmp/_sync_md_err.$$; then
            : # ok
        else
            file_failed=1
            echo "FAIL (html): $md" >&2
            sed 's/^/    /' /tmp/_sync_md_err.$$ >&2 2>/dev/null || true
        fi
    fi

    # --- PDF via weasyprint (from the HTML we just made, if present) ---------
    if [ "$need_pdf" -eq 1 ]; then
        if [ -e "$html" ]; then
            if run_with_timeout 60 weasyprint "$html" "$pdf" 2>/tmp/_sync_md_err.$$; then
                : # ok
            else
                file_failed=1
                echo "FAIL (pdf): $md" >&2
                sed 's/^/    /' /tmp/_sync_md_err.$$ >&2 2>/dev/null || true
            fi
        else
            file_failed=1
            echo "FAIL (pdf): $md (no HTML to render from)" >&2
        fi
    fi

    if [ "$file_failed" -eq 1 ]; then
        FAILED=$((FAILED + 1))
        FAILED_LIST="$FAILED_LIST
  $md"
    else
        GENERATED=$((GENERATED + 1))
    fi
done <"$FILELIST"

rm -f "$FILELIST" 2>/dev/null || true
rm -f /tmp/_sync_md_err.$$ 2>/dev/null || true

# --- Summary -----------------------------------------------------------------
echo ""
echo "generated=$GENERATED skipped=$SKIPPED failed=$FAILED"
if [ "$FAILED" -gt 0 ]; then
    echo "Failed files:$FAILED_LIST" >&2
    exit 1
fi
exit 0

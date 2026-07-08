#!/usr/bin/env python3
# ============================================================================
# render_mermaid_blocks.py
# ============================================================================
#
# Purpose:
#   Pre-process a Markdown file so that every ```mermaid fenced block is
#   replaced by a reference to a rendered SVG image, BEFORE the file is handed
#   to pandoc. This closes the Constitution §11.4.168 gap where pandoc (which
#   has no mermaid filter) and weasyprint (which cannot execute JavaScript)
#   would otherwise emit the diagram *source* as literal text into the HTML and
#   PDF exports.
#
#   Diagrams are rendered with mermaid-cli (`mmdc`) to PNG at scale 3 using
#   mermaid's default config (see scripts/testing/mermaid.config.json).
#   Rationale (proven empirically, §11.4.6): weasyprint cannot execute JS and
#   does NOT render SVG <foreignObject> (mermaid's default text container), so a
#   default-config SVG would lose all label text in the PDF. Rendering to PNG
#   lets Chromium (inside mmdc) do the full rendering — every mermaid label
#   feature including <br/> line breaks — and weasyprint simply embeds the
#   raster, which it can never fail to display. OCR of the rendered PDF page
#   confirms readability (the §11.4.168 validation oracle). Proven across all 8
#   diagram types used by the spec (flowchart, sequenceDiagram, stateDiagram-v2,
#   graph, erDiagram, gantt, mindmap, C4Container).
#
# Usage:
#   render_mermaid_blocks.py <input.md> <output.md> <svg_dir> [<cache_dir>]
#
#   <input.md>   Source Markdown (never modified).
#   <output.md>  Where the pre-processed Markdown is written. Each mermaid
#                fence is replaced by `![diagram](<abs-svg-path>)`. If the input
#                has no mermaid fences, the input is copied verbatim.
#   <svg_dir>    Directory to write per-block PNGs into (created if absent).
#   <cache_dir>  Optional. A content-addressed cache. A diagram whose
#                sha256(source + config) already has a PNG in the cache is
#                reused instead of re-rendered — making incremental regens of
#                an otherwise-unchanged spec near-instant.
#
# Outputs:
#   - The pre-processed Markdown at <output.md>.
#   - One PNG per mermaid block under <svg_dir> (or reused from <cache_dir>).
#   - On stdout: "mermaid: rendered=N reused=M failed=K".
#
# Exit codes:
#   0  every mermaid block rendered (or reused) successfully.
#   1  at least one block FAILED to render (mmdc error / syntax error). The
#      failing block(s) are listed on stderr. A render failure is surfaced as
#      a hard error — NEVER silently passed through as raw source, which would
#      re-introduce the exact §11.4.168 bluff this script exists to prevent.
#   2  bad invocation / missing mmdc.
#
# Dependencies:
#   - python3 (stdlib only: re, sys, os, hashlib, shutil, subprocess, pathlib)
#   - mmdc (mermaid-cli) on PATH; config at scripts/testing/mermaid.config.json
#
# Cross-references:
#   - Constitution §11.4.168 (exported-document visual validation mandate)
#   - Constitution §11.4.65  (universal Markdown export)
#   - Constitution §11.4.18  (script documentation mandate)
#   - Companion guide: docs/scripts/render_mermaid_blocks.md
#   - Driver: scripts/testing/sync_all_markdown_exports.sh
# ============================================================================

import hashlib
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

CONFIG = Path(__file__).resolve().parent / "mermaid.config.json"

# A fenced mermaid block: an opening fence line whose info string is exactly
# `mermaid` (optionally surrounded by whitespace), through the next closing
# fence line of the same fence character run. Matched line-wise to stay robust.
OPEN_RE = re.compile(r'^(\s*)(`{3,}|~{3,})\s*mermaid\s*$')

# Some hosts ship a Puppeteer-bundled Chromium build (cached under
# ~/.cache/puppeteer) that fails to launch in this environment (observed:
# "Failed to launch the browser process!" with no further detail from mmdc,
# root-caused via a direct `puppeteer.launch({dumpio: true})` probe to a
# headless-EGL/X-display warning storm that is cosmetic — the REAL blocker
# was the specific cached Chromium build, not the sandbox or the display).
# A system-installed Chromium/Chrome (its shared-library deps resolved by
# the OS package manager) reliably launches instead. Never hardcode a path —
# discover it fresh each run via PATH, and only opt in when one is found, so
# hosts where the bundled Puppeteer Chrome DOES work are left untouched.
_SYSTEM_BROWSER_CANDIDATES = (
    "chromium", "chromium-browser", "google-chrome", "google-chrome-stable",
)


def _puppeteer_config_path() -> str | None:
    """Return a path to a temp puppeteer config pointing at a discovered
    system Chromium/Chrome with --no-sandbox, or None if none is found (in
    which case mmdc falls back to its own default Puppeteer resolution)."""
    exe = next((p for name in _SYSTEM_BROWSER_CANDIDATES if (p := shutil.which(name))), None)
    if not exe:
        return None
    cfg = {"executablePath": exe, "args": ["--no-sandbox", "--disable-setuid-sandbox"]}
    fd, path = tempfile.mkstemp(prefix="helixvpn-puppeteer-", suffix=".json")
    with os.fdopen(fd, "w", encoding="utf-8") as f:
        json.dump(cfg, f)
    return path


_PUPPETEER_CONFIG = _puppeteer_config_path()


def _config_bytes() -> bytes:
    try:
        return CONFIG.read_bytes()
    except OSError:
        return b""


def render_one(source: str, out_png: Path) -> bool:
    """Render a single mermaid `source` string to `out_png` (PNG @ scale 3). Returns True on success."""
    tmp_mmd = out_png.with_suffix(".mmd")
    tmp_mmd.write_text(source, encoding="utf-8")
    cmd = ["mmdc", "-i", str(tmp_mmd), "-o", str(out_png), "-b", "white", "-s", "3"]
    if CONFIG.exists():
        cmd += ["-c", str(CONFIG)]
    if _PUPPETEER_CONFIG:
        cmd += ["-p", _PUPPETEER_CONFIG]
    try:
        proc = subprocess.run(cmd, capture_output=True, text=True, timeout=120)
    except (subprocess.TimeoutExpired, FileNotFoundError) as exc:
        sys.stderr.write(f"    mmdc error: {exc}\n")
        return False
    finally:
        try:
            tmp_mmd.unlink()
        except OSError:
            pass
    if proc.returncode != 0 or not out_png.exists() or out_png.stat().st_size == 0:
        sys.stderr.write(proc.stderr.strip()[:800] + "\n" if proc.stderr else "    (no stderr)\n")
        return False
    return True


def main(argv):
    if len(argv) < 4:
        sys.stderr.write(__doc__ or "usage: render_mermaid_blocks.py <in.md> <out.md> <svg_dir> [cache_dir]\n")
        return 2
    if shutil.which("mmdc") is None:
        sys.stderr.write("ERROR: mmdc (mermaid-cli) not found on PATH\n")
        return 2

    in_md, out_md, svg_dir = Path(argv[1]), Path(argv[2]), Path(argv[3])
    cache_dir = Path(argv[4]) if len(argv) > 4 else None
    svg_dir.mkdir(parents=True, exist_ok=True)
    if cache_dir:
        cache_dir.mkdir(parents=True, exist_ok=True)

    lines = in_md.read_text(encoding="utf-8").splitlines(keepends=False)
    cfg = _config_bytes()
    out_lines, rendered, reused, failed, idx = [], 0, 0, 0, 0
    i, n = 0, len(lines)

    while i < n:
        m = OPEN_RE.match(lines[i])
        if not m:
            out_lines.append(lines[i])
            i += 1
            continue
        indent, fence = m.group(1), m.group(2)
        # Gather block body until the matching closing fence.
        close_re = re.compile(r'^\s*' + re.escape(fence[0]) + '{' + str(len(fence)) + r',}\s*$')
        j = i + 1
        body = []
        while j < n and not close_re.match(lines[j]):
            body.append(lines[j])
            j += 1
        # j is the closing fence (or EOF). Render the block.
        source = "\n".join(body) + "\n"
        idx += 1
        key = hashlib.sha256(source.encode("utf-8") + cfg).hexdigest()
        target = svg_dir / f"diagram_{idx:03d}_{key[:12]}.png"
        cached = (cache_dir / f"{key}.png") if cache_dir else None

        ok = False
        if cached and cached.exists() and cached.stat().st_size > 0:
            shutil.copyfile(cached, target)
            reused += 1
            ok = True
        elif render_one(source, target):
            rendered += 1
            ok = True
            if cached:
                shutil.copyfile(target, cached)
        else:
            failed += 1
            sys.stderr.write(f"  FAILED mermaid block #{idx} in {in_md}\n")

        if ok:
            out_lines.append(f"{indent}![diagram]({target.resolve()})")
        else:
            # Surface the failure visibly in the output too, but the non-zero
            # exit is the authoritative signal the caller must act on.
            out_lines.append(f"{indent}**[MERMAID RENDER FAILED — block #{idx}]**")
        i = j + 1 if j < n else j

    out_md.write_text("\n".join(out_lines) + "\n", encoding="utf-8")
    print(f"mermaid: rendered={rendered} reused={reused} failed={failed}")
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))

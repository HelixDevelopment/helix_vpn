#!/usr/bin/env python3
# ============================================================================
# validate_mermaid_blocks.py
# ============================================================================
#
# Purpose:
#   Render-validate every ```mermaid fenced block under a path (file or dir)
#   using mermaid-cli (`mmdc`) with the project mermaid config. A block that
#   fails to render is a §11.4.168 defect: under the export pipeline it would
#   ship as RAW SOURCE in the HTML/PDF instead of an image. This is the
#   anti-bluff gate that proves every diagram in the spec actually renders.
#
# Usage:
#   validate_mermaid_blocks.py [PATH]
#     PATH   File or directory to scan. Default: docs/research/mvp/final.
#
# Outputs (stdout):
#   - One "FAIL <file>:<line> [<type>] -> <error>" line per broken diagram.
#   - A summary "mermaid-validate: total=N ok=M failed=K".
#
# Exit codes:
#   0  every mermaid block rendered successfully.
#   1  at least one block failed to render (defect list on stdout).
#   2  mmdc not on PATH / bad invocation.
#
# Dependencies:
#   - python3 (stdlib), mmdc (mermaid-cli), scripts/testing/mermaid.config.json
#
# Cross-references:
#   - Constitution §11.4.168 (exported-document visual validation mandate)
#   - Constitution §11.4.118 (enumerated discovery coverage)
#   - Renderer: scripts/testing/render_mermaid_blocks.py
#   - Companion guide: docs/scripts/validate_mermaid_blocks.md
# ============================================================================

import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

ROOT_DEFAULT = "docs/research/mvp/final"
CONFIG = Path(__file__).resolve().parent / "mermaid.config.json"
OPEN_RE = re.compile(r'^(\s*)(`{3,}|~{3,})\s*mermaid\s*$')

# Mirror the fallback in render_mermaid_blocks.py: Puppeteer's bundled Chromium
# often fails to launch in headless/server environments, while a system Chromium
# with --no-sandbox works. Discover it on PATH and pass it to mmdc via a temp
# puppeteer config. If none is found, mmdc uses its default resolution.
_SYSTEM_BROWSER_CANDIDATES = (
    "chromium", "chromium-browser", "google-chrome", "google-chrome-stable",
)


def _puppeteer_config_path() -> str | None:
    exe = next((p for name in _SYSTEM_BROWSER_CANDIDATES if (p := shutil.which(name))), None)
    if not exe:
        return None
    cfg = {"executablePath": exe, "args": ["--no-sandbox", "--disable-setuid-sandbox"]}
    fd, path = tempfile.mkstemp(prefix="helixvpn-puppeteer-", suffix=".json")
    with os.fdopen(fd, "w", encoding="utf-8") as f:
        json.dump(cfg, f)
    return path


_PUPPETEER_CONFIG = _puppeteer_config_path()


def iter_blocks(md_path: Path):
    lines = md_path.read_text(encoding="utf-8").splitlines()
    i, n = 0, len(lines)
    while i < n:
        m = OPEN_RE.match(lines[i])
        if not m:
            i += 1
            continue
        fence = m.group(2)
        close = re.compile(r'^\s*' + re.escape(fence[0]) + '{' + str(len(fence)) + r',}\s*$')
        start = i + 1  # 1-based line of the content's first line
        body, j = [], i + 1
        while j < n and not close.match(lines[j]):
            body.append(lines[j])
            j += 1
        yield (start + 1, body)
        i = j + 1


def render_ok(source: str):
    with tempfile.NamedTemporaryFile("w", suffix=".mmd", delete=False) as f:
        f.write(source)
        mmd = f.name
    png = mmd + ".png"
    cmd = ["mmdc", "-i", mmd, "-o", png, "-b", "white", "-s", "3"]
    if CONFIG.exists():
        cmd += ["-c", str(CONFIG)]
    if _PUPPETEER_CONFIG:
        cmd += ["-p", _PUPPETEER_CONFIG]
    try:
        proc = subprocess.run(cmd, capture_output=True, text=True, timeout=180)
    except Exception as exc:  # noqa: BLE001
        return False, str(exc)[:160]
    finally:
        for p in (mmd, png):
            try:
                os.unlink(p)
            except OSError:
                pass
    ok = proc.returncode == 0
    err = ""
    if not ok:
        cand = [l for l in proc.stderr.splitlines() if "rror" in l or "Expecting" in l]
        err = (cand[0] if cand else (proc.stderr.splitlines() or ["?"])[0])[:160]
    return ok, err


def main(argv):
    if shutil.which("mmdc") is None:
        sys.stderr.write("ERROR: mmdc (mermaid-cli) not found on PATH\n")
        return 2
    root = Path(argv[1]) if len(argv) > 1 else Path(ROOT_DEFAULT)
    files = [root] if root.is_file() else sorted(root.rglob("*.md"))
    total = ok = failed = 0
    for md in files:
        for line, body in iter_blocks(md):
            total += 1
            src = "\n".join(body) + "\n"
            good, err = render_ok(src)
            if good:
                ok += 1
            else:
                failed += 1
                dtype = (body[0].strip().split() or ["?"])[0] if body else "?"
                print(f"FAIL {md}:{line} [{dtype}] -> {err}")
    print(f"mermaid-validate: total={total} ok={ok} failed={failed}")
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))

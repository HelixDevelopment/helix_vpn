# Helix VPN

Go-based VPN project. Early scaffolding only — no VPN implementation exists yet.

## Status

**Implementation not started.** The repository contains engineering-governance
scaffolding (see below) but no VPN application code, protocols, tunneling logic,
CLI, or configuration have been written.

## Engineering governance

Helix VPN inherits the **Helix Constitution** via a Git submodule at `constitution/`.

**Submodule remote:** `git@github.com:HelixDevelopment/HelixConstitution.git` (branch `main`)

The constitution enforces a mandatory anti-bluff covenant and other universal
engineering rules. Three parent files wire in the inheritance:

| File | Role |
|------|------|
| `CLAUDE.md` | Claude Code agent rules — inherits `constitution/CLAUDE.md` |
| `AGENTS.md` | Agent rules — inherits `constitution/AGENTS.md` |
| `docs/guides/HELIX_VPN_CONSTITUTION.md` | Project constitution extending `constitution/Constitution.md` |

### Running the pre-build gate

A gate script checks five constitution-inheritance invariants before any build:

```bash
bash tests/pre_build_verification.sh
```

All five invariants must report `PASS` and the final line must read `STATUS: PASS`.

### Running the anti-bluff mutation proof

To confirm the gate is discriminating (not a bluff gate that always passes):

```bash
bash scripts/testing/meta_test_false_positive_proof.sh
```

This mutates `constitution/Constitution.md`, asserts the gate fails, then restores
the file from a backup and verifies byte-identical restore via SHA-256 checksum.

### Setup after clone (§11.4.36)

After cloning this repository, initialise the constitution submodule and opt in to
the local pre-commit enforcement gate:

```bash
# 1. Initialise the submodule
git submodule update --init --recursive

# 2. Opt in to the pre-commit hook (local enforcement — see §11.4.75)
bash scripts/install_git_hooks.sh
```

> **CI is intentionally DISABLED** per Constitution §11.4.156.
> `.github/workflows/constitution.yml` has been renamed to
> `.github/workflows/constitution.yml.disabled-local-only` and is not executed
> by any remote runner. Enforcement is local only: the pre-commit hook (installed
> via `scripts/install_git_hooks.sh`) runs the gate before every commit, and
> `scripts/commit_all.sh` pushes to all configured upstreams (§2.1).

## Session resumption

See `docs/CONTINUATION.md` for a standing session-resumption file with the current
branch, submodule SHA, completed work, remaining work, and re-run commands.

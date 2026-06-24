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

### Initialising the submodule after a fresh clone

```bash
git submodule update --init --recursive
```

## Session resumption

See `docs/CONTINUATION.md` for a standing session-resumption file with the current
branch, submodule SHA, completed work, remaining work, and re-run commands.

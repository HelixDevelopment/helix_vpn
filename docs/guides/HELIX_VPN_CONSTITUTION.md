# Helix VPN Constitution

This constitution **extends** the Helix Universal Constitution at
`constitution/Constitution.md`. All clauses there apply unless
explicitly overridden below with an explicit `Override §X.Y` section.

## Overrides

_None._

## Updating the constitution submodule (§11.4.26)

When a new version of the Helix Universal Constitution is published, follow this
7-step pipeline exactly. Never skip or reorder steps.

### Step 1 — Fetch and pull all remotes inside the submodule

```bash
cd constitution
git fetch --all
git pull --ff-only origin main   # fast-forward only; never force inside the submodule
cd ..
```

### Step 2 — Apply the change

Review the diff before proceeding:

```bash
git diff constitution/Constitution.md | head -80
```

If the update is as expected, move on. If you see unexpected changes, abort and
investigate the upstream.

### Step 3 — Validate via post-update script

Run the post-update hook and full rule verification in one command:

```bash
bash scripts/post_constitution_update.sh
```

`scripts/post_constitution_update.sh` invokes the post-update hook and then calls
`scripts/verify-all-constitution-rules.sh`. Both must exit 0 before continuing.

### Step 4 — Commit and push to ALL upstreams

Use `scripts/commit_all.sh` to satisfy the §2.1 multi-upstream push requirement.
Do **not** use plain `git push`; it pushes to only one remote.

```bash
bash scripts/commit_all.sh "chore(constitution): bump submodule to <short-SHA>"
```

### Step 5 — Conflict resolution (if step 4 fails)

If a push is rejected due to a diverged remote:

1. **Never force-push** — this is prohibited by §11.4.26.
2. Fetch the conflicting remote: `git fetch <remote>`
3. Rebase locally: `git rebase <remote>/<branch>`
4. Re-run `scripts/verify-all-constitution-rules.sh` to confirm validity after rebase.
5. Retry `scripts/commit_all.sh`.

### Step 6 — Post-merge validation via cascade verifier

After the push succeeds, run the full test suite to confirm nothing broke:

```bash
bash test_all.sh
# expected final line: OVERALL STATUS: PASS
```

### Step 7 — Bump the submodule pointer in the consuming project

The bumped pointer must be committed in the **same** commit as any consuming-project
changes that depend on the new constitution content. `scripts/commit_all.sh` in
step 4 handles this when called from the repo root after `cd constitution && git pull`.

Verify the pointer is updated:

```bash
git submodule status
# Should show the new SHA without a leading '+'
```

---

## Project-specific clauses

_None yet — Helix VPN is in early scaffolding (Go). Project-specific
clauses (service names, ports, protocol versions, numeric thresholds,
dependency workarounds) live here, never in the constitution
submodule._

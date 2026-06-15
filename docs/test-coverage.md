# LLMOrchestrator — Symbol→Test Coverage Ledger

**Round:** 275 (deep-doc + Challenge enrichment, mirror round-220 template)
**Generated:** 2026-05-19
**Mandate:** CONST-048 (full-automation coverage) + CONST-050 (no-fakes-beyond-unit-tests) + CONST-035 / Article XI §11.9 (anti-bluff)

> Verbatim 2026-05-19 operator mandate (preserved per CONST-049 §11.4.17):
>
> "all existing tests and Challenges do work in anti-bluff manner — they
> MUST confirm that all tested codebase really works as expected! We had
> been in position that all tests do execute with success and all
> Challenges as well, but in reality the most of the features does not
> work and can't be used! This MUST NOT be the case and execution of
> tests and Challenges MUST guarantee the quality, the completition and
> full usability by end users of the product!"

## Purpose

This ledger maps every exported symbol in the LLMOrchestrator submodule to
the test or Challenge that exercises it with runtime evidence. A row
without runtime evidence is a §11.4 PASS-bluff regardless of how green the
green column reads.

## Convention

- **Layer:** `unit` (mocks allowed per CONST-050(A)), `integration` (real
  collaborators, no mocks), `challenge` (out-of-process bash wrapper, real
  subprocess + real disk + real wire).
- **Evidence:** anti-bluff artefact emitted during the test (file path,
  stdout substring, captured JSON). A green PASS column without an
  evidence column is a critical defect.
- **Mutation:** for every gate, a paired meta-test mutation (§1.1) that
  inverts the polarity of one invariant. The mutation MUST flip green→red.

## Coverage Ledger

| Package                 | Symbol                                     | Layer       | Test / Challenge                                                                   | Evidence                                                       | Paired Mutation                                                |
|-------------------------|--------------------------------------------|-------------|------------------------------------------------------------------------------------|----------------------------------------------------------------|----------------------------------------------------------------|
| `pkg/parser`            | `DefaultParser.Parse`                      | unit        | `pkg/parser/parser_test.go::TestDefaultParser_Parse_*`                             | parsed JSON struct, action slice                               | `parser_security_test.go` size + traversal mutations           |
| `pkg/parser`            | `DefaultParser.Parse` (empty)              | unit        | `pkg/parser/parser_test.go::TestParse_Empty`                                       | `ErrEmptyInput` returned                                       | round-275 runner flips polarity under `LLMORCH_MUTATE_RUNNER=1`|
| `pkg/parser`            | `DefaultParser.Parse` (5-locale prompts)   | challenge   | `challenges/runner/main.go` + `challenges/llmorchestrator_describe_challenge.sh`   | `parser.Parse.<locale>.action  PASS` line + actions=[...] dump | `challenges/llmorchestrator_describe_challenge.sh mutate` → 99 |
| `pkg/parser`            | `DefaultParser.ExtractJSON`                | unit        | `parser_test.go::TestExtractJSON_*`                                                | unmarshalled `map[string]any`                                  | fuzz: `parser_fuzz_test.go`                                    |
| `pkg/parser`            | `DefaultParser.ExtractActions`             | unit        | `parser_test.go::TestExtractActions_*`                                             | non-empty `[]agent.Action`                                     | regex + JSON injection in `parser_security_test.go`            |
| `pkg/parser`            | `DefaultParser.ExtractIssues`              | unit        | `parser_test.go::TestExtractIssues_*`                                              | non-empty `[]agent.Issue`                                      | text/JSON dual-path mutations                                  |
| `pkg/protocol`          | `PipeMessage` JSON encoding                | challenge   | `challenges/runner/main.go::Invariant 3` + `pipe_test.go::TestPipeMessage_*`       | round-trip bytes per locale; `Content == expect_pipe_content`  | mutation strips `Type` field → unmarshal mismatches            |
| `pkg/protocol`          | `MessageTypePrompt` constant               | challenge   | `challenges/runner/main.go::Invariant 3`                                           | `back.Type == prompt` per locale                               | type-rename mutation surfaces via test fail                    |
| `pkg/protocol`          | `NewFileTransport`                         | challenge   | `challenges/runner/main.go::Invariant 4` + `file_test.go::TestNewFileTransport_*`  | inbox/outbox/shared subdirs created at `tmp` path              | empty-sessionDir mutation triggers error path                  |
| `pkg/protocol`          | `FileTransport.WriteToInbox`               | challenge   | `challenges/runner/main.go::Invariant 4` + `file_test.go::TestWriteToInbox_*`      | `<ID>.json` lands in inbox dir                                 | invalid path traversal triggers `ErrPathTraversal`             |
| `pkg/protocol`          | `FileTransport.ReadFromInbox`              | challenge   | `challenges/runner/main.go::Invariant 4` + `file_test.go::TestReadFromInbox_*`     | `messages_in_inbox=1` per locale, content equality             | missing-dir mutation returns sentinel                          |
| `pkg/protocol`          | `FileTransport.WriteSharedFile`            | unit        | `file_test.go::TestSharedFile_*`                                                   | bytes round-trip; path traversal rejected                      | `..` segment mutation surfaces `ErrPathTraversal`              |
| `pkg/protocol`          | `validatePath` (private, exercised)        | unit+integ  | `file_test.go::TestPathTraversal_*` + `protocol_integration_test.go`               | rejection events captured                                      | absolute-path mutation rejected                                |
| `pkg/protocol`          | `PipeTransport.Send` / `Recv`              | unit        | `pkg/protocol/pipe_test.go`                                                        | JSON-lines parsed back to `PipeMessage`                        | stdin/stdout swap mutation surfaces decode error               |
| `pkg/i18n`              | `Translator` interface                     | unit        | `pkg/i18n/translator_test.go::TestTranslatorContract_*`                            | contract assertions on `NoopTranslator`                        | nil-arg mutation                                               |
| `pkg/i18n`              | `NoopTranslator.T`                         | challenge   | `challenges/runner/main.go::Invariant 5`                                           | `got == expectMessageID` per locale                            | strip verbatim contract → returns empty → gate FAIL            |
| `pkg/i18n`              | `NoopTranslator.TPlural`                   | challenge   | `challenges/runner/main.go::Invariant 5 (plural variant)`                          | count-aware id still verbatim per locale                       | count-mutation does not change output                          |
| `pkg/i18n`              | `SetPkgTranslator(nil)` + `Pkg()`          | challenge   | `challenges/runner/main.go::Invariant 5 (Pkg.reset_to_noop)`                       | `pkg.T("round_275_pkg_probe") == "round_275_pkg_probe"`        | nil-handling mutation would yield empty string                 |
| `pkg/agent`             | `Agent` interface                          | unit+integ  | `pkg/agent/agent_test.go` + per-adapter `*_agent_test.go`                          | mock pool round-trip; capability matching                      | capability-bitmask mutations                                   |
| `pkg/agent`             | `AgentPool.Acquire`                        | unit        | `pkg/agent/pool_test.go::TestPool_Acquire_*`                                       | per-requirement match log                                      | stress: `pool_stress_test.go`                                  |
| `pkg/agent`             | `SimplePool` round-robin                   | unit        | `pkg/agent/simple_pool_test.go`                                                    | ordered acquire/release sequence                               | concurrency-race mutation                                      |
| `pkg/agent`             | `HealthMonitor` circuit-breaker            | unit        | `pkg/agent/health_test.go::TestHealthMonitor_Trip`                                 | trip after 3 consecutive failures                              | threshold mutation → trips at 1                                |
| `pkg/agent`             | `ClaudeCodeAgent.Send`                     | unit        | `pkg/agent/claudecode_agent_test.go`                                               | mock-pool stdin/stdout transcript                              | env-not-set mutation                                           |
| `pkg/agent`             | `GeminiAgent.Send`                         | unit        | `pkg/agent/gemini_agent_test.go`                                                   | mock-pool stdin/stdout transcript                              | env-not-set mutation                                           |
| `pkg/agent`             | `JunieAgent.Send`                          | unit        | `pkg/agent/junie_agent_test.go`                                                    | mock-pool stdin/stdout transcript                              | env-not-set mutation                                           |
| `pkg/agent`             | `OpenCodeAgent.Send`                       | unit        | `pkg/agent/opencode_agent_test.go`                                                 | mock-pool stdin/stdout transcript                              | env-not-set mutation                                           |
| `pkg/agent`             | `QwenCodeAgent.Send`                       | unit        | `pkg/agent/qwencode_agent_test.go`                                                 | mock-pool stdin/stdout transcript                              | env-not-set mutation                                           |
| `pkg/agent`             | `MultiPool.Add` / `Pick`                   | unit        | `pkg/agent/mock_pool_test.go`                                                      | capability dispatch table                                      | identity mutation in `Pick`                                    |
| `pkg/adapter`           | `BaseAdapter` lifecycle                    | unit+integ  | `pkg/adapter/adapter_test.go` + `adapter_integration_test.go`                      | process spawn/stop transcript                                  | env-not-set mutation                                           |
| `pkg/adapter`           | `OpenCodeHeadless.Run`                     | unit        | `pkg/adapter/opencode_headless_test.go`                                            | parsed JSON-lines back to `Response`                           | broken-pipe mutation                                           |
| `pkg/config`            | `FromEnv`                                  | unit        | `pkg/config/config_test.go::TestFromEnv_*`                                         | parsed struct values per env scenario                          | missing-env mutation                                           |
| `cmd/orchestrator`      | `main` entry                               | integ       | `automation_test.go::TestOrchestratorCmd_Compile`                                  | compile + smoke run                                            | broken-flag mutation                                           |
| Chaos                   | failure injection across pool              | challenge   | `challenges/scripts/chaos_failure_injection_challenge.sh`                          | circuit-breaker trip + recovery log                            | inject-zero-failures fails the gate                            |
| DDoS                    | health-endpoint flood                      | challenge   | `challenges/scripts/ddos_health_flood_challenge.sh`                                | RPS sustained, no GC death                                     | flood-rate=0 → no failure detected                             |
| Scaling                 | horizontal pool growth                     | challenge   | `challenges/scripts/scaling_horizontal_challenge.sh`                               | N→2N capacity transcript                                       | scale-step=0 → no growth detected                              |
| Stress                  | sustained-load mix                         | challenge   | `challenges/scripts/stress_sustained_load_challenge.sh`                            | p95 latency under cap                                          | load-mix=empty → no measurement                                |
| UI                      | terminal interaction                       | challenge   | `challenges/scripts/ui_terminal_interaction_challenge.sh`                          | stdin/stdout transcript                                        | tty-detached mutation                                          |
| UX                      | end-to-end flow                            | challenge   | `challenges/scripts/ux_end_to_end_flow_challenge.sh`                               | full-cycle log                                                 | step-skip mutation                                             |

## Invariant Floor (per CONST-048)

For every row above, six invariants are asserted:

1. **Anti-bluff posture** — captured runtime evidence per §11.4.
2. **Proof of working capability** — end-to-end on the documented topology.
3. **Implementation matches docs** — README + USER_GUIDE + this ledger reflect actual API.
4. **No open issues** — `docs/Issues.md` empty for this row or row marked `OPERATOR-BLOCKED` per §11.4.21.
5. **Documentation in sync** — `.md` + `.html` + `.pdf` mtimes lockstep per §11.4.12 / §11.4.53.
6. **Four-layer test floor** — pre-build + post-build + runtime + paired mutation.

## How to Re-validate

```bash
# Round-275 Challenge (this round's deliverable)
cd dependencies/HelixDevelopment/LLMOrchestrator
bash challenges/llmorchestrator_describe_challenge.sh normal   # → exit 0
bash challenges/llmorchestrator_describe_challenge.sh mutate   # → exit 99

# Full unit + integration with race detector
go test -race -count=1 ./...
```

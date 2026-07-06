# 10 — Design System

**Revision:** 1
**Last modified:** 2026-07-05T14:20:00Z
**Status:** Draft — consolidated from `docs/design/README.md` and `v10-design/*`.

---

## 1. Position

This section defines the **single source of design truth** for every HelixVPN user-facing surface: the Client (8 platforms), the Console (admin web), and the Connector (appliance UI).

It is a **spec-only** consolidation. The authoritative design files live under `docs/design/` and the Volume-10 nano-detail docs; this README provides the navigable entry point and the architectural invariants.

---

## 2. Decoupled, reusable `helix_design`

The design system is a **standalone own-org submodule** at `vasic-digital/helix_design`, consumed as a flat submodule from the HelixVPN root. It is project-not-aware: HelixVPN brand values are injected as a swappable preset, never hardcoded inside the submodule.

| Invariant | Why |
|---|---|
| Project-not-aware | Reusable across unrelated projects per §11.4.74 |
| Flat layout | No nested own-org submodule chains per §11.4.28(C) |
| Equal codebase | Same test/doc/push discipline as the parent repo |
| Catalogue-first | Missing patterns are extended upstream, never forked in-project |

The Flutter `helix-ui` workspace re-exports the generated Dart package; native shims consume the polyglot forms directly.

---

## 3. Token tiers

There is exactly one authored token source — JSON files in `helix_design/tokens/**`. Every consumable form is generated from it.

| Tier | Role | Example |
|---|---|---|
| **Primitive** | Raw, theme-agnostic values | `color.primitive.helix.500 = #3D5AF1` |
| **Semantic** | Intent, light + dark fork | `color.semantic.state.connected.fill` |
| **Component** | Widget slot, references semantic only | `comp.connectButton.bg.connected` |

Rules:

- Primitive tokens are literals; semantic tokens reference primitives; component tokens reference semantic tokens only.
- Every semantic color/elevation token defines **both** `light` and `dark` values.
- Hand-editing a generated form (CSS, Dart, etc.) is a drift defect.

---

## 4. Export forms

The canonical JSON source is emitted into every target form so no platform re-types a color by hand.

| Target | Output | Consumer |
|---|---|---|
| CSS | `dist/css/helix.css` | Console web, OpenDesign round-trip |
| Dart | `dist/dart/helix_tokens.dart` + `ThemeExtension` | Flutter `helix-ui` |
| SwiftUI | `dist/swift/HelixTokens.swift` | iOS/macOS NE config UI |
| Jetpack Compose | `dist/compose/HelixTokens.kt` | Android quick-settings / notifications |
| ArkTS | `dist/arkts/helix_tokens.ets` | HarmonyOS native surfaces |
| C / Qt | `dist/cqt/helix_tokens.h` + `.qml` | Aurora OS |

`dist/**` is a build-derivative; the source (`tokens/**`, `presets/**`, `export/**`) is tracked.

---

## 5. Component & screen index

The component library and screen wireframes are catalogued under `docs/design/`:

| Area | Docs |
|---|---|
| Component specs | [`docs/design/components/README.md`](../../../../../design/components/README.md) — desktop, mobile, Aurora, web |
| Screen wireframes | [`docs/design/screens/README.md`](../../../../../design/screens/README.md) — every screen across all platforms |
| Interaction patterns | [`docs/design/interaction/README.md`](../../../../../design/interaction/README.md) — animation, gestures, accessibility |
| Assets & icons | [`docs/design/exports/`](../../../../../design/exports/) — PNG/PDF component and screen exports |

Signature components include: `ConnectButton`, `StatusChip`, `ExitPicker`, `ShieldIndicator`, `AdaptiveScaffold`, and the connection-state palette mapped to the 7-variant FFI `TunnelStatus`.

---

## 6. OpenDesign integration

OpenDesign (`submodules/open-design`) is the mandatory design-and-refinement system per §11.4.162. `helix_design` authors an OpenDesign-native design system under `helix_design/opendesign/helix/` (`DESIGN.md` + `tokens.css` + `manifest.json` + `components.html`).

- OpenDesign owns **authoring, refinement, and the CSS/markdown design-system form**.
- `helix_design` owns the **canonical JSON source and the polyglot distribution**.
- If OpenDesign lacks a needed pattern, the fix is a PR upstream to `nexu-io/open-design`, recorded with a `Catalogue-Check: extend nexu-io/open-design@<sha>` line.

Current status:

- ✅ OpenDesign submodule built and daemon starts.
- ✅ HelixVPN design system imports successfully.
- ⚠️ Token contract validation reports `needs-rebuild` (low source-backed A1 coverage).
- ❌ No built-in Figma/Sketch/Adobe/XD/Penpot exporters; only Figma import is available.

For full validation results, see [`docs/reviews/mvp-final/findings/phase2-opendesign-report.md`](../../../../../reviews/mvp-final/findings/phase2-opendesign-report.md).

---

## 7. Quality gates owned here

| Gate | Asserts |
|---|---|
| `DS-DRIFT` | Generated forms are byte-identical to a fresh `pnpm export` |
| `DS-LIGHT-DARK-COMPLETE` | Every semantic color/elevation token defines both themes |
| `DS-CONTRAST` | Every `on-X / X` pair clears WCAG AA in light and dark |
| `DS-NO-OVERLAP` | Golden screenshots show no overlapping elements / labels |
| `DS-VISUAL-REGRESSION` | Per-component golden screenshots match baseline |
| `DS-DECOUPLING` | A non-HelixVPN preset builds green with zero HelixVPN refs in machinery |
| `DS-EXPORT-FORM-VALID` | Each emitted form parses in its target toolchain |
| `DS-SCHEMA` | `tokens/**` validate against JSON schema |

---

## 8. Cross-references

- Design-system master index → [`docs/design/README.md`](../../../../../design/README.md)
- Submodule overview → [`../../v10-design/00-overview-and-submodule.md`](../../v10-design/00-overview-and-submodule.md)
- OpenDesign foundation → [`../../v10-design/opendesign-foundation.md`](../../v10-design/opendesign-foundation.md)
- Design tokens → [`../../v10-design/design-tokens.md`](../../v10-design/design-tokens.md)
- Client core & UI → [05 — Client Core & UI](../05-client-core-ui/README.md)

---

*Sources: `docs/design/README.md`, `v10-design/*.md`.*

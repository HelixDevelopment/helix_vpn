# Helix VPN — Design System & UI/UX Documentation

**Revision:** 1
**Last modified:** 2026-07-04T12:00:00Z
**Status:** Complete

## Overview

Enterprise-grade, cross-platform design system for Helix VPN. 8 platforms, 3 UI frameworks, unified design language, light+dark themes, customizable palettes, WCAG 2.1 AA accessibility, and bleeding-edge quality.

### Quick Navigation

| Document | Description |
|----------|-------------|
| [OpenDesign DESIGN.md](./opendesign/helix/DESIGN.md) | 9-section design system spec (OpenDesign format) |
| [OpenDesign tokens.css](./opendesign/helix/tokens.css) | Compiled CSS custom properties (light+dark) |
| [OpenDesign manifest.json](./opendesign/helix/manifest.json) | Design system manifest |
| [OpenDesign components.html](./opendesign/helix/components.html) | Interactive component reference |
| [Component Library](./components/README.md) | Full component specs (all platforms) |
| [Screen Wireframes](./screens/README.md) | Every screen across all platforms |
| [Interaction Patterns](./interaction/README.md) | Animation specs, gestures, accessibility |

---

## Design System Summary

### 1. Brand Identity

- **Name:** Helix VPN
- **Primary color:** Teal (`#00897B`) — trust, security, technological sophistication
- **Logo:** Shield/globe motif
- **Tagline:** "Your privacy starts here"
- **Voice:** Confident, calm, competent — security is serious, not playful

### 2. Platform Coverage

| # | Platform | UI Framework | OS Versions | Bundle Target | Theme |
|---|----------|-------------|-------------|---------------|-------|
| 1 | macOS (Intel + Apple Silicon) | Tauri v2 + React | macOS 12+ | .dmg Universal | System (light/dark) |
| 2 | Windows (x64 + ARM64) | Tauri v2 + React | Win 10/11 | .msi / .exe | System (light/dark) |
| 3 | Linux (x64 + ARM64) | Tauri v2 + React | Ubuntu 22.04+ | .AppImage / .deb / .rpm | System (light/dark) |
| 4 | Android (ARM64 + x86_64) | Flutter + Dart | API 26+ (8.0+) | .apk / .aab | Material You + Dynamic |
| 5 | iOS (ARM64) | Flutter + Dart | iOS 15+ | .ipa | System (light/dark) |
| 6 | HarmonyOS (ARM64) | Flutter + ohos | API 12+ | .hap | System (light/dark) |
| 7 | Aurora OS (ARM64) | Qt6 / QML | Aurora 4.x+ | .rpm | Silica Ambiance (dark-first) |
| 8 | Web Extension | MV3 + React | Chrome 90+, FF 88+, Edge 90+, Safari 15+ | .zip | Browser |

### 3. Technology Stack

| Layer | Desktop | Mobile | Aurora | Web |
|-------|---------|--------|--------|-----|
| **UI Framework** | Tauri v2 | Flutter 3.29+ | Qt6 / QML | React + MV3 |
| **Rendering** | WKWebView/WebView2/WebKitGTK | Impeller | Silica (Qt) | Browser engine |
| **State Management** | Zustand / Valtio | flutter_bloc / Riverpod | C++ QObject | Zustand |
| **Rust Bridge** | Tauri Commands (IPC) | flutter_rust_bridge + UniFFI | Direct C FFI | wasm-bindgen |
| **Icons** | Lucide React | Material Symbols / SF Symbols | Silica icons | Lucide |
| **Design Tokens** | CSS custom properties | ThemeData / CupertinoTheme | QML properties | CSS vars |

### 4. Design Token Summary

| Domain | Count | Key Tokens |
|--------|-------|------------|
| Colors (primary) | 10 | Primary-50 through Primary-900 |
| Colors (semantic) | 12 | connected, connecting, disconnected, error, warning, info |
| Colors (surfaces) | 8 | bg-primary, bg-secondary, bg-tertiary, bg-elevated (× light/dark) |
| Typography | 18 | display, headline, title-lg/md/sm, body-lg/md/sm, caption, overline |
| Typography (mono) | 4 | mono-lg, mono-md, mono-sm, mono-caption |
| Spacing | 12 | 2px through 80px (4px base grid) |
| Border radius | 7 | none through full (9999px) |
| Elevation | 5 | 0–4 (flat → glow) |
| Motion | 4 | fast, base, slow, page (easing variants) |

---

## Theme System

### 5. Light + Dark Theme

Both themes are **authored** — every color token is explicitly set. Dark mode is NOT an automatic inversion.

| Surface | Light | Dark |
|---------|-------|------|
| App background | `#F0F4F8` | `#0A1929` |
| Cards/panels | `#FFFFFF` | `#132F4C` |
| Elevated surfaces (bg-tertiary) | `#E8EDF2` | `#1E4976` |
| Modals/dialogs | `#FFFFFF` | `#243B55` |
| Primary text | `#0A1929` | `#F0F4F8` |
| Secondary text | `#4A6375` | `#8BA3B8` |

### 6. Custom Palette Architecture

The token system supports any brand seed color via CSS custom property override:

```css
/* Built-in presets */
[data-theme="ocean-blue"] { --hx-primary-500: #2563EB; }
[data-theme="midnight"]   { --hx-primary-500: #7C3AED; }
[data-theme="forest"]     { --hx-primary-500: #059669; }
[data-theme="ruby"]       { --hx-primary-500: #E11D48; }
[data-theme="amethyst"]   { --hx-primary-500: #A855F7; }

/* User custom */
:root[data-theme="custom"] { --hx-primary-500: <user-color>; }
```

Theme/appearance setting: System (default) · Light · Dark · Custom (with palette picker)

---

## Implementation Checklist

### Phase 1: Foundation
- [x] Token architecture (colors, typography, spacing, elevation, motion, radii)
- [x] Light + dark theme definitions
- [x] Custom palette system (5 presets + user custom)
- [x] Font family selection per platform
- [x] Responsive breakpoints

### Phase 2: Core Components
- [x] Button variants (primary, secondary, ghost, danger, icon)
- [x] Connection toggle with full state machine
- [x] Server list item
- [x] Protocol badge
- [x] Latency indicator
- [x] Input fields
- [x] Modal/dialog
- [x] Toast notification
- [x] Card component
- [x] Toggle/switch (adaptive per platform)

### Phase 3: Specialized Components
- [x] Speed graph / sparkline
- [x] Data usage indicator
- [x] Bottom sheet (mobile)
- [x] Tab bar (adaptive)
- [x] Pull-to-refresh
- [x] Swipe actions (mobile)
- [x] System tray menu (desktop)
- [x] Quick Settings tile (Android)
- [x] Home screen widget (Android/iOS)
- [x] Aurora Silica components
- [x] Web extension popup components

### Phase 4: Screen Layouts
- [x] Desktop main window (connected/disconnected)
- [x] Desktop server selection
- [x] Desktop settings (tabbed)
- [x] Desktop connection details/stats
- [x] Mobile connection home screen
- [x] Mobile server selection
- [x] Mobile settings
- [x] Mobile split tunneling config
- [x] Mobile connection stats
- [x] Mobile support/help
- [x] Aurora main view
- [x] Aurora cover page
- [x] Aurora pulley menu
- [x] Web extension popup (connected/disconnected)
- [x] Web extension options page
- [x] Admin dashboard
- [x] Onboarding flow

### Phase 5: Interaction & Animation
- [x] Connection flow animation (3-phase, 2200ms)
- [x] Disconnect flow (800ms)
- [x] Page transitions (per platform)
- [x] Micro-interactions (16 defined)
- [x] Reduced motion support
- [x] Keyboard navigation (desktop)
- [x] Gesture navigation (mobile)
- [x] Haptic feedback patterns
- [x] Loading states (skeleton, spinner, progress)
- [x] Screen reader announcements

### Phase 6: OpenDesign Deliverables
- [x] DESIGN.md (9-section OpenDesign spec)
- [x] tokens.css (canonical CSS custom properties)
- [x] manifest.json (design system metadata)
- [x] components.html (interactive reference)
- [x] Component library documentation
- [x] Screen wireframes documentation
- [x] Interaction patterns specification
- [x] Full design documentation index (this document)

### Phase 7: Export & Validation
- [ ] Export DESIGN.md → PDF
- [ ] Export components.html → PNG screenshots
- [ ] Export all documentation to PDF/HTML
- [ ] Visual regression golden screenshots
- [ ] WCAG 2.1 AA contrast validation
- [ ] Cross-platform consistency review
- [ ] Figma design file generation

---

## Design Rules (Non-Negotiable)

1. **No overlapping elements.** Layout must never produce overlapping interactive elements or superimposed labels.
2. **No color-only indicators.** Every status must pair color + icon + text.
3. **No platform ignorance.** A Material-style switch on iOS is a defect — use CupertinoSwitch.
4. **No decorative-only effects.** Every gradient, shadow, and animation serves a functional purpose.
5. **No skeleton-only loading.** Show real content within 2s or error state.
6. **No dark-mode inversion.** Dark theme is explicitly authored (every `[data-theme="dark"]` token).
7. **No hardcoded thresholds.** Every latency/spacing/timing constant reads from tokens.
8. **No accessibility shortcuts.** WCAG 2.1 AA minimum enforced on every screen.

---

## File Structure

```
docs/design/
├── README.md                           ← This file (master index)
├── opendesign/
│   └── helix/
│       ├── manifest.json               ← OpenDesign design system manifest
│       ├── DESIGN.md                   ← 9-section OpenDesign spec
│       ├── tokens.css                  ← Compiled CSS custom properties
│       ├── components.html             ← Interactive component reference
│       ├── assets/                     ← Design assets (logos, icons, gradients)
│       ├── preview/                    ← Preview renders
│       └── source/                     ← Source files
├── tokens/
│   ├── color.json                      ← Color token definitions
│   ├── typography.json                 ← Type scale token definitions
│   ├── spacing.json                    ← Spacing scale definitions
├── components/
│   ├── README.md                       ← Component library master
│   ├── desktop/                        ← Desktop-specific components
│   ├── mobile/                         ← Mobile-specific components
│   ├── aurora/                         ← Aurora-specific components
│   └── web/                            ← Web-specific components
├── screens/
│   ├── README.md                       ← All screen wireframes
│   ├── desktop/                        ← Desktop screen specs
│   ├── mobile/                         ← Mobile screen specs
│   ├── aurora/                         ← Aurora screen specs
│   └── web/                            ← Web extension screen specs
├── interaction/
│   └── README.md                       ← Animations, gestures, accessibility
├── icons/
│   ├── app-icon/                       ← App icon variants (all sizes)
│   ├── tray-icon/                      ← System tray icon per platform
│   └── extension-icon/                 ← Browser extension icon
└── exports/
    ├── HelixVPN-Design-System.pdf       ← Full design system export
    ├── HelixVPN-Component-Library.pdf   ← Component library export
    ├── HelixVPN-Screen-Wireframes.pdf   ← Screen wireframes export
    └── HelixVPN-Interaction-Specs.pdf   ← Interaction patterns export
```

---

## OpenDesign Integration

OpenDesign is included as a git submodule under `submodules/open-design` and is consumed through a project wrapper script at `tools/opendesign`.

### Installation path

- Submodule: `submodules/open-design` (https://github.com/nexu-io/open-design, main branch)
- CLI wrapper: `tools/opendesign`
- Built artifacts live in `submodules/open-design/apps/daemon/dist/`

### Environment requirements

- Node.js 22+ (OpenDesign declares `~24`, but the CLI builds and runs on Node v22.19.0)
- pnpm 10.33.2 (installed globally via `npm install -g pnpm@10.33.2`)
- No Docker is used

### How to install / build

```bash
# 1. Initialize the submodule (already done)
git submodule update --init --recursive submodules/open-design

# 2. Install dependencies
cd submodules/open-design
pnpm config set engine-strict false
pnpm install

# The postinstall script builds the daemon CLI automatically.
```

### CLI command

Use the wrapper from the repository root. Do **not** use the system `/usr/bin/od` (GNU coreutils `od`).

```bash
./tools/opendesign --help
./tools/opendesign version
```

> **Note:** The OpenDesign CLI writes to a TTY but may appear empty when its stdout is piped or redirected. For scripted capture, wrap invocations in `script -q -c '<cmd>' /dev/null` or use the daemon HTTP API directly (see below).

### How to regenerate assets

1. Start the daemon (port 7456 by default):

   ```bash
   ./tools/opendesign --no-open --port 7456
   ```

2. Import / refresh the Helix VPN design system:

   ```bash
   ./tools/opendesign design-systems import-local \
     docs/design/opendesign/helix --name "Helix VPN" --json
   ```

3. Download the full export archive:

   ```bash
   curl -s -o docs/design/opendesign/helix/exports/helix-vpn-opendesign-archive.zip \
     http://127.0.0.1:7456/api/design-systems/user:helix-vpn/archive
   ```

   Extract it to regenerate `docs/design/opendesign/helix/exports/` contents.

### Current status

- ✅ OpenDesign submodule added and built
- ✅ Daemon starts on `http://127.0.0.1:7456`
- ✅ `design-systems import-local` successfully imports the Helix VPN design system
- ✅ Brand archive export generated (tokens, components, previews, source reports)
- ⚠️ Token contract validation reports **grade: needs-rebuild** (score 31/100, 23% source-backed A1 coverage)
- ⚠️ CLI stdout may not flush when piped; use TTY or HTTP API for automation
- ❌ No built-in Figma/Sketch/Adobe/XD/Penpot exporters found; only Figma **import** is available

### Known issues

1. **Manifest key mapping.** OpenDesign normalizes the manifest to `schemaVersion: od-design-system-project/v1` and expects keys such as `files.design`, `files.tokens`, and `files.components`. The existing `manifest.json` uses `schemaVersion: "1.0"` and keys `designMd`, `tokensCss`, `componentsHtml`. The importer handles this automatically, but the source manifest is not natively aligned with the OpenDesign schema.
2. **Token contract quality.** Many A1 structure tokens fall back to importer defaults because the source `tokens.css` uses project-specific prefixes (`--hx-*`) rather than OpenDesign's canonical token names.
3. **CLI output buffering.** Non-TTY consumers (pipes, redirect files) may receive empty output from `od design-systems list`, `od design-systems show`, etc. Work around with `script` or the REST API.

For full validation results, generated files, and command logs, see `docs/reviews/mvp-final/findings/phase2-opendesign-report.md`.

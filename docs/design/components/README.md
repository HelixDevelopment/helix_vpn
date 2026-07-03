# Helix VPN — Component Library

**Revision:** 1
**Last modified:** 2026-07-04T12:00:00Z
**Status:** Complete

## Overview

Enterprise-grade cross-platform UI component library for Helix VPN. All components ship with light + dark theme variants, responsive breakpoints, WCAG 2.1 AA accessibility, and platform-specific adaptations.

### Platform Matrix

| Component | Desktop (Tauri/React) | Mobile (Flutter) | Aurora (Qt/QML) | Web Extension |
|-----------|----------------------|-------------------|-----------------|---------------|
| Button (5 variants) | ✓ | ✓ | ✓ | ✓ |
| Connection Toggle | ✓ (140px) | ✓ (140dp) | ✓ (Cover) | ✓ (mini) |
| Server List Item | ✓ (64px) | ✓ (72dp) | ✓ (80px) | ✓ (compact) |
| Protocol Badge | ✓ | ✓ | ✓ | ✓ |
| Latency Indicator | ✓ | ✓ | ✓ | ✓ |
| Input Field | ✓ (48px) | ✓ (56/44dp) | ✓ (48px) | ✓ |
| Modal/Dialog | ✓ | ✓ | ✓ | — |
| Toast Notification | ✓ | ✓ | ✓ | — |
| Card | ✓ (12px r) | ✓ (16dp r) | ✓ | ✓ |
| Toggle/Switch | ✓ | ✓ (adaptive) | ✓ (Silica) | ✓ |
| Speed Graph | ✓ (80px) | ✓ (80dp) | — | — |
| Data Usage Indicator | ✓ | ✓ | — | — |
| Bottom Sheet | — | ✓ | — | — |
| Tab Bar | ✓ | ✓ (adaptive) | — | ✓ |
| Pull-to-Refresh | — | ✓ | ✓ (Pulley) | — |
| Swipe Action | — | ✓ | ✓ | — |
| System Tray | ✓ | — | Cover page | Toolbar badge |
| Quick Settings Tile | — | ✓ (Android) | — | — |
| Skeleton Loader | ✓ | ✓ | ✓ | ✓ |
| Progress Bar | ✓ | ✓ | ✓ | ✓ |

---

## Component Specifications

### 1. Button

**Full spec:** [buttons.md](buttons.md)

| Variant | Background | Text | Border | Hover | Pressed | Disabled |
|---------|-----------|------|--------|-------|---------|----------|
| **Primary** | `--hx-primary-500` | White | None | `--hx-primary-600` | `--hx-primary-700` | 38% opacity |
| **Secondary** | Transparent | `--hx-primary-500` | 1px | overlay-hover | overlay-pressed | 38% opacity |
| **Ghost** | Transparent | text-primary | None | overlay-hover | overlay-pressed | 38% opacity |
| **Danger** | `--hx-semantic-error` | White | None | #B71C1C | #9A0007 | 38% opacity |
| **Icon** | Transparent | text-primary | None | overlay-hover | overlay-pressed | 38% opacity |

**Sizes:**
- Standard: 40px height, 24px padding-x, 8px radius
- Compact: 32px height, 16px padding-x, 8px radius
- Pill: 48px height, 32px padding-x, 9999px radius

**States:** Default · Hover · Pressed · Focused (2px outline) · Disabled (38% opacity) · Loading (spinner overlay)

---

### 2. Connection Toggle (Hero Component)

**The signature element of Helix VPN — a large circular button driving the core action.**

- **Size:** 140×140px (desktop), 140×140dp (mobile)
- **Shape:** Full circle with 4px state-colored ring
- **Icon:** Power/shield glyph, 48px white

**State machine:**

| State | Fill | Ring | Glow | Animation |
|-------|------|------|------|-----------|
| **Disconnected** | `--hx-semantic-disconnected` (#F44336) | Red | None | Static |
| **Connecting** | `--hx-semantic-connecting` (#FF9800) | Amber | 20px blur | Spinner rotation 360°/1500ms |
| **Connected** | `--hx-primary-500` (#00897B) | Green (#4CAF50) | 20px blur, 30% opacity | Pulse scale 1.0→1.08, 1500ms |
| **Error** | `--hx-semantic-error` (#D32F2F) | Red | Flash | Fast pulse 800ms |

**Transition sequence (2200ms total):**
1. Tap → scale 0.95 (100ms ease-out)
2. Red→amber transition (200ms)
3. Spinner + "Connecting…" (500–2000ms, typical handshake)
4. Amber→green (300ms ease-in-out)
5. Checkmark spring-in (400ms bounce)
6. Glow pulse loop begins + stats panel slide-up (400ms)

---

### 3. Server List Item

| Dimension | Desktop | Mobile | Aurora | Web |
|-----------|---------|--------|--------|-----|
| Height | 64px | 72dp | 80px | 56px |
| Flag icon | 36×36px, 8px r | 40dp circle | 32px | 28px |
| Radius | 8px | — | — | 6px |

**Composition (L→R):** Flag icon → Server name + location subtitle → Latency dot + value → Protocol badge → Favorite star → Chevron

**States:** Default · Hover · Selected (primary tint 10%) · Pressed

---

### 4. Protocol Badge

| Property | Value |
|----------|-------|
| Height | 20px |
| Padding | 0 6px |
| Bg | Primary at 10% opacity |
| Text | Primary, 11px/500 |
| Radius | 4px |
| Variants | WG, SS, MQ, OV, IK |

---

### 5. Latency Indicator

| Range | Color | Name | Animation |
|-------|-------|------|-----------|
| < 50ms | #4CAF50 | Excellent | Static |
| 50–100ms | #FF9800 | Good | Subtle pulse |
| 100–200ms | #F57C00 | Fair | Pulse |
| > 200ms | #F44336 | Poor | Fast pulse |

Dot size: 8px, with 4px glow (same color, 40% opacity)

---

### 6. Input Fields

| Dimension | Desktop | Android | iOS | Aurora |
|-----------|---------|---------|-----|--------|
| Height | 48px | 56dp | 44dp | 48px |
| Radius | 8px | 8dp | 8pt | 0 (Silica rect) |
| Font | Body Medium | Body Medium | Body Medium | Body Medium |
| Focus | 2px outline | 2dp outline | 2pt border | 2px underline |

**States:** Default · Focused (primary) · Error (red) · Disabled (38% opacity)

---

### 7. Modal / Dialog

| Property | Value |
|----------|-------|
| Min width | 360px / max 560px |
| Radius | 16px (24px top for bottom sheets) |
| Shadow | Level 3 |
| Backdrop | `--hx-scrim` with 150ms fade-in |
| Header | Title Large (22px/600) + close button |
| Actions | Right-aligned, primary + secondary |

---

### 8. Toast Notification

| Property | Value |
|----------|-------|
| Max width | 400px |
| Height | 48px |
| Radius | 8px |
| Border-left | 4px semantic color |
| Shadow | Level 2 |
| Duration | 4000ms auto-dismiss |
| Animation | Slide in from top-right, 300ms ease-out |

---

### 9. Card

| Property | Value |
|----------|-------|
| Bg | Surface secondary |
| Radius | 12px (desktop) / 16dp (mobile) |
| Padding | 16px |
| Shadow | Level 1 (flat preferred) |

---

### 10. Toggle / Switch

Adaptive per platform:
- **Android:** Material 3 Switch, Primary active track (28dp × 48dp)
- **iOS:** CupertinoSwitch, Primary active color (32dp × 52dp)
- **Desktop:** Custom toggle with CSS transition (200ms)
- **Aurora:** Silica slider-style

---

### 11. Speed Graph / Sparkline

| Property | Value |
|----------|-------|
| Height | 80px |
| Download line | #00897B, 2px stroke, gradient fill |
| Upload line | #00BCD4, 2px dashed, gradient fill |
| Grid | Dotted at 25/50/75% |
| Update | 1000ms |

---

### 12. System Tray Icon

| Platform | Format | Badge |
|----------|--------|-------|
| macOS | Template PNG 22×22 | Native overlay |
| Windows | .ico 16×16 | Custom overlay (8px dot) |
| Linux | StatusNotifierItem 22×22 | Native |

Badge colors: green (connected), amber (connecting), red (disconnected/error)

---

## Platform-Specific Component Collections

### Desktop (Tauri/React)
- [Desktop Components](./desktop/desktop-components.md)
- System tray context menu
- Menu bar integration
- Settings tab navigation
- Drag-and-drop server reordering
- Window management (snap, resize, minimize-to-tray)

### Mobile (Flutter — Android/iOS/HarmonyOS)
- [Mobile Components](./mobile/mobile-components.md)
- Bottom sheet (50% → 90%, 24dp top radius)
- Tab bar (Android: Material / iOS: Cupertino)
- Pull-to-refresh (64dp threshold)
- Swipe actions (72dp width)
- Quick Settings tile (Android, 1×1)
- Home screen widget (Android 4×1, iOS small/medium)
- Persistent notification with in-line controls

### Aurora OS (Qt6/QML — Silica)
- [Aurora Components](./aurora/aurora-components.md)
- Pulley menu (pull-down, 56px items)
- Cover page (1:1, gradient + status dot)
- Context menu (long-press, 48px items)
- Silica text fields (underline focus)
- Dialogs (full-width stacked actions)
- Slider (4px track, 20px thumb)

### Web Extension
- [Web Components](./web/web-components.md)
- Popup UI (380×480px max)
- Toolbar badge (state dot)
- Options page (tabbed, 600px)
- Native messaging indicator
- Content script controls

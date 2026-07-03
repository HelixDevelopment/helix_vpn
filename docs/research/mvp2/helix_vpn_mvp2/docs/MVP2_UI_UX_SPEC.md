# MVP2 UI/UX Design System Specification

## Helix VPN Cross-Platform Design System

**Version:** 1.0.0  
**Date:** July 2025  
**Status:** Draft for Implementation  
**Scope:** Desktop (macOS, Windows, Linux), Mobile (Android, iOS, HarmonyOS), Web, Browser Extension  
**Frameworks:** Tauri v2 (Desktop), Flutter (Mobile), React (Web/Extension)  

---

## Table of Contents

1. [Design System Overview](#1-design-system-overview)
2. [Color System](#2-color-system)
3. [Typography](#3-typography)
4. [Layout Grid](#4-layout-grid)
5. [Component Library](#5-component-library)
6. [Screen Specifications](#6-screen-specifications)
7. [Interaction Patterns](#7-interaction-patterns)
8. [Iconography](#8-iconography)
9. [Animation & Motion](#9-animation--motion)
10. [Platform Adaptations](#10-platform-adaptations)

---

## 1. Design System Overview

### 1.1 Design Philosophy

The Helix VPN design system follows a **clean, minimal, security-focused** philosophy. The UI prioritizes clarity and trust, reflecting the app's core purpose: protecting user privacy. Every design decision serves the goal of making security feel effortless and approachable.

**Core Principles:**

| Principle | Description | Implementation |
|---|---|---|
| **Clarity** | Every element has a clear purpose | Minimal chrome, focused layouts, no decorative noise |
| **Trust** | Security should feel tangible | Color-coded states, clear visual feedback, transparent status |
| **Efficiency** | Primary actions are immediately accessible | One-tap connect, gesture shortcuts, contextual options |
| **Consistency** | Unified experience across all platforms | Shared design tokens, consistent interaction patterns |
| **Accessibility** | Usable by everyone | WCAG 2.1 AA compliance, screen reader support, keyboard navigation |

### 1.2 Design Tokens Approach

Design tokens are the single source of truth for all visual properties. They enable cross-platform consistency while allowing platform-specific adaptations.

```
tokens/
├── colors/                 # Color palette tokens
│   ├── primary.json       # Brand colors (teal/cyan)
│   ├── semantic.json      # Status/state colors
│   ├── surfaces.json      # Background/surface colors
│   └── text.json          # Text colors
├── typography/            # Type scale tokens
│   ├── scale.json         # Size ramp (Display -> Overline)
│   └── family.json        # Font family per platform
├── spacing/               # Spacing scale tokens
│   └── scale.json         # 4px base grid
├── elevation/             # Shadow/depth tokens
│   └── shadows.json
├── motion/                # Animation tokens
│   ├── duration.json
│   └── easing.json
└── radii/                 # Border radius tokens
    └── scale.json
```

**Token Format (JSON/CSS Variable):**
```css
/* Example: Primary token cascade */
--helix-primary-50:  #E0F2F1;   /* Teal tint */
--helix-primary-100: #B2DFDB;
--helix-primary-200: #80CBC4;
--helix-primary-300: #4DB6AC;
--helix-primary-400: #26A69A;
--helix-primary-500: #00897B;   /* Primary brand */
--helix-primary-600: #00796B;
--helix-primary-700: #00695C;
--helix-primary-800: #004D40;
--helix-primary-900: #00332C;
```

### 1.3 Cross-Platform Consistency Strategy

The design system employs a **"unified core with native skin"** approach:

- **Shared Design Tokens**: Colors, spacing, typography scales are consistent across all platforms
- **Platform-Native Components**: Widgets adapt to each platform's design language (Material 3, Cupertino, Fluent, Silica)
- **Unified Interaction Patterns**: Connection flows, state transitions, and gestures are consistent
- **Platform-Specific Chrome**: Navigation, menus, and system integration follow platform conventions

```
+-----------------------------------------------------+
|                    DESIGN SYSTEM                     |
+-----------------------------------------------------+
|  Core Layer (Shared)                                 |
|  - Color tokens, spacing, iconography               |
|  - Connection state semantics                       |
|  - Information architecture                         |
|  - Accessibility standards                          |
+----------+-------------+-------------+-------------+
|  Desktop |   Mobile    |    Web      |  Browser    |
|  (Tauri) |  (Flutter)  |   (React)   |  Extension  |
|          |             |             |             |
| - Web UI | - Material3 | - Tailwind  | - Popup UI  |
| - System | - Cupertino |   + shadcn  | - Options   |
|   tray   | - ArkUI     |             | - Content   |
| - Menu   | - Silica    |             |   script    |
+----------+-------------+-------------+-------------+
```

### 1.4 Accessibility First (WCAG 2.1 AA)

All components and screens must meet **WCAG 2.1 Level AA** compliance:

**Requirements:**
- Color contrast ratio >= 4.5:1 for body text
- Color contrast ratio >= 3:1 for large text (18pt+) and UI components
- All interactive elements have visible focus indicators (`focus-visible`)
- All form inputs have associated `<label>` elements
- Dynamic content changes announced via `aria-live` regions
- Full keyboard navigation support (Tab order, Enter/Space activation)
- Touch targets minimum 44x44dp/dpx (mobile)
- `prefers-reduced-motion` respected for all animations
- Screen reader tested with NVDA (Windows), JAWS (Windows), VoiceOver (macOS/iOS), TalkBack (Android)

---

## 2. Color System

### 2.1 Primary Palette (Teal/Cyan)

The primary palette uses teal/cyan tones to evoke trust, security, and technological sophistication.

| Token | Hex | HSL | Usage |
|---|---|---|---|
| `--helix-primary-50` | #E0F2F1 | 174° 52% 91% | Lightest tint, backgrounds |
| `--helix-primary-100` | #B2DFDB | 174° 47% 78% | Hover states, light fills |
| `--helix-primary-200` | #80CBC4 | 174° 42% 65% | Secondary highlights |
| `--helix-primary-300` | #4DB6AC | 174° 38% 51% | Accent elements |
| `--helix-primary-400` | #26A69A | 174° 62% 40% | Links, active states |
| `--helix-primary-500` | #00897B | 174° 100% 27% | **Primary brand color** |
| `--helix-primary-600` | #00796B | 174° 100% 24% | Primary hover |
| `--helix-primary-700` | #00695C | 174° 100% 21% | Primary pressed |
| `--helix-primary-800` | #004D40 | 174° 100% 15% | Strong emphasis |
| `--helix-primary-900` | #00332C | 174° 100% 10% | Deepest shade |

**Secondary Cyan Accent:**

| Token | Hex | Usage |
|---|---|---|
| `--helix-accent-400` | #00BCD4 | Secondary accent, data viz |
| `--helix-accent-500` | #00ACC1 | Accent hover |
| `--helix-accent-600` | #0097A7 | Accent pressed |

### 2.2 Semantic Colors

Semantic colors convey meaning universally across all platforms.

| State | Token | Hex | RGBA | Usage |
|---|---|---|---|---|
| **Connected** | `--helix-connected` | #4CAF50 | rgba(76,175,80,1) | VPN connected, success states |
| **Connecting** | `--helix-connecting` | #FF9800 | rgba(255,152,0,1) | In-progress, warning states |
| **Disconnected** | `--helix-disconnected` | #F44336 | rgba(244,67,54,1) | VPN disconnected, offline |
| **Error** | `--helix-error` | #D32F2F | rgba(211,47,47,1) | Critical errors, destructive |
| **Warning** | `--helix-warning` | #F57C00 | rgba(245,124,0,1) | Caution, attention needed |
| **Info** | `--helix-info` | #1976D2 | rgba(25,118,210,1) | Informational messages |

**Semantic Opacity Variants:**
```css
--helix-connected-bg: rgba(76, 175, 80, 0.12);
--helix-connecting-bg: rgba(255, 152, 0, 0.12);
--helix-disconnected-bg: rgba(244, 67, 54, 0.12);
--helix-error-bg: rgba(211, 47, 47, 0.12);
--helix-warning-bg: rgba(245, 124, 0, 0.12);
--helix-info-bg: rgba(25, 118, 210, 0.12);
```

### 2.3 Latency Color Coding

Latency indicators use a traffic-light system:

| Latency Range | Color | Hex | Semantic |
|---|---|---|---|
| < 50ms | Green | #4CAF50 | Excellent |
| 50-100ms | Amber | #FF9800 | Good |
| 100-200ms | Orange | #F57C00 | Fair |
| > 200ms | Red | #F44336 | Poor |

### 2.4 Dark Theme Colors

```css
.dark {
  /* Surfaces */
  --helix-bg-primary: #0A1929;       /* Deepest background */
  --helix-bg-secondary: #132F4C;     /* Cards, panels */
  --helix-bg-tertiary: #1E4976;      /* Elevated surfaces */
  --helix-bg-elevated: #243B55;      /* Modals, dialogs */

  /* Text */
  --helix-text-primary: #F0F4F8;     /* Primary text */
  --helix-text-secondary: #8BA3B8;   /* Secondary/muted */
  --helix-text-tertiary: #5A7A94;    /* Placeholder, disabled */
  --helix-text-disabled: #3D5A80;    /* Disabled text */

  /* Borders */
  --helix-border-default: #1E4976;
  --helix-border-subtle: #132F4C;
  --helix-border-focus: #00897B;
}
```

### 2.5 Light Theme Colors

```css
.light {
  /* Surfaces */
  --helix-bg-primary: #F0F4F8;       /* Background */
  --helix-bg-secondary: #FFFFFF;     /* Cards, panels */
  --helix-bg-tertiary: #E8EDF2;     /* Elevated surfaces */
  --helix-bg-elevated: #FFFFFF;      /* Modals, dialogs */

  /* Text */
  --helix-text-primary: #0A1929;     /* Primary text */
  --helix-text-secondary: #4A6375;   /* Secondary/muted */
  --helix-text-tertiary: #8BA3B8;    /* Placeholder */
  --helix-text-disabled: #B0C4D4;    /* Disabled text */

  /* Borders */
  --helix-border-default: #C8D8E4;
  --helix-border-subtle: #E2EBF2;
  --helix-border-focus: #00897B;
}
```

### 2.6 Surface & Background Colors Summary

| Surface | Dark Theme | Light Theme |
|---|---|---|
| App background | #0A1929 | #F0F4F8 |
| Card/panel | #132F4C | #FFFFFF |
| Elevated (modal) | #1E4976 | #E8EDF2 |
| Input field | #0D2137 | #FFFFFF |
| Selected item | rgba(0,137,123,0.15) | rgba(0,137,123,0.10) |
| Hover overlay | rgba(255,255,255,0.05) | rgba(0,0,0,0.04) |
| Pressed overlay | rgba(255,255,255,0.08) | rgba(0,0,0,0.08) |
| Divider | #1E4976 | #E2EBF2 |
| Scrim (modal backdrop) | rgba(0,0,0,0.7) | rgba(0,0,0,0.5) |

---

## 3. Typography

### 3.1 Font Family Selection

| Platform | Font Stack | Fallback |
|---|---|---|
| **Desktop (Tauri)** | `system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif` | sans-serif |
| **Android (Flutter)** | `Roboto` (Material 3 default) | Device sans-serif |
| **iOS (Flutter)** | `SF Pro Text/Display` (Cupertino) | Device sans-serif |
| **HarmonyOS** | `HarmonyOS Sans` | Device default |
| **Aurora OS** | `Sailfish Silica` | System font |
| **Web/Extension** | `Inter, system-ui, -apple-system, sans-serif` | sans-serif |

**Monospace (Technical Data):**
| Platform | Font | Usage |
|---|---|---|
| All | `JetBrains Mono, 'Fira Code', 'SF Mono', monospace` | IP addresses, connection stats, protocol info, keys |

### 3.2 Type Scale

The type scale follows a 1.25x ratio (major third) with a 14px base.

| Token | Size | Weight | Line Height | Letter Spacing | Usage |
|---|---|---|---|---|---|
| **Display** | 36px / 2.25rem | 300 (Light) | 1.2 | -0.5px | Connection status numbers |
| **Headline** | 28px / 1.75rem | 600 (SemiBold) | 1.3 | -0.25px | Screen titles (desktop) |
| **Title Large** | 22px / 1.375rem | 600 | 1.3 | 0 | Dialog titles, section headers |
| **Title Medium** | 18px / 1.125rem | 600 | 1.4 | 0.15px | Card titles, sub-screens |
| **Title Small** | 16px / 1rem | 500 (Medium) | 1.4 | 0.1px | List section headers |
| **Body Large** | 16px / 1rem | 400 (Regular) | 1.5 | 0.5px | Primary body text |
| **Body Medium** | 14px / 0.875rem | 400 | 1.5 | 0.25px | Default body, descriptions |
| **Body Small** | 12px / 0.75rem | 400 | 1.5 | 0.4px | Captions, metadata |
| **Caption** | 11px / 0.6875rem | 500 | 1.4 | 0.5px | Labels, timestamps |
| **Overline** | 10px / 0.625rem | 600 | 1.4 | 1.5px | Category labels, all-caps |

### 3.3 Font Weights

| Weight | Name | Usage |
|---|---|---|
| 300 | Light | Large display numbers, hero text |
| 400 | Regular | Body text, descriptions |
| 500 | Medium | Buttons, list items, emphasis |
| 600 | SemiBold | Headlines, titles, section headers |
| 700 | Bold | Critical labels, connection status |

### 3.4 Monospace Type Scale (Technical Data)

| Token | Size | Weight | Usage |
|---|---|---|---|
| Mono Large | 18px | 400 | Connection time, large stats |
| Mono Medium | 14px | 400 | IP addresses, server info |
| Mono Small | 12px | 500 | Protocol badges, key fragments |
| Mono Caption | 11px | 400 | Debug info, logs |

---

## 4. Layout Grid

### 4.1 Desktop Layout Grid

The desktop UI uses a responsive grid system with a minimum width of 360px and maximum of 1440px.

| Property | Value |
|---|---|
| Min width | 360px (small popups) |
| Max width | 1440px (full admin panel) |
| Main window | 420px (connection) / 520px (servers) / 640px (settings) |
| Gutter | 24px |
| Column count | 12 (fluid) |
| Margin | 24px (desktop), 16px (compact) |

**Desktop Window Sizes:**
| Screen | Width | Height | Purpose |
|---|---|---|---|
| Main (connected) | 420px | 600px | Primary interface |
| Main (disconnected) | 420px | 480px | Compact state |
| Server Selection | 520px | 700px | Full server list |
| Settings | 640px | 700px | Tabbed settings |
| Connection Details | 420px | 550px | Stats overlay |

### 4.2 Mobile Layout Grid

| Platform | System | Base Unit |
|---|---|---|
| Android | Material 3 | 8dp grid |
| iOS | Cupertino | 8pt grid |
| HarmonyOS | ArkUI | 4vp grid |

**Mobile Safe Areas:**
- Top: Respect `SafeArea` top inset (notch, status bar)
- Bottom: Respect `SafeArea` bottom inset (home indicator, nav bar)
- Horizontal: 16dp/dpx padding (mobile), 24dp (tablet)

### 4.3 Spacing Scale (4px Base)

All spacing uses a 4px base unit for consistency.

| Token | Value | Pixels | Usage |
|---|---|---|---|
| `--space-0-5` | 0.125rem | 2px | Icon gaps, hairline spacing |
| `--space-1` | 0.25rem | 4px | Tight component internal padding |
| `--space-2` | 0.5rem | 8px | Default component internal padding |
| `--space-3` | 0.75rem | 12px | Small component margin |
| `--space-4` | 1rem | 16px | Default padding, card gutters |
| `--space-5` | 1.25rem | 20px | Medium spacing |
| `--space-6` | 1.5rem | 24px | Section padding, dialog margins |
| `--space-8` | 2rem | 32px | Large section gaps |
| `--space-10` | 2.5rem | 40px | Section separators |
| `--space-12` | 3rem | 48px | Major section padding |
| `--space-16` | 4rem | 64px | Hero spacing, connection button margin |
| `--space-20` | 5rem | 80px | Large visual separation |

### 4.4 Responsive Breakpoints

| Token | Width | Target |
|---|---|---|
| `xs` | < 380px | Extension popup (minimum) |
| `sm` | 380-640px | Extension popup, mobile PWA |
| `md` | 640-768px | Tablet PWA, small panels |
| `lg` | 768-1024px | Tablet landscape, compact desktop |
| `xl` | 1024-1280px | Desktop standard |
| `2xl` | 1280px+ | Desktop wide, admin panel |

### 4.5 Border Radius Scale

| Token | Value | Usage |
|---|---|---|
| `--radius-none` | 0 | Sharp edges (tables, dividers) |
| `--radius-sm` | 4px | Small tags, badges |
| `--radius-md` | 8px | Buttons, inputs, small cards |
| `--radius-lg` | 12px | Cards, panels |
| `--radius-xl` | 16px | Large cards, bottom sheets |
| `--radius-2xl` | 24px | Modals, dialogs |
| `--radius-full` | 9999px | Pills, avatars, connection button |

---

## 5. Component Library

### 5.1 Desktop Components (Tauri/WebView)

#### 5.1.1 Button Variants

| Variant | Background | Text | Border | Hover | Pressed |
|---|---|---|---|---|---|
| **Primary** | #00897B | #FFFFFF | none | #00796B | #00695C |
| **Secondary** | transparent | #00897B | 1px #00897B | rgba(0,137,123,0.08) | rgba(0,137,123,0.16) |
| **Ghost** | transparent | #F0F4F8 | none | rgba(255,255,255,0.08) | rgba(255,255,255,0.12) |
| **Danger** | #D32F2F | #FFFFFF | none | #B71C1C | #9A0007 |
| **Disabled** | rgba(255,255,255,0.12) | rgba(255,255,255,0.38) | none | - | - |

**Button Specs:**
- Min height: 40px (standard), 32px (compact)
- Horizontal padding: 24px (standard), 16px (compact)
- Border radius: 8px (standard), 4px (compact)
- Font: Body Medium, 500 weight
- Icon + text gap: 8px

#### 5.1.2 Connection Toggle (Desktop)

```
+-------------------+
|  [Power Icon]     |
|                   |
|    CONNECTED      |
|  US East (23ms)   |
+-------------------+
```

- Size: 140px x 140px circular
- Connected: Fill #00897B with #4CAF50 glow pulse
- Connecting: Animated spinner on #FF9800 ring
- Disconnected: #F44336 with subtle shadow
- Border: 4px solid ring (color = state)
- Glow: 20px blur, state color at 30% opacity, pulsing
- Icon: 48px, white
- Label below: Body Large, 600 weight, state color

#### 5.1.3 Server List Item

```
+------------------------------------------+
| [FLAG]  US East - New York     [Fav] WG  |
|         23ms  * * * *  34% load          |
+------------------------------------------+
```

- Height: 64px
- Left: Flag icon (36x36px, 8px radius) + 12px gap
- Title: Body Large, 500 weight
- Subtitle: Body Small, secondary color
- Right: Favorite star + Protocol badge + Chevron
- Divider: 1px bottom border
- Selected state: Primary tint background (10% opacity)
- Hover: Hover overlay

#### 5.1.4 Protocol Badge

```
+------+
|  WG  |
+------+
```

- Size: auto-width x 20px
- Background: Primary at 10% opacity
- Text: Primary color, Caption weight 500
- Border radius: 4px
- Padding: 0 6px
- Variants: "WG" (WireGuard), "OV" (OpenVPN), "IK" (IKEv2)

#### 5.1.5 Latency Indicator

| Color | Size | Animation |
|---|---|---|
| Green (<50ms) | 8px circle | Static |
| Amber (50-100ms) | 8px circle | Subtle pulse |
| Orange (100-200ms) | 8px circle | Pulse |
| Red (>200ms) | 8px circle | Fast pulse |

- Displayed as colored dot + "23ms" label
- Dot has 4px colored glow (same color, 40% opacity)

#### 5.1.6 Settings Card/Section

- Background: bg-secondary
- Border radius: 12px
- Padding: 16px
- Section header: Title Small, primary color, uppercase
- Item height: 56px
- Divider between items: 1px border-subtle
- Left icon: 24px, primary or secondary color

#### 5.1.7 Input Fields

```
+-------------------------------+
| Label                         |
| +---------------------------+ |
| | Placeholder text       [] | |
| +---------------------------+ |
| Helper text                   |
+-------------------------------+
```

- Height: 48px
- Background: bg-primary (dark) / #FFFFFF (light)
- Border: 1px border-default
- Border radius: 8px
- Padding: 0 16px
- Font: Body Medium
- Focus: 2px border-focus outline
- Error: 2px #D32F2F outline + red helper text
- Disabled: 38% opacity

#### 5.1.8 Modal/Dialog

- Min width: 360px, Max: 560px
- Background: bg-elevated
- Border radius: 16px
- Shadow: 0 24px 48px rgba(0,0,0,0.4)
- Header: Title Large, 600 weight
- Content: Body Medium
- Actions: Right-aligned, Primary + Secondary buttons
- Backdrop: Scrim color with fade-in 150ms

#### 5.1.9 Toast Notification

```
+----------------------------------+
| [Icon]  Message              [X] |
+----------------------------------+
```

- Width: auto (max 400px)
- Height: 48px
- Background: bg-elevated
- Border left: 4px (semantic color)
- Border radius: 8px
- Shadow: 0 8px 24px rgba(0,0,0,0.3)
- Duration: 4000ms (auto-dismiss)
- Animation: Slide in from top-right, 300ms ease-out

#### 5.1.10 System Tray Menu (Desktop)

```
+----------------------------+
|  HelixVPN - Connected      |
|  US East (23ms)            |
|----------------------------|
|  [ICON] Connect            |
|  [ICON] Disconnect         |
|----------------------------|
|  [ICON] Select Server...   |
|  [ICON] Preferences...     |
|----------------------------|
|  [ICON] Quit               |
+----------------------------+
```

- Follows platform menu conventions
- macOS: NSStatusItem with template icon
- Windows: NotifyIcon with context menu
- Linux: AppIndicator/StatusNotifierItem
- Items: 28px height, 16px icon, 12px gap
- Separator: 1px line

---

### 5.2 Mobile Components (Flutter)

#### 5.2.1 Connection Button (Large Circular)

- Size: 140dp diameter
- Shape: Circle with radial gradient glow
- Connected: #00897B fill, #4CAF50 pulsing outer ring (20dp spread)
- Connecting: #FF9800 fill, rotating spinner overlay
- Disconnected: #F44336 fill, static
- Icon: 48dp, white, centered
- Shadow: 0 20px 40px state-color at 30%
- Animation: Pulse scale 1.0 -> 1.08, 1500ms ease-in-out (connected only)

#### 5.2.2 Server Selection Tile

- Height: 72dp (Material 3 ListTile)
- Leading: Country flag circle (40dp)
- Title: Body Large, 500 weight
- Subtitle: Body Small (latency dot + "23ms" + load %)
- Trailing: Protocol chips + favorite icon
- Selected: Primary tinted background
- Ripple: Platform default

#### 5.2.3 Bottom Sheet

- Initial height: 50% screen
- Max height: 90% screen
- Background: Surface color
- Top handle: 4dp x 32dp, #8BA3B8, 2dp radius
- Border radius: 24dp top corners
- Entry animation: Slide up 300ms, ease-out
- Drag to dismiss: Velocity threshold 300dp/s

#### 5.2.4 Tab Bar

- Height: 48dp (Android) / Cupertino navigation bar (iOS)
- Background: Surface color
- Indicator: 2dp height, Primary color
- Active text: Primary, 500 weight
- Inactive text: Secondary color, 400 weight
- iOS: Segmented control or bottom Cupertino tab bar

#### 5.2.5 Switch/Toggle (Platform-Adaptive)

- Android: Material 3 Switch with Primary track
- iOS: CupertinoSwitch with Primary active color
- Track: 32dp x 52dp (iOS) / 28dp x 48dp (Android)
- Thumb: 24dp (iOS) / 20dp (Android)
- Active: Primary color track
- Inactive: Surface variant track

#### 5.2.6 Text Field (Adaptive)

- Android: Material 3 OutlinedTextField with Primary focus
- iOS: CupertinoTextField with subtle border
- Height: 56dp (Android) / 44dp (iOS)
- Border radius: 8dp
- Focus: 2dp Primary outline
- Error: 2px #D32F2F outline

#### 5.2.7 Card

- Background: Surface color
- Border radius: 16dp
- Elevation: 0 (flat), shadow for elevated
- Padding: 16dp
- Content: Title + body + optional actions

#### 5.2.8 Speed Graph/Sparkline

- Height: 80dp
- Width: Parent full width
- Download line: #00897B, 2dp stroke
- Upload line: #00BCD4, 2dp stroke, dashed
- Fill: Gradient fade to transparent (line color 20%)
- Grid: Dotted horizontal lines at 25/50/75%
- Labels: Mono Caption, secondary color
- Update interval: 1000ms

#### 5.2.9 Data Usage Indicator

```
+----------------------------------+
| Download      |  Upload          |
| [==== 1.2 GB] | [==== 340 MB]   |
| 45% of limit  |                  |
+----------------------------------+
```

- Two-column layout
- Value: Mono Medium, 500 weight
- Bar: 4dp height, full border radius
- Fill: Primary (download), Accent (upload)
- Background: Surface variant

#### 5.2.10 Quick Settings Tile (Android)

- Size: 1x1 tile (standard QS tile)
- Active: Primary color background, white icon
- Inactive: Surface color, secondary icon
- Label: "Helix VPN" / "Helix VPN: ON"
- Toggle animation: Color crossfade 200ms
- Long-press: Opens app

---

### 5.3 Aurora OS Components (Silica)

#### 5.3.1 Pulley Menu

- Trigger: Pull down from top edge
- Background: Primary color, dark theme
- Items: 56px height, white text
- Icon: 24px left-aligned
- Selection highlight: rgba(255,255,255,0.15)
- Animation: Slide down from top, items stagger 50ms

#### 5.3.2 Context Menu

- Trigger: Long-press on list items
- Background: Elevated surface
- Items: 48px height, Body Medium
- Divider: 1px subtle
- Animation: Fade in 150ms + scale from anchor point

#### 5.3.3 List Item (Silica Style)

- Height: 80px (standard), 64px (compact)
- Background: Transparent
- Primary label: Body Large, text-primary
- Secondary label: Body Small, text-secondary
- Icon: 32px, left-aligned, 16px padding
- Divider: 1px bottom border-subtle
- Pressed: rgba(255,255,255,0.08) overlay

#### 5.3.4 Text Field (Silica Style)

- Background: Surface color
- Border: 1px border-default, no radius (Silica rectangular)
- Height: 48px
- Font: Body Medium
- Placeholder: text-tertiary
- Focus: 2px Primary underline
- Error: 2px #D32F2F underline

#### 5.3.5 Slider

- Track height: 4px
- Active track: Primary color
- Inactive track: Surface variant
- Thumb: 20px circle, Primary fill, 2px white border
- Value label: Floating tooltip above thumb

#### 5.3.6 Dialog (Silica)

- Width: 90% screen, max 480px
- Background: Elevated surface
- Border radius: 8px (Silica style)
- Header: Title Medium, 600 weight
- Content: Body Medium
- Actions: Full-width buttons stacked
- Entry: Fade 200ms + slight scale

#### 5.3.7 Cover Page

- Size: 1:1 square (app cover)
- Background: Gradient from Primary-800 to Primary-900
- Connection status: Centered large dot (12px) + state text
- Server: Caption text below status
- Animation: Dot pulse glow matching state color
- Ambiance: Respects system ambiance settings

---

## 6. Screen Specifications

### 6.1 Desktop Screens

#### 6.1.1 Main Window (Connected State)

```
+----------------------------------+
|  [TRAY]  Helix VPN         [_][X] |
+----------------------------------+
|                                  |
|  [Server Selector Bar]           |
|  [US Flag] US East - New York >  |
|                                  |
|         +----------+             |
|         | [POWER]  |  <- Large   |
|         | CONNECTED|    toggle   |
|         +----------+             |
|          (glowing ring)          |
|                                  |
|  Connected                       |
|  23ms latency                    |
|                                  |
|  IP: 203.0.113.45                |
|  Protocol: WireGuard             |
|  Duration: 02:34:18              |
|                                  |
|  [Speed graph sparkline]         |
|                                  |
|  Download: 1.2 GB  Upload: 340MB |
|                                  |
+----------------------------------+
```

**Specs:**
- Window: 420px x 600px
- Background: bg-primary
- Server selector: 64px height, bg-secondary, 12px radius
- Connection button: 140px circle, centered
- Status text: Title Medium, #4CAF50
- Info panel: bg-secondary card, 12px radius, Mono Small text
- Speed graph: 80px height, full width minus 32px padding
- Data usage: Two-column flex layout

#### 6.1.2 Main Window (Disconnected State)

- Window: 420px x 480px (compact)
- Connection button: 140px, #F44336 fill
- Status: "Disconnected", Title Medium, #F44336
- Info panel hidden
- Prompt: "Tap to connect securely", Body Small, secondary

#### 6.1.3 Server Selection

- Window: 520px x 700px
- Search bar: Sticky top, 48px height
- Quick connect: "Optimal Location" tile, 56px
- Section headers: "Favorites", "All Locations" - Overline style
- Server list: Scrollable, items 64px height
- Country groups: Collapsible accordion
- Latency sort: Default (ascending)
- Selected: Primary tint background + checkmark

#### 6.1.4 Settings (Tabbed)

- Window: 640px x 700px
- Tabs: General | Connection | Account | Advanced
- Tab height: 40px
- Content: Settings cards in vertical scroll
- Each card: bg-secondary, 12px radius, 16px padding
- Card title: Title Small, primary color
- Card items: 56px height rows

#### 6.1.5 Connection Details / Stats

- Overlay modal or separate tab
- Real-time: Download/upload speed (Display size)
- Connection time: Mono Large, counting up
- Protocol info: Badge + details
- Server info: Flag + name + IP
- Encryption: Cipher info in monospace
- Graphs: 5-minute rolling sparkline

---

### 6.2 Mobile Screens

#### 6.2.1 Connection (Home Screen)

```
+---------------------------+
| [Menu] Helix VPN  [Cog]   |
+---------------------------+
|                           |
|  [Server Bar]             |
|  [US] US East        23ms |
|                           |
|         +-----+           |
|         |  O  |           |
|         +-----+           |
|      CONNECTED            |
|                           |
|  [Connection Stats Card]  |
|  IP: 203.0.113.45         |
|  Time: 02:34:18           |
|  Protocol: WireGuard      |
|                           |
|  [Speed Sparkline]        |
|                           |
|  DL: 45 Mbps  UL: 12 Mbps |
|                           |
+---------------------------+
```

**Specs:**
- Full screen, SafeArea respected
- App bar: 56dp, transparent or surface
- Server bar: 72dp, tappable, navigates to server list
- Connection button: 140dp circle, centered vertically
- Status text: Title Medium below button
- Stats card: bg-secondary, 16dp radius, 16dp margin
- Bottom section: Speed + data, Body Small

#### 6.2.2 Server Selection

- Full screen with search AppBar
- Search field: 56dp, bg-secondary
- Quick connect: "Optimal Location" tile, 72dp
- Favorites section: Horizontal scroll chips
- Country groups: Expandable
- Items: 72dp ServerListTile
- Sort options: Latency (default), Load, Name, Distance
- Filter: Protocol, Streaming, P2P

#### 6.2.3 Settings

- Sections: Connection, Auto-Connect, Security, App, Account, About
- Section header: Overline, Primary color
- Items: Material ListTile, 56dp height
- Toggle items: Switch trailing
- Navigation items: Chevron trailing
- Sub-screens: Push navigation (slide in from right)

#### 6.2.4 Split Tunneling Configuration

- Tab bar: Off | Include | Exclude
- Mode explanation: Info banner at top
- App list: Checkbox tiles with app icons
- Search: Filter installed apps
- Save button: Bottom sticky, full width
- iOS limitation: Show warning about app enumeration restrictions

#### 6.2.5 Connection Stats

- Real-time speed: Display size, Mono
- Session duration: Mono Large, counting
- Total data: Download/Upload counters
- Graph: 5-minute sparkline, full width
- Protocol details: Expandable card
- Server info: Location, IP, load

#### 6.2.6 Support / Help

- Search FAQ at top
- Categories: Cards in grid (2 columns)
- Contact options: Chat, Email
- Diagnostics: "Send logs" button
- Version info: Caption at bottom

---

## 7. Interaction Patterns

### 7.1 Connection Flow Animation

```
DISCONNECTED -> CONNECTING -> CONNECTED
     |              |              |
   Red dot     Spinning ring   Green glow
   Static      Amber color     Pulse animation
                              Checkmark appears
```

**Phase 1: Initiate (0-500ms)**
- Button press: Scale to 0.95, 100ms
- Ring color transition: Red -> Amber, 200ms
- Spinner appears: Fade in + start rotation

**Phase 2: Handshake (500ms-2000ms)**
- Spinner: Continuous rotation (360deg/1500ms)
- Status text: "Connecting..." fade in
- Optional: Progress ring around button

**Phase 3: Success (2000ms+)**
- Ring color: Amber -> Green, 300ms
- Spinner fades out, checkmark scales in (0->1, 400ms bounce)
- Glow effect: 20px blur, Green at 30%, pulse loop begins
- Status: "Connected" text + server info slide up
- Stats panel: Slide in from bottom, 400ms

**Disconnect Flow:**
- Reverse of connect
- Green -> Red transition, 300ms
- Stats panel slides out
- Glow fades

### 7.2 State Transitions

| Transition | Duration | Easing | Visual |
|---|---|---|---|
| Button press | 100ms | ease-out | Scale 0.95 |
| Color change | 200ms | ease-in-out | Crossfade |
| Panel slide | 400ms | cubic-bezier(0.4, 0, 0.2, 1) | translateY |
| Fade in | 300ms | ease-out | opacity 0->1 |
| Scale in | 400ms | spring(1, 80, 10) | scale 0->1 |
| Page push (mobile) | 300ms | ease-in-out | translateX |
| Bottom sheet | 300ms | ease-out | translateY |

### 7.3 Pull-to-Refresh

- Trigger: Pull down past 64dp on scrollable lists
- Indicator: Circular progress, Primary color
- Release threshold: 80dp
- Animation: Rotation + arc sweep
- Haptic: Light impact at threshold (iOS)
- Completion: Snap back 200ms

### 7.4 Swipe Actions (Mobile)

| Direction | Action | Target |
|---|---|---|
| Swipe left (server) | Delete favorite | Server list item |
| Swipe right (server) | Quick connect | Server list item |
| Swipe left (setting) | Reset to default | Settings item |

- Action width: 72dp
- Background: Semantic color (green for connect, red for delete)
- Icon: 24dp, white
- Full swipe: Auto-execute action

### 7.5 Long-Press Menus

- Trigger: 500ms long press
- Menu: Context menu / Bottom sheet (adaptive)
- Items: Copy IP, View details, Quick actions
- Haptic: Medium impact on trigger
- Dismiss: Tap outside or back gesture

### 7.6 Keyboard Navigation (Desktop)

| Key | Action |
|---|---|
| Tab | Move focus forward |
| Shift+Tab | Move focus backward |
| Enter/Space | Activate focused element |
| Escape | Close modal/dialog/menu |
| Ctrl+K | Quick connect/disconnect |
| Ctrl+Shift+S | Open server selection |
| Ctrl+, | Open preferences |
| Ctrl+Q | Quit application |

- Focus ring: 2px Primary outline, 2px offset
- Focus visible: Only on keyboard navigation (not mouse click)

### 7.7 Screen Reader Support

- All interactive elements have descriptive `aria-label`
- Connection state changes announced via `aria-live="polite"`
- Server selection announces: "Connected to US East, 23 milliseconds latency"
- Status updates: "Connection lost, reconnecting"
- Landmark regions: `main`, `navigation`, `complementary`
- Skip links for main content

---

## 8. Iconography

### 8.1 Icon Set Requirements

**Unified icon set across all platforms:**
- Style: Outlined (line weight 1.5px-2px), rounded caps
- Size variants: 16px, 20px, 24px, 32px, 48px
- Color: Inherits text color by default
- Platform icon libraries:
  - Desktop/Web: Lucide React (or Phosphor Icons)
  - Mobile (Android): Material Symbols Outlined
  - Mobile (iOS): SF Symbols
  - Aurora: Sailfish Silica icons

### 8.2 Core Icon Inventory

| Icon | Name | Usage | Platforms |
|---|---|---|---|
| Power | `power` / `power_settings_new` | Connection toggle | All |
| Shield | `shield` / `shield.checkered` | Security status | All |
| Shield Check | `shield-check` | Protected state | All |
| Globe | `globe` | Server/location | All |
| Server | `server` | Server list | All |
| Settings | `settings` / `gearshape` | Preferences | All |
| Chevron Right | `chevron-right` | Navigation | All |
| Chevron Down | `chevron-down` | Expand | All |
| Search | `search` / `magnifyingglass` | Search | All |
| Close | `x` / `xmark` | Close/dismiss | All |
| Check | `check` / `checkmark` | Selected | All |
| Star | `star` / `star.fill` | Favorite | All |
| Clock | `clock` | Duration/time | All |
| Speed | `gauge` / `speedometer` | Speed test | All |
| Download | `download` / `arrow.down` | Download data | All |
| Upload | `upload` / `arrow.up` | Upload data | All |
| Lock | `lock` | Secure/encrypted | All |
| Lock Open | `lock-open` | Unsecured | All |
| Wifi | `wifi` | Network | All |
| Wifi Off | `wifi-off` | No network | All |
| Alert | `alert-triangle` | Warning | All |
| Info | `info` | Information | All |
| Trash | `trash` | Delete | All |
| Refresh | `refresh-cw` | Refresh/retry | All |
| Menu | `menu` | Hamburger menu | All |
| Help | `help-circle` | Support | All |
| Log Out | `log-out` | Sign out | All |

### 8.3 Platform-Specific Icons

#### System Tray Icon (Desktop)

| State | macOS | Windows | Linux |
|---|---|---|---|
| Disconnected | Template PNG (B&W) | .ico 16x16 | PNG 22x22 |
| Connected | Template + green dot | .ico + overlay | PNG + badge |
| Connecting | Template + amber dot | .ico + overlay | PNG + badge |
| Error | Template + red dot | .ico + overlay | PNG + badge |

- macOS: Use template image for dark mode compatibility
- Badge dot: 8px diameter, bottom-right offset
- Tooltip: "HelixVPN - Connected (US East)"

#### App Icon (All Platforms)

- Primary: Teal (#00897B) shield/globe motif
- Format: SVG source, exported to all platform sizes
- Sizes: 16, 32, 48, 72, 96, 128, 144, 192, 256, 512, 1024px
- Shape: Adaptive (Android), Rounded corners (iOS), Square (desktop)
- Background: Transparent or Primary-900

#### Notification Icon

- Android: Monochrome vector drawable, white on notification background
- iOS: App icon with tinted overlay
- Desktop: Platform-native notification with app icon

#### Quick Settings Icon (Android)

- Size: 24dp (standard QS icon)
- Style: Monochrome, white fill
- States: Disconnected (outline), Connected (filled)
- Label: "Helix VPN"

#### Extension Icon (Browser)

- Toolbar: 16px, 32px (PNG)
- Extension page: 48px, 128px
- Manifest: 16, 32, 48, 128px
- Color: Follows connection state badge overlay

---

## 9. Animation & Motion

### 9.1 Connection Animation

**Button Pulse (Connected State):**
```
Animation: scale 1.0 -> 1.08 -> 1.0
Duration: 1500ms
Easing: ease-in-out
Iteration: Infinite
Target: Outer glow ring + shadow
```

**Ripple Effect (On Connect):**
```
Origin: Button center
Animation: Scale from 0 to 4x, opacity 0.5 -> 0
Duration: 600ms
Easing: ease-out
Color: Primary at 20%
```

**Status Glow:**
```
Connected: Green glow, 20px blur, pulse opacity 0.2 -> 0.4
Connecting: Amber glow, rotating sweep gradient
Disconnected: No glow, static red
Error: Red pulse, 800ms (faster than connected)
```

### 9.2 Page Transitions

**Desktop:**
- Modal: Fade in backdrop (200ms) + scale content (300ms, spring)
- Page switch: Crossfade 200ms
- Drawer: Slide from right, 300ms ease-out

**Mobile (Android):**
- Push: Slide in from right (enter) + fade out (exit), 300ms
- Pop: Slide out to right (exit) + fade in (enter), 300ms
- Bottom sheet: Slide up 300ms, drag dismiss with velocity

**Mobile (iOS):**
- Push: Slide in from right, 350ms, Cupertino transition
- Pop: Slide out to right, 350ms
- Modal: Cover vertical, 400ms

### 9.3 Loading States

**Skeleton Loading:**
- Background: Surface color
- Shimmer: Linear gradient sweep, 1200ms
- Shape: Rounded rectangles matching content
- Trigger: Data fetch > 300ms

**Spinner:**
- Size: 24px (small), 40px (medium), 56px (large)
- Stroke: 3px, Primary color
- Duration: 1000ms per rotation
- Track: Surface variant, 20% opacity

**Progress Bar:**
- Height: 4px (linear), 48px (circular)
- Fill: Primary gradient
- Track: Surface variant
- Animation: Smooth width transition, 300ms

### 9.4 Micro-Interactions

| Interaction | Trigger | Animation | Duration |
|---|---|---|---|
| Button hover | Mouse over | Scale 1.02, subtle lift | 150ms |
| Button press | Mouse down | Scale 0.98 | 100ms |
| Toggle switch | Tap | Thumb translateX + color | 200ms |
| Checkbox | Tap | Checkmark stroke draw | 200ms |
| Radio | Tap | Dot scale in (spring) | 200ms |
| Card hover | Mouse over | translateY -2px, shadow increase | 200ms |
| List item tap | Touch | Ripple from touch point | 400ms |
| Refresh | Pull | Rotation + arc sweep | 1000ms |
| Toast | Trigger | Slide in from top-right | 300ms |
| Copy feedback | Tap copy | Checkmark flash | 500ms |

### 9.5 Reduced Motion Support

All animations respect `prefers-reduced-motion: reduce`:

```css
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}
```

**Reduced Motion Fallbacks:**
- Pulse -> Static glow (no animation)
- Page transitions -> Instant
- Spinners -> Static indicator + text "Loading..."
- Ripples -> Instant opacity change
- Graphs -> Static (no draw animation)

---

## 10. Platform Adaptations

### 10.1 macOS

**Aesthetic Principles:**
- Safari-style minimal chrome
- Translucent sidebar support (vibrancy)
- Rounded window corners (system default)
- System font: SF Pro

**Native Integration:**
- MenuBarExtra: Icon + status tooltip, click to show window
- Native menu bar: HelixVPN > About, Preferences, Services, Quit
- Window: Title bar with traffic lights, no custom chrome
- Keyboard shortcuts: Cmd+K (connect), Cmd+, (prefs), Cmd+Q (quit)
- Notifications: NSUserNotification with action buttons
- Auto-start: Login item via SMLoginItemSetEnabled

**Design Adjustments:**
- Increased border radius (16px cards vs 12px default)
- Subtle shadows (macOS design language)
- Sidebar navigation for settings (3-column layout optional)
- Translucent toolbar: `window.vibrancy = 'under-window'`

### 10.2 Windows

**Aesthetic Principles:**
- Fluent Design influence (Mica/Acrylic materials where supported)
- Segoe UI font family
- Corner radius: 8px (Windows 11) / 0px (Windows 10)
- System context menu styling

**Native Integration:**
- System tray: NotifyIcon with context menu
- Jump list: Connect, Disconnect, Recent servers
- Notifications: Windows Toast with inline actions
- Installer: MSI with WiX, UAC elevation
- Auto-start: Registry Run key

**Design Adjustments:**
- Snap layouts: Window supports Windows 11 snap zones
- Title bar: Custom with Mica material on Win11
- Settings: NavigationView-style left sidebar
- Density: Slightly more compact (8px spacing vs 12px)

### 10.3 Linux

**Aesthetic Principles:**
- Adaptable to GTK/Qt themes
- Respects system accent color where possible
- Follows freedesktop.org HIG
- Font: System default (usually Cantarell, Noto Sans)

**Native Integration:**
- System tray: AppIndicator3 / StatusNotifierItem
- Desktop file: Categories=Network;Security;VPN;
- Notifications: dbus notification server
- Auto-start: .desktop file in ~/.config/autostart
- Theming: Detect GTK theme, adapt colors

**Design Adjustments:**
- Respect `GTK_THEME` for color adaptation
- Flat design (no shadows) for GNOME compatibility
- Header bar: Integrate with CSD if available
- Density: Configurable (compact/comfortable)

### 10.4 Android (Material You)

**Aesthetic Principles:**
- Material 3 design system
- Dynamic color: Primary derived from wallpaper (Android 12+)
- Rounded shapes: 12dp cards, 24dp dialogs
- Elevation: Surface tint instead of shadows

**Native Integration:**
- Quick Settings tile: 1x1 toggle in notification shade
- Foreground service: Persistent notification with controls
- App shortcuts: Long-press launcher icon for connect/disconnect
- Widget: 4x1 home screen widget with status + toggle
- Biometric: Fingerprint/face unlock for app access
- Always-on VPN: `SUPPORTS_ALWAYS_ON` metadata

**Design Adjustments:**
- Dynamic color: Use `dynamicColorScheme()` when available
- Fallback: Static teal palette on Android <12
- Navigation: Bottom nav bar (3 items: Connect, Servers, Settings)
- Status bar: Transparent with dark/light icon adaptation
- Edge-to-edge: Extend content behind system bars

### 10.5 iOS (Cupertino Design)

**Aesthetic Principles:**
- Cupertino design language
- SF Symbols iconography
- Blur effects: System materials (systemThinMaterial, etc.)
- Rounded corners: System default radii
- Font: SF Pro Text/Display

**Native Integration:**
- Control Center: VPN toggle in Settings > VPN
- Widget: Small + medium home screen widgets
- Shortcuts: Siri Shortcuts for "Connect VPN"
- Biometric: Face ID / Touch ID gate
- Push: APNs for service alerts
- URL scheme: `helixvpn://connect`, `helixvpn://disconnect`

**Design Adjustments:**
- Navigation: CupertinoNavigationBar with blur
- Lists: CupertinoFormSection.insetGrouped for settings
- Buttons: CupertinoButton with system styling
- Switches: CupertinoSwitch (slimmer than Material)
- Pickers: CupertinoPicker / modal bottom picker
- Modals: CupertinoModalPopup with sheet presentation

### 10.6 HarmonyOS (ArkUI Design Language)

**Aesthetic Principles:**
- ArkUI design language
- Smooth rounded corners: 16vp cards, 24vp dialogs
- Subtle gradients and depth
- Font: HarmonyOS Sans

**Native Integration:**
- Service widget: Form card for home screen quick connect
- VpnExtensionAbility: System VPN integration
- Notification: HMS Push Kit for alerts
- Distributed capability: Cross-device sync (future)
- Biometric: System fingerprint/face

**Design Adjustments:**
- Colors: Teal primary with HarmonyOS accent adaptation
- Cards: Elevated with subtle shadow
- Typography: HarmonyOS Sans throughout
- Animations: Smooth spring physics (ArkUI default)
- Dialogs: Centered with blur backdrop

### 10.7 Aurora OS (Silica)

**Aesthetic Principles:**
- Silica UI design language
- Dark-first design (ambiance integration)
- Edge gestures: Swipe from edges for navigation
- Pulley menus: Pull down for context actions
- Transparency and blur effects
- Font: Sailfish Silica

**Native Integration:**
- Cover page: 1x1 app cover with connection status
- Pulley menu: Pull down for connect/disconnect/server
- Ambiance: Adapt UI tint to system ambiance
- Sailfish notifications: Platform-native alerts
- D-Bus: System integration for network state

**Design Adjustments:**
- Transparency: Silica glass effect for panels
- Navigation: No bottom nav, use page stack + gestures
- Input: Silica-styled text fields with underline focus
- Lists: Sailfish list items with press overlay
- Primary action: Often in pulley menu, not always visible
- Page transitions: Silica horizontal slide with depth

---

## Appendix A: Token Reference Quick Sheet

### Colors
```css
/* Primary */
#00897B (500), #00796B (600), #00695C (700)
/* Accent */
#00BCD4 (400), #00ACC1 (500)
/* Semantic */
#4CAF50 (connected), #FF9800 (connecting), #F44336 (disconnected)
#D32F2F (error), #F57C00 (warning), #1976D2 (info)
/* Dark theme bg */
#0A1929 (primary), #132F4C (secondary), #1E4976 (tertiary)
/* Light theme bg */
#F0F4F8 (primary), #FFFFFF (secondary), #E8EDF2 (tertiary)
```

### Spacing
```
4px, 8px, 12px, 16px, 20px, 24px, 32px, 40px, 48px, 64px
```

### Border Radius
```
4px (sm), 8px (md), 12px (lg), 16px (xl), 24px (2xl), 9999px (full)
```

### Animation
```
Fast: 150ms, Base: 200ms, Slow: 300ms, Page: 400ms
Easing: cubic-bezier(0.4, 0, 0.2, 1)
Spring: type-spring, damping: 20, stiffness: 300
```

---

## Appendix B: Implementation Checklist

### Phase 1: Foundation
- [ ] Define all design tokens in JSON
- [ ] Implement color system (dark + light themes)
- [ ] Set up typography scale per platform
- [ ] Configure spacing scale
- [ ] Implement border radius tokens

### Phase 2: Core Components
- [ ] Button variants (primary, secondary, ghost, danger)
- [ ] Connection toggle with animation
- [ ] Server list item
- [ ] Protocol badge
- [ ] Latency indicator
- [ ] Input fields
- [ ] Modal/dialog
- [ ] Toast notification

### Phase 3: Screen Layouts
- [ ] Desktop main window (connected/disconnected)
- [ ] Desktop server selection
- [ ] Desktop settings (tabbed)
- [ ] Mobile connection screen
- [ ] Mobile server selection
- [ ] Mobile settings
- [ ] Mobile split tunneling

### Phase 4: Interactions
- [ ] Connection flow animation
- [ ] State transitions
- [ ] Pull-to-refresh
- [ ] Swipe actions
- [ ] Long-press menus
- [ ] Keyboard navigation
- [ ] Screen reader support

### Phase 5: Polish
- [ ] Icon set implementation
- [ ] Animation system
- [ ] Reduced motion support
- [ ] Platform-specific adaptations
- [ ] Accessibility audit
- [ ] Cross-platform consistency review

---

*Document generated: July 2025*  
*Version: 1.0.0-MVP2*  
*Classification: Design System Specification*  
*Total Platforms: 7 (macOS, Windows, Linux, Android, iOS, HarmonyOS, Aurora OS)*

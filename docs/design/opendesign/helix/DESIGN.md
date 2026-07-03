# Helix VPN Design System
> Category: Security & Privacy

Enterprise-grade cross-platform VPN client design system. Spans 8 platforms (macOS, Windows, Linux, Android, iOS, HarmonyOS, Aurora OS, Web Extension) with shared Rust core (`helix-core`), three UI frameworks (Tauri v2 + React for desktop, Flutter + Dart for mobile, Qt6/QML for Aurora), and WASM-based crypto for web. Mandatory light+dark themes, fully customizable color palettes, WCAG 2.1 AA accessibility.

---

## 1. Visual Theme & Atmosphere

### 1.1 Design Philosophy

**Clean. Minimal. Security-focused.** Every pixel communicates trust and protection. The UI makes security feel effortless and approachable — never intimidating.

**Core Principles:**
- **Clarity** — Minimal chrome, focused layouts, zero decorative noise. Every element has a purpose.
- **Trust** — Security feels tangible through color-coded states, clear visual feedback, transparent status indicators.
- **Efficiency** — One-tap connect, gesture shortcuts, contextual options. Primary actions are always immediately accessible.
- **Consistency** — Unified experience across all 8 platforms via shared design tokens and interaction patterns.
- **Accessibility** — WCAG 2.1 AA minimum. Touch targets ≥44×44dp. Screen reader + keyboard navigation.

### 1.2 Design Language

Bleeding-edge enterprise quality. Glassmorphism and subtle depth for elevated surfaces (modals, dialogs). Flat, clean backgrounds for primary surfaces. The connection button is the hero element — a large circular control with dynamic glow states (green/amber/red) and pulse animation.

### 1.3 Platform-Specific Adaptations

| Platform | Aesthetic | Distinctive Elements |
|----------|-----------|---------------------|
| **macOS** | Safari-style minimal chrome, vibrancy | Translucent sidebar, native menu bar, SF Pro |
| **Windows** | Fluent Design (Mica/Acrylic) | Snap layouts, NavigationView sidebar |
| **Linux** | GNOME HIG / GTK theme adaptable | Flat design, CSD integration |
| **Android** | Material 3 + Dynamic Color | Bottom nav bar, Quick Settings tile, edge-to-edge |
| **iOS** | Cupertino design | SF Symbols, blur materials, inset grouped lists |
| **HarmonyOS** | ArkUI + HarmonyOS Sans | Service widgets, distributed capabilities |
| **Aurora OS** | Silica UI, dark-first, gesture-driven | Pulley menus, Cover pages, ambiance integration |
| **Web Extension** | Compact, browser-native | Popup UI (react), toolbar badge, options page |

---

## 2. Color Palette

### 2.1 Brand Primary (Teal/Cyan — Trust & Security)

```
Primary-50:  #E0F2F1    Lightest tint, backgrounds
Primary-100: #B2DFDB    Hover states, light fills
Primary-200: #80CBC4    Secondary highlights
Primary-300: #4DB6AC    Accent elements
Primary-400: #26A69A    Links, active states
Primary-500: #00897B    ★ PRIMARY BRAND COLOR
Primary-600: #00796B    Primary hover
Primary-700: #00695C    Primary pressed/active
Primary-800: #004D40    Strong emphasis surfaces
Primary-900: #00332C    Deepest shade, hero backgrounds

Accent-400: #00BCD4    Secondary accent, data visualization
Accent-500: #00ACC1    Accent hover
Accent-600: #0097A7    Accent pressed
```

> **Color rationale:** Teal (#00897B) was chosen as the primary brand color for its strong associations with trust, security, and technological sophistication — the core brand values of a VPN service. It sits at the intersection of blue (trust, corporate security) and green (safety, permission), avoiding the coldness of pure blue or the alert-like quality of pure green. Teal also performs well across both light and dark themes with adequate WCAG 2.1 AA contrast ratios against the selected surface backgrounds, and supports a wide range of accent and custom palette seeds.

### 2.2 Semantic States (Connection-Driven)

```
◆ Connected       #4CAF50    Green — tunnel active, traffic protected
   Background:     rgba(76, 175, 80, 0.12)  /* Dark theme: 0.18 */

◆ Connecting      #FF9800    Amber — handshake in progress
   Background:     rgba(255, 152, 0, 0.12)

◆ Disconnected    #F44336    Red — no protection
   Background:     rgba(244, 67, 54, 0.12)

◆ Error           #D32F2F    Dark red — critical failure
   Background:     rgba(211, 47, 47, 0.12)

◆ Warning         #F57C00    Caution
   Background:     rgba(245, 124, 0, 0.12)

◆ Info            #1976D2    Information
   Background:     rgba(25, 118, 210, 0.12)

◆ Neutral         #6B7280    Disconnected/idle state
```

### 2.3 Light Theme

```css
/* Surfaces */
--hx-bg-primary:       #F0F4F8    /* App background */
--hx-bg-secondary:     #FFFFFF    /* Cards, panels */
--hx-bg-tertiary:      #E8EDF2    /* Elevated surfaces */
--hx-bg-elevated:      #FFFFFF    /* Modals, dialogs */

/* Text */
--hx-text-primary:     #0A1929    /* Primary text */
--hx-text-secondary:   #4A6375    /* Secondary/muted */
--hx-text-tertiary:    #8BA3B8    /* Placeholder, disabled */
--hx-text-disabled:    #B0C4D4    /* Disabled text */

/* Borders */
--hx-border-default:   #C8D8E4
--hx-border-subtle:    #E2EBF2
--hx-border-focus:     #00897B    /* Primary focus ring */
--hx-border-error:     #D32F2F    /* Error border */
```

### 2.4 Dark Theme

```css
/* Surfaces */
--hx-bg-primary:       #0A1929    /* Deepest background */
--hx-bg-secondary:     #132F4C    /* Cards, panels */
--hx-bg-tertiary:      #1E4976    /* Elevated surfaces */
--hx-bg-elevated:      #243B55    /* Modals, dialogs */

/* Text */
--hx-text-primary:     #F0F4F8    /* Primary text */
--hx-text-secondary:   #8BA3B8    /* Secondary/muted */
--hx-text-tertiary:    #5A7A94    /* Placeholder */
--hx-text-disabled:    #3D5A80    /* Disabled text */

/* Borders */
--hx-border-default:   #1E4976
--hx-border-subtle:    #132F4C
--hx-border-focus:     #00897B
--hx-border-error:     #F87171
```

### 2.5 Latency Color Coding (Traffic Light)

| Latency | Color | Hex | Semantic |
|---------|-------|-----|----------|
| < 50ms  | Green  | #4CAF50 | Excellent |
| 50–100ms | Amber | #FF9800 | Good |
| 100–200ms | Orange | #F57C00 | Fair |
| > 200ms | Red | #F44336 | Poor |

### 2.6 Customizable Palette Architecture

All color tokens use CSS custom properties. To customize:
```css
/* Override in user theme */
:root[data-theme="custom-blue"] {
  --hx-color-action: #2563EB;
  --hx-primary-500: #2563EB;
  --hx-primary-600: #1D4ED8;
}
```

The token system supports _any_ brand seed color at build time via the `helix_design` token compiler. Presets include: Teal (default), Ocean Blue, Midnight, Forest, Ruby, Amethyst.

---

## 3. Typography

### 3.1 Type Scale (Major Third 1.25×, 14px base)

| Token | Size | Weight | Line Ht | Letter | Usage |
|-------|------|--------|---------|--------|-------|
| **Display** | 36px / 2.25rem | 300 Light | 1.2 | -0.5px | Connection status numbers |
| **Headline** | 28px / 1.75rem | 600 SemiBold | 1.3 | -0.25px | Screen titles (desktop) |
| **Title Large** | 22px / 1.375rem | 600 | 1.3 | 0 | Dialog titles, section headers |
| **Title Medium** | 18px / 1.125rem | 600 | 1.4 | 0.15px | Card titles, sub-screens |
| **Title Small** | 16px / 1rem | 500 Medium | 1.4 | 0.1px | List section headers |
| **Body Large** | 16px / 1rem | 400 Regular | 1.5 | 0.5px | Primary body text |
| **Body Medium** | 14px / 0.875rem | 400 | 1.5 | 0.25px | Default body, descriptions |
| **Body Small** | 12px / 0.75rem | 400 | 1.5 | 0.4px | Captions, metadata |
| **Caption** | 11px / 0.6875rem | 500 | 1.4 | 0.5px | Labels, timestamps |
| **Overline** | 10px / 0.625rem | 600 | 1.4 | 1.5px | Category labels (all-caps) |

**Line-height token mapping:** The per-style line heights above map to four canonical `--hx-lh-*` tokens in the design system:
- `--hx-lh-tight: 1.2` — Display
- `--hx-lh-heading: 1.3` — Headline, Title Large
- `--hx-lh-body: 1.4` — Title Medium, Title Small, Caption, Overline
- `--hx-lh-relaxed: 1.5` — Body Large, Body Medium, Body Small

These four line-height tokens cover all typography variants. Use the per-style value from the table when implementing; the named tokens ensure consistent vertical rhythm across platforms.

### 3.2 Monospace (Technical Data)

| Token | Size | Weight | Usage |
|-------|------|--------|-------|
| Mono Large | 18px | 400 | Connection time, large stats |
| Mono Medium | 14px | 400 | IP addresses, server info |
| Mono Small | 12px | 500 | Protocol badges, key fragments |
| Mono Caption | 11px | 400 | Debug info, logs |

Font: `JetBrains Mono`, `Fira Code`, `SF Mono`, `Cascadia Code`

### 3.3 Font Stacks by Platform

| Platform | Stack |
|----------|-------|
| Desktop (Tauri) | `system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif` |
| Android (Flutter) | Roboto (Material 3 default) |
| iOS (Flutter) | SF Pro Text/Display (Cupertino) |
| HarmonyOS | HarmonyOS Sans |
| Aurora OS | Sailfish Silica |
| Web/Extension | Inter, system-ui, -apple-system, sans-serif |

---

## 4. Spacing & Layout

### 4.1 Spacing Scale (4px Base Grid)

| Token | Rem | Px | Usage |
|-------|-----|----|-------|
| `--hx-space-0-5` | 0.125 | 2px | Icon gaps, hairline spacing |
| `--hx-space-1` | 0.25 | 4px | Tight component internal padding |
| `--hx-space-2` | 0.5 | 8px | Default component internal padding |
| `--hx-space-3` | 0.75 | 12px | Small component margin |
| `--hx-space-4` | 1 | 16px | Default padding, card gutters |
| `--hx-space-5` | 1.25 | 20px | Medium spacing |
| `--hx-space-6` | 1.5 | 24px | Section padding, dialog margins |
| `--hx-space-8` | 2 | 32px | Large section gaps |
| `--hx-space-10` | 2.5 | 40px | Section separators |
| `--hx-space-12` | 3 | 48px | Major section padding |
| `--hx-space-16` | 4 | 64px | Hero spacing, connection button margin |
| `--hx-space-20` | 5 | 80px | Large visual separation |

### 4.2 Desktop Window Sizes

| Screen | Width | Height | Purpose |
|--------|-------|--------|---------|
| Main (connected) | 420px | 600px | Primary interface with stats |
| Main (disconnected) | 420px | 480px | Compact disconnected state |
| Server Selection | 520px | 700px | Full server list with search |
| Settings | 640px | 700px | Tabbed settings panel |
| Connection Details | 420px | 550px | Stats overlay/modal |

### 4.3 Responsive Breakpoints

| Token | Width | Target |
|-------|-------|--------|
| `xs` | < 380px | Extension popup minimum |
| `sm` | 380–640px | Extension popup, mobile PWA |
| `md` | 640–768px | Tablet PWA, small panels |
| `lg` | 768–1024px | Tablet landscape, compact desktop |
| `xl` | 1024–1280px | Desktop standard |
| `2xl` | 1280px+ | Desktop wide, admin panel |

### 4.4 Border Radius Scale

| Token | Value | Usage |
|-------|-------|-------|
| `--hx-radius-none` | 0 | Tables, dividers |
| `--hx-radius-sm` | 4px | Small tags, badges |
| `--hx-radius-md` | 8px | Buttons, inputs, small cards |
| `--hx-radius-lg` | 12px | Cards, panels |
| `--hx-radius-xl` | 16px | Large cards, bottom sheets |
| `--hx-radius-2xl` | 24px | Modals, dialogs |
| `--hx-radius-full` | 9999px | Pills, avatars, connection button |

### 4.5 Elevation & Shadows

| Level | Usage | Shadow |
|-------|-------|--------|
| 0 | Flat surfaces | None |
| 1 | Cards, panels | `0 1px 3px rgba(0,0,0,0.12), 0 1px 2px rgba(0,0,0,0.08)` |
| 2 | Elevated cards, dropdowns | `0 4px 6px rgba(0,0,0,0.12), 0 2px 4px rgba(0,0,0,0.08)` |
| 3 | Modals, dialogs | `0 10px 24px rgba(0,0,0,0.16), 0 4px 8px rgba(0,0,0,0.12)` |
| 4 | Connection button glow | `0 20px 40px rgba(0,137,123,0.30)` (state-colored) |

---

## 5. Components

### 5.1 Core Component Inventory

All components ship in light + dark theme variants and are responsive across the target breakpoints.

#### Buttons
| Variant | Bg | Text | Border | Hover | Pressed | Disabled |
|---------|----|------|--------|-------|---------|----------|
| **Primary** | `#00897B` | White | None | `#00796B` | `#00695C` | 38% opacity |
| **Secondary** | Transparent | `#00897B` | 1px `#00897B` | 8% tint bg | 16% tint bg | 38% opacity |
| **Ghost** | Transparent | Text primary | None | 8% overlay | 12% overlay | 38% opacity |
| **Danger** | `#D32F2F` | White | None | `#B71C1C` | `#9A0007` | 38% opacity |
| **Size** | Standard 40px h | Compact 32px h | Pill 48px h | — | — | — |

#### Connection Toggle (Hero Component)
- **Size:** 140×140px circle (desktop), 140×140dp (mobile)
- **Connected:** `#00897B` fill + `#4CAF50` glow pulse (20px blur, 1500ms, opacity 0.2→0.4)
- **Connecting:** `#FF9800` fill + rotating sweep gradient (360°/1500ms)
- **Disconnected:** `#F44336` fill + subtle shadow (static)
- **Center icon:** 48px white (power/shield glyph)
- **Label below:** Title Medium, 600 weight, state color
- **Pulse animation:** scale 1.0 → 1.08 → 1.0, 1500ms ease-in-out (connected only)
- **Ripple effect:** scale 0→4×, opacity 0.5→0, 600ms on connect

#### Server List Item
- Height: 64px (desktop), 72dp (mobile)
- Leading: 36×36px flag icon (8px radius)
- Title: Body Large 500 + subtitle Body Small
- Trailing: Favorite star + protocol badge + chevron
- Selected: Primary tint bg (10% opacity) + checkmark
- Divider: 1px bottom border-subtle
- Hover: Hover overlay (4% / 8% dark/light)

#### Protocol Badge
- Auto-width × 20px height
- Bg: Primary at 10% opacity
- Text: Primary, Caption 500
- Radius: 4px
- Variants: `WG`, `OV`, `IK`, `SS`, `MQ` (multi-hop)

#### Latency Indicator
- 8px colored dot + "23ms" label
- 4px glow (same color, 40% opacity)
- Colors per latency table (§2.5)

#### Input Fields
- Height: 48px (desktop), 56dp (Android), 44dp (iOS)
- Radius: 8px
- Border: 1px default, 2px focus (Primary)
- Error: 2px `#D32F2F` + red helper text
- Font: Body Medium

#### Modal/Dialog
- Min: 360px, Max: 560px
- Radius: 16px (top-only for bottom sheets)
- Shadow: Level 3
- Backdrop: Scrim with fade-in 150ms
- Header: Title Large 600 + close button

#### Toast Notification
- Max 400px wide × 48px height
- Left border: 4px semantic color
- Radius: 8px
- Shadow: Level 2
- Duration: 4s auto-dismiss
- Animation: Slide in top-right, 300ms ease-out

#### Cards (Settings, Stats)
- Bg: secondary surface
- Radius: 12px (desktop), 16dp (mobile)
- Padding: 16px
- Shadow: Level 1

#### Toggle/Switch
- Android: Material 3 Switch, Primary active track
- iOS: CupertinoSwitch, Primary active
- Track: 28×48dp (Android), 32×52dp (iOS)
- Animation: thumb translateX, 200ms

#### Speed Graph / Sparkline
- Height: 80px
- Download line: `#00897B`, 2px stroke, gradient fill
- Upload line: `#00BCD4`, 2px dashed, gradient fill
- Grid: Dotted horizontal lines at 25/50/75%
- Update: 1s interval

#### System Tray Icon (Desktop)
- macOS: Template PNG (status-aware), 22×22
- Windows: .ico 16×16 + overlay badge (green/amber/red dot)
- Linux: StatusNotifierItem PNG 22×22
- Badge dot: 8px diameter, bottom-right

### 5.2 Platform-Specific Components

#### Desktop (Tauri/React)
- Tray context menu (status, connect/disconnect, server, prefs, quit)
- Menu bar (macOS native menu)
- Settings tab bar (General | Connection | Account | Advanced)
- Drag-and-drop server reordering

#### Mobile (Flutter — Android/iOS/HarmonyOS)
- Bottom sheet (50% initial, 90% max, 24dp top radius)
- Tab bar (48dp, Android) / Cupertino nav bar (iOS)
- Pull-to-refresh (64dp threshold, 80dp release)
- Swipe actions (72dp action width)
- Quick Settings tile (Android, 1×1, monochrome)
- Home screen widget (Android 4×1, iOS small+medium)
- Persistent notification with controls (Android foreground service)

#### Aurora OS (Qt6/QML — Silica)
- Pulley menu (pull-down context actions, 56px items)
- Cover page (1:1 square, gradient bg, status dot)
- Context menu (long-press, 48px items)
- Silica-style text fields (underline focus, rectangular)
- Dialog (full-width stacked actions)
- Slider (4px track, 20px thumb)

#### Web Extension
- Popup UI (380px × 480px max)
- Toolbar action badge (connection state color dot)
- Options page (600px, tabbed)
- Native messaging indicator
- Content script injected controls

### 5.3 Component States Matrix

Every interactive component has these states:
- **Default** — idle, enabled
- **Hover** — pointer within bounds (desktop only)
- **Pressed/Active** — mouse down / touch engaged
- **Focused** — keyboard focus ring (2px Primary, 2px offset, keyboard-only)
- **Disabled** — 38% opacity, no interaction
- **Loading** — spinner overlay or skeleton shimmer
- **Error** — red border + error message
- **Selected** — primary tint (checkmark where applicable)

---

## 6. Motion

### 6.1 Animation Tokens

| Token | Duration | Easing | Usage |
|-------|----------|--------|-------|
| Instant | 0ms | — | Reduced motion override |
| Fast | 100ms | ease-out | Button press, micro-interactions |
| Base | 200ms | ease-in-out | Color transitions, toggles, crossfades |
| Slow | 300ms | ease-out | Fade in, slide-in toasts |
| Page | 400ms | cubic-bezier(0.4, 0, 0.2, 1) | Page transitions, panel slides |
| Spring | — | spring(1, 80, 10) | Scale-in, icon morphs |

### 6.2 Connection Flow Animation (Critical Path)

```
DISCONNECTED ──tap──> CONNECTING ──handshake──> CONNECTED
     |                      |                        |
  Red dot (static)    Amber spinner (rotating)   Green glow (pulse)
  No glow             Progress ring (optional)    Checkmark spring-in
                                                   Stats panel slide-up
```

**Phase 1 (0–500ms):** Button press → scale 0.95 (100ms) → ring color red→amber (200ms) → spinner fade-in + rotation start.
**Phase 2 (500ms–2000ms):** Spinner 360°/1500ms continuous → "Connecting..." text fade-in → optional progress ring.
**Phase 3 (2000ms+):** Amber→green (300ms) → spinner fade-out → checkmark scale 0→1 (400ms bounce) → glow pulse loop begins → stats panel slide-up (400ms).

### 6.3 Disconnect Flow
Reverse of connect: green→red (300ms) → stats panel slide-out → glow fades.

### 6.4 Page Transitions

| Platform | Transition | Duration |
|----------|-----------|----------|
| Desktop | Crossfade | 200ms |
| Desktop modal | Fade backdrop (200ms) + scale content (300ms spring) |
| Android | Slide in from right | 300ms |
| iOS | Slide in from right (Cupertino) | 350ms |
| Aurora | Horizontal slide with depth | Silica default |
| HarmonyOS | Smooth spring physics | ArkUI default |

### 6.5 Micro-Interactions

| Interaction | Animation | Duration |
|-------------|-----------|----------|
| Button hover | Scale 1.02, subtle lift | 150ms |
| Button press | Scale 0.98 | 100ms |
| Toggle switch | Thumb translateX + color | 200ms |
| Checkbox | Checkmark stroke draw | 200ms |
| Card hover | translateY -2px, shadow increase | 200ms |
| Copy feedback | Checkmark flash | 500ms |
| Toast | Slide in from top-right | 300ms |

### 6.6 Reduced Motion

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

Fallbacks: pulse → static glow, page transitions → instant, spinners → static indicator + text "Loading..." ripples → instant opacity change, graphs → static (no draw animation).

---

## 7. Voice & Tone

### 7.1 UI Text Principles
- **Confident, calm, competent** — the user's security is not a subject for jokes
- **Short and actionable** — every label answers "what do I need to know/do?"
- **Technical but not intimidating** — explain without jargon panic
- **Positive framing** — "You're protected" not "No leaks detected"

### 7.2 Connection States — Copy

| State | Primary Label | Secondary Label |
|-------|---------------|-----------------|
| Disconnected | "Disconnected" | "Tap to connect securely" |
| Connecting | "Connecting…" | "Establishing secure tunnel" |
| Connected | "Connected" | "US East · 23ms latency" |
| Reconnecting | "Reconnecting…" | "Network changed — restoring tunnel" |
| Error | "Connection Failed" | "Unable to reach server. Tap to retry." |
| Kill Switch Active | "Traffic Blocked" | "Kill switch active — no data leakage" |

### 7.3 Error Messages
Don't blame the user. Don't use "you" in error states unless it's actionable:
- ✓ "Unable to connect to server. Retrying in 15 seconds…"
- ✗ "You entered the wrong server address"

---

## 8. Brand Identity

### 8.1 App Icon
- **Primary:** Teal (`#00897B`) shield/globe motif
- **Sizes:** 16, 32, 48, 72, 96, 128, 144, 192, 256, 512, 1024px
- **Shape:** Adaptive (Android) / Rounded corners (iOS) / Square (desktop)
- **Background:** Transparent on desktop, Primary-900 on mobile splash

### 8.2 System Tray Icon States
- Disconnected: Monochrome outline (B&W template on macOS)
- Connected: Filled icon + green badge dot (8px, bottom-right)
- Connecting: Filled icon + amber badge dot
- Error: Filled icon + red badge dot
- Tooltip: "Helix VPN — <state> (<server>)"

### 8.3 Extension Toolbar Icon
- 16px, 32px PNG for toolbar
- 48px, 128px for extension pages
- Connection state overlay: green/amber/red dot on bottom-right

### 8.4 Brand Voice Application
- Product name always capitalized: "Helix VPN" (not "HelixVPN" in UI)
- Feature names title-case: "Kill Switch", "Split Tunneling", "Multi-Hop"
- Protocol names uppercase: "WireGuard", "Shadowsocks", "MASQUE"

---

## 9. Anti-patterns

### 🚫 Forbidden Design Patterns

1. **Overlapping elements.** Any condition where interactive elements or labels overlap is a critical defect. Use the spacing scale (§4) exclusively for layout — never negative margins, absolute positioning, or z-index stacking to fix overlaps.

2. **Label overlay.** Text MUST NOT overflow its container, clip, or overlay adjacent controls. Text truncation with ellipsis (`text-overflow: ellipsis`) is permitted ONLY for server names in compact layouts — all other text MUST be fully visible.

3. **Color-only indicators.** Never use color alone to convey state. Every status indicator must pair color + icon + text label (WCAG 2.1 AA — not just contrast, but perception).

4. **Decorative noise.** Gradients, shadows, and animations must serve a functional purpose (state communication, affordance). No decorative-only effects.

5. **Inconsistent interaction patterns.** The same action (connect, select server, toggle setting) must behave identically on every platform. Platform UI chrome adapts; core interactions do not.

6. **Skeleton-screen-only loading.** Skeleton screens must be followed by real content within 2s or replaced by an error state — never leave the user looking at a permanent skeleton.

7. **Deep-link shortcuts as test proof.** The realistic user path (launch → browse → tap connect) is the only valid test path. Testing via deep links / intents alone is a bluff — it bypasses the transition paths where real defects live.

8. **"Dark mode = inverted colors."** Dark theme is an authored design, not an automatic inversion. Every color token is explicitly set for dark mode per §2.4.

9. **Hardcoded thresholds.** No latency thresholds, spacing values, or timing constants are hardcoded in UI code — all read from design tokens.

10. **Platform ignorance.** A component designed for one platform's aesthetic MUST NOT be forced onto another without adaptation (e.g., a Material switch on iOS — use CupertinoSwitch).

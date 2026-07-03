# Helix VPN — Interaction Patterns & Animation Specs

**Revision:** 1
**Last modified:** 2026-07-04T12:00:00Z
**Status:** Complete

## 1. Connection Flow (Critical Path)

The connection lifecycle is the most important interaction in the product. Every animation serves to communicate state clearly and build trust.

### Flow Diagram

```
DISCONNECTED ──tap──> CONNECTING ──handshake──> CONNECTED
     │                    │                        │
   Static red         Amber spinner           Green glow
   No glow            Rotating sweep           Pulse loop
   "Disconnected"     "Connecting…"           "Connected"
                                                 │
                                     Stats panel slide-up
                                     Server info appears
                                     Tray icon → green dot

CONNECTED ──tap──> DISCONNECTING ──teardown──> DISCONNECTED
     │                    │                        │
   Green glow          Amber fade              Static red
   Pulse stops          Spinner (brief)         No glow
   Stats slide-down                             "Disconnected"
```

### Timing Table

| Phase | Event | Duration | Easing | Visual |
|-------|-------|----------|--------|--------|
| 1a | Button press | 100ms | ease-out | Scale 0.95, no animation on release |
| 1b | Disconnected→Connecting | 200ms | ease-in-out | Ring color red→amber crossfade |
| 1c | Spinner appears | 300ms | ease-out | Fade in + rotation start |
| 1d | "Connecting…" text | 200ms | ease-out | Fade in |
| 2a | Handshake progress | 500–2500ms | — | Continuous spinner (360°/1500ms) |
| 2b | Optional progress ring | — | — | Arc sweep around button |
| 2c | Error during handshake | — | 800ms flash | Red fast pulse |
| 3a | Connecting→Connected | 300ms | ease-in-out | Ring amber→green crossfade |
| 3b | Spinner→Checkmark | 400ms | spring(1,80,10) | Spinner fades, checkmark scales 0→1 (bounce) |
| 3c | Glow establishes | 500ms | ease-out | 20px blur, 30% opacity, pulse loop begins |
| 3d | Stats panel slide-up | 400ms | cubic-bezier(0.4,0,0.2,1) | translateY from below + opacity 0→1 |
| 3e | Status text update | 200ms | ease-out | "Connected" + server info |
| 3f | Tray icon update | 150ms | — | Badge turns green |

**Disconnect:** Reverse sequence, 800ms total:
- Green→red ring (300ms)
- Stats panel slide-down (300ms)
- Glow fades (200ms)
- Tray icon → outline/disconnected

### Animation Code (CSS Keyframes)

```css
@keyframes connect-pulse {
  0%, 100% { box-shadow: 0 0 20px rgba(76, 175, 80, 0.3); }
  50%      { box-shadow: 0 0 30px rgba(76, 175, 80, 0.5); }
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to   { transform: rotate(360deg); }
}

@keyframes glow-pulse {
  0%, 100% { opacity: 0.2; }
  50%      { opacity: 0.4; }
}

@keyframes ripple {
  0%   { transform: scale(0); opacity: 0.5; }
  100% { transform: scale(4); opacity: 0; }
}
```

---

## 2. Page Transitions

| Platform | Transition | Duration | Easing | Notes |
|----------|-----------|----------|--------|-------|
| **Desktop** | Crossfade | 200ms | ease-in-out | Between pages within window |
| Desktop modal | Fade in backdrop | 200ms | ease-out | Backdrop opacity 0→scrim |
| Desktop modal | Content scale in | 300ms | spring | Content scale 0.95→1 + fade |
| Desktop drawer | Slide from right | 300ms | ease-out | 300px width, scrim behind |
| Desktop settings tab | Crossfade | 200ms | ease-in-out | Tab content only |
| **Android** | Push (slide right) | 300ms | ease-in-out | Enter: translateX 100%→0, Exit: 0→-30% |
| Android | Bottom sheet | 300ms | ease-out | translateY 100%→0 |
| **iOS** | Push (Cupertino) | 350ms | ease-in-out | Standard Cupertino transition |
| iOS | Modal cover | 400ms | ease-out | translateY from bottom |
| iOS | Bottom sheet | 350ms | ease-out (iOS 15+) | Interactive drag dismissal |
| **HarmonyOS** | Smooth spring | 350ms | ArkUI spring | Native ArkUI physics |
| **Aurora** | Slide with depth | Silica default | Silica default | Horizontal parallax |
| **Web** | Instant | 0ms | — | No transitions (popup is small) |

---

## 3. Micro-Interactions

| Interaction | Animation | Duration | Easing | Trigger |
|-------------|-----------|----------|--------|---------|
| Button hover | Scale 1.02, lift 1px | 150ms | ease-out | Mouse enter |
| Button press | Scale 0.98 | 100ms | ease-out | Mouse down |
| Button release | Scale 1.0 | 100ms | ease-out | Mouse up |
| Toggle/Switch on | Thumb translateX + bg color | 200ms | ease-in-out | Tap |
| Toggle/Switch off | Thumb translateX reverse | 200ms | ease-in-out | Tap |
| Checkbox check | Checkmark stroke draw | 200ms | ease-out | Tap |
| Card hover | translateY -2px, shadow inc | 200ms | ease-out | Mouse enter |
| List item tap | Ripple from touch point | 400ms | ease-out | Tap (mobile) |
| Pull-to-refresh | Rotation + arc sweep | 1000ms | linear | Pull past threshold |
| Toast appear | Slide in top-right + fade | 300ms | ease-out | Trigger |
| Toast dismiss | Slide out + fade | 300ms | ease-in | 4000ms timer |
| Copy feedback | Checkmark flash | 500ms | ease-out | Tap copy |
| Focus ring | Outline fade in | 200ms | ease-out | Keyboard Tab |
| Scroll bar | Fade in on scroll | 200ms | ease-out | Scroll start |
| Scroll bar | Fade out | 500ms | ease-in | Scroll end + 1s delay |
| Icon morph | Fill→Outline / Chevron→X | 200ms | ease-in-out | State change |
| Skeleton shimmer | Gradient sweep | 1200ms | linear | Data loading >300ms |
| Progress bar | Smooth width | 300ms | ease-out | Upload/download |
| Server item selected | Tint bg fade in | 200ms | ease-out | Selection |
| Notification badge | Scale bounce 0→1 | 300ms | spring | New notification |

### Reduced Motion Override

```css
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
  
  .connect-toggle {
    /* Static glow — no pulse */
    animation: none;
  }
  
  .connect-toggle.connected {
    box-shadow: 0 0 15px rgba(76, 175, 80, 0.4); /* Fixed glow */
  }
  
  .connect-toggle.connecting {
    /* Static spinner icon, no rotation */
    /* Replace with static "uploading" icon */
  }
}
```

**Fallback behaviors:**
- Pulse → static glow
- Page transitions → instant (no animation)
- Spinners → static indicator + "Loading..." text
- Ripples → instant opacity change
- Graphs → static (no draw animation)
- Toast → instant appear

---

## 4. Gesture & Navigation Patterns

### Desktop Keyboard Navigation

| Key | Action |
|-----|--------|
| Tab | Move focus forward |
| Shift+Tab | Move focus backward |
| Enter/Space | Activate focused element |
| Escape | Close modal/dialog/menu |
| Ctrl/Cmd+K | Quick connect/disconnect |
| Ctrl/Cmd+Shift+S | Open server selection |
| Ctrl/Cmd+, | Open preferences |
| Ctrl/Cmd+Q | Quit application |

Focus ring: 2px `--hx-border-focus` outline, 2px offset. Visible only on keyboard navigation (not mouse click).

### Mobile Gestures

| Gesture | Platform | Action |
|---------|----------|--------|
| Swipe left (server item) | Android | Delete favorite (72dp action) |
| Swipe right (server item) | Android | Quick connect (72dp, green bg) |
| Swipe left (setting item) | Android | Reset to default |
| Pull down (server list) | All mobile | Refresh server list |
| Pull down (main screen) | Aurora | Open pulley menu |
| Long-press (server) | All mobile | Context menu: copy IP, details |
| Long-press (launcher icon) | Android | App shortcuts: connect, servers |
| Back swipe (left edge) | iOS/Android | Navigate back |
| Bottom sheet drag | All mobile | Sheet dismiss (velocity >300dp/s) |

### Haptic Feedback

| Interaction | Platform | Type |
|-------------|----------|------|
| Connect tap | iOS | Heavy impact |
| Connect success | iOS | Success notification |
| Connect failure | iOS | Error notification |
| Swipe action | iOS | Medium impact |
| Long-press trigger | iOS | Medium impact |
| Pull-to-refresh threshold | iOS | Light impact |

---

## 5. Loading States

### Timing-Based Display

| Time | Loading Indicator | Behavior |
|------|------------------|----------|
| 0–300ms | None (instant response) | Content already displayed |
| 300ms–2s | Skeleton loading | Shimmer animation on placeholder shapes |
| >2s | Spinner + status text | Indeterminate progress + "Still connecting…" |

### Skeleton Screen Specification

- Shape: Rounded rectangles matching content layout
- Color: `bg-tertiary` (light) / `bg-secondary` (dark)
- Animation: Shimmer gradient sweep (1200ms linear, diagonal)
- Trigger: Data fetch >300ms

### Spinner Specifications

| Size | Size (px) | Stroke | Usage |
|------|-----------|--------|-------|
| Small | 24×24 | 3px | Inline loading, button replacement |
| Medium | 40×40 | 3px | Section loading |
| Large | 56×56 | 4px | Full-page loading |

Color: `--hx-primary-500` (track: surface variant, 20% opacity)
Duration: 1000ms per rotation

### Progress Bar

- Height: 4px (linear, top of content area)
- Fill: Primary color gradient
- Track: Surface variant
- Animation: Smooth width transition, 300ms
- Determinate: Percentage-based width
- Indeterminate: Continuous sweeping motion

---

## 6. Notification Patterns

### Desktop Notifications

| Event | Type | Action |
|-------|------|--------|
| Connected | Toast (top-right) | — |
| Disconnected | Toast + tray icon highlight | "Tap to reconnect" (tray) |
| Connection error | Toast (error style) | "Tap to retry" |
| Reconnecting | Toast (info style) | — |
| Kill Switch Active | Critical alert | "Traffic blocked — no leakage" |
| Update available | Toast (info) | Auto-download start |
| Auto-connect | Silent | No notification |

### Mobile Notifications

| Platform | Type | Content |
|----------|------|---------|
| Android | Foreground service notification | Status: Connected to US East, Data: 1.2 GB, Disconnect button |
| iOS | System notification (non-intrusive) | Status updates via NE provider |
| HarmonyOS | Notification via HMS Push Kit | Status, data usage |

### Notification Design

```
┌─────────────────────────────────────┐
│  [icon]  Connected to US East    [×]│  ← Toast: 48px height
│          IP: 203.0.113.45           │     4px left border (green)
└─────────────────────────────────────┘
```

---

## 7. Screen Reader Support

### ARIA / Accessibility Attributes

| Element | Attribute | Value |
|---------|-----------|-------|
| Connection toggle | `aria-label` | "Toggle VPN connection — currently connected" |
| Server list items | `aria-label` | "US East server, 23 milliseconds latency" |
| Status text | `aria-live` | "polite" |
| Connection stats | `aria-live` | "polite" |
| Modal | `role` | "dialog" |
| Modal close | `aria-label` | "Close dialog" |
| Navigation | `role` | "navigation" |
| Main content | `role` | "main" |
| Alerts | `role` | "alert" |
| Progress | `role` | "progressbar" |

### Connection State Announcements

| State | Announcement |
|-------|-------------|
| Connected | "Connected to US East, 23 milliseconds latency" |
| Connecting | "Connecting to US East server" |
| Disconnected | "Disconnected — network traffic is not protected" |
| Kill Switch | "Kill switch active — all traffic blocked" |
| Error | "Connection failed — unable to reach server" |

---

## 8. Platform-Specific Pattern Adaptations

### macOS
- Standard traffic light window controls — never custom chrome
- Native menu bar (HelixVPN > About, Preferences, Services, Quit)
- `Cmd+` shortcuts for all primary actions
- Vibrancy effect on sidebar
- SF Pro font throughout

### Windows 11
- Mica material on title bar and background surfaces
- NavigationView sidebar for settings (WinUI-style)
- Snap layout support (window zones)
- Context menu styling (system-native)

### Linux (GNOME)
- Header bar integrated with CSD when available
- Flat design — no unnecessary shadows
- GTK theme detection for color adaptation
- Compact/comfortable density configurable

### Android
- Material 3 with dynamic color (wallpaper-derived primary on 12+)
- Bottom navigation bar (3 items: Home, Servers, Settings)
- Edge-to-edge display behind system bars
- Quick Settings tile (1×1, monochrome)

### iOS
- Cupertino navigation bar with blur
- Inset grouped table sections for settings
- System blur materials (thin, regular, thick)
- SF Symbols for all icons
- Standard iOS back gesture + swipe navigation

### HarmonyOS
- ArkUI design language with smooth spring physics
- HarmonyOS Sans typography
- Service widget (form card) for home screen
- Distributed capability framework (future cross-device)

### Aurora OS
- Silica gesture navigation (no bottom nav)
- Pulley menu for primary actions
- Cover page for background status
- Ambiance-aware coloring
- Sailfish Silica component library (no Qt Quick Controls)

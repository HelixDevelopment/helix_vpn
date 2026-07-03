# Helix VPN — Button Component

**Part of:** Component Library
**Type:** Core interactive component

## Variants

| Variant | Background | Text | Border | Hover | Pressed | Disabled |
|---------|-----------|------|--------|-------|---------|----------|
| **Primary** | `--hx-primary-500` | White | None | `--hx-primary-600` | `--hx-primary-700` | 38% opacity |
| **Secondary** | Transparent | `--hx-primary-500` | 1px `--hx-primary-500` | `--hx-overlay-hover` | `--hx-overlay-pressed` | 38% opacity |
| **Ghost** | Transparent | `--hx-text-primary` | None | `--hx-overlay-hover` | `--hx-overlay-pressed` | 38% opacity |
| **Danger** | `--hx-semantic-error` | White | None | #B71C1C | #9A0007 | 38% opacity |
| **Icon** | Transparent | `--hx-text-primary` | None | `--hx-overlay-hover` | `--hx-overlay-pressed` | 38% opacity |

## Sizes

| Size | Height | Padding-X | Radius | Usage |
|------|--------|-----------|--------|-------|
| Standard | `--hx-btn-height-standard` (40px) | `--hx-btn-padding-x` (24px) | `--hx-btn-radius` (8px) | Default buttons |
| Compact | `--hx-btn-height-compact` (32px) | 16px | 8px | Dense UIs, toolbars |
| Pill | 48px | 32px | `--hx-radius-full` (9999px) | Prominent CTAs |

## States

All button variants support these states:
- **Default** — idle, enabled (full opacity)
- **Hover** — 150ms ease-out transition, pointer cursor (desktop only)
- **Pressed** — scale 0.98, 100ms ease-out
- **Focused** — 2px `--hx-border-focus` outline, 2px offset (keyboard-only)
- **Loading** — spinner overlay replacing icon, button remains clickable but disabled
- **Disabled** — 38% opacity, `cursor: not-allowed`, no hover effect

## Spacing

- Icon + text gap: 8px (`--hx-space-2`)
- Multiple buttons in a group: 8px gap between them

## Platform Adaptation

| Platform | Adaptation |
|----------|-----------|
| Desktop | Native hover + focus via CSS |
| Android | Material 3 button with ripple effect |
| iOS | Cupertino button styling (no hover) |
| Aurora | Silica button (rectangular, no radius) |
| Web | Standard HTML button + CSS |

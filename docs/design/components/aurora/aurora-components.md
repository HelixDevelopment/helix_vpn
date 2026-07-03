# Aurora OS Components (Qt6/QML + Silica)

## Platform-Specific Components

### Cover Page
- 1:1 square aspect ratio (app cover)
- Bg: Radial gradient Primary-800 → Primary-900
- Status dot: 12px, state-colored (green/amber/red)
- Dot glow: 20px blur at 30%
- State text: "Connected" / "Connecting…" / "Disconnected"
- Server name: caption below status
- Ambiance: respects system ambiance settings (chat/amber/blue)

### Pulley Menu
- Trigger: pull down from top edge
- Bg: Primary-900, dark theme
- Items: 56px height, white text, left-aligned 24px icons
- Selection: rgba(255,255,255,0.15)
- Animation: slide down from top, items stagger 50ms
- Typical items: Connect, Disconnect, Select Server, Settings, About, Quit

### Silica List Item
- Height: 80px (standard), 64px (compact)
- Bg: transparent (no card)
- Primary label: Body Large, text-primary
- Secondary label: Body Small, text-secondary
- Leading: 32px icon left-aligned, 16px padding
- Divider: 1px bottom, border-subtle
- Pressed: overlay-pressed (rgba white 8%)

### Silica Text Field
- Bg: bg-secondary
- Border: 1px border-default, no radius (rectangular Silica style)
- Height: 48px
- Font: Body Medium
- Placeholder: text-tertiary
- Focus: 2px Primary underline
- Error: 2px #D32F2F underline
- No floating label pattern (Silica convention)

### Aurora Dialog
- Width: 90% screen, max 480px
- Bg: bg-elevated
- Radius: 8px (Silica)
- Header: Title Medium 600
- Content: Body Medium
- Actions: full-width stacked (no side-by-side)
- Entry: fade 200ms + slight scale

### Aurora Slider
- Track: 4px height
- Active track: Primary
- Inactive track: bg-tertiary
- Thumb: 20px circle, Primary fill, 2px white border
- Value label: floating tooltip above thumb

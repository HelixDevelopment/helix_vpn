# Web Extension Components (MV3 + React)

## Platform-Specific Components

### Popup UI
- Window: 380px × 480px max (browser-enforced limit on popup)
- Resizable: no (fixed popup size)
- Header: 44px, Helix logo + settings gear
- Connection state: mini connection toggle (80px)
- Status card: compact, monospace techncial info
- Footer: link to dashboard, version string

### Toolbar Icon
- Size: 16px, 32px PNG
- State overlay: 6px colored dot bottom-right
- States: disconnected (outline), connected (filled green), connecting (amber)
- Badge text: none (icon-only, color is status)

### Options Page
- Full browser tab: 600px width
- Tab nav: General | Connection | Account
- Sections per tab: card-based layout
- Save/Cancel footer bar

### Native Messaging Status
- Indicator in popup: green dot + "Connected to desktop app"
- OR grey dot + "Desktop app not detected"
- Triggers: install prompt if native messaging host missing

### Content Script UI
- Minimal, non-intrusive
- Floating badge when extension detects secure page
- WebRTC leak protection toggle
- Injected at `document_start` via `"run_at"` in manifest

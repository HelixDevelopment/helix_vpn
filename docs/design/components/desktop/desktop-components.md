# Desktop Components (Tauri v2 + React)

## Platform-Specific Components

### System Tray / Menu Bar

#### macOS (MenuBarExtra)
- StatusItem with template image (adapts to dark/light menu bar)
- Dropdown menu: status header, connect/disconnect, server selection, preferences, quit
- Tooltip: "Helix VPN — Connected to US East"
- Badge: 8px colored dot (green/amber/red)

#### Windows (NotifyIcon)
- Taskbar icon with context menu
- Balloon notifications for connection events
- Jump list: recent servers, connect, disconnect

#### Linux (AppIndicator / StatusNotifierItem)
- Indicator icon in notification area
- Standard indicator menu pattern
- GTK theme-compatible icons

### Window Controls

#### macOS
- Standard traffic light (red/yellow/green) — no custom chrome
- Titlebar: Helix VPN app name (optional)
- Window: Rounded corners (system default)
- Vibrancy: `NSVisualEffectView` for sidebar/bg

#### Windows 11
- Mica material on title bar (translucent wallpaper-aware bg)
- Custom frame with snap layout zones
- System backdrop effect

#### Linux
- CSD (Client-Side Decoration) integration when available
- Adaptive to GNOME/Budgie/KDE theming
- Flat header bar

### Settings Navigation

#### Desktop Tab Bar
- Height: 40px
- Tabs: General | Connection | Account | Advanced
- Active: Primary underline (2px height, full width transition 200ms)
- Inactive: Text-secondary, hover: overlay-hover
- Bg: Transparent (no card), on bg-primary surface

# Mobile Components (Flutter — Android/iOS/HarmonyOS)

## Platform-Specific Components

### Bottom Sheet
- Initial height: 50% screen
- Max height: 90% screen
- Top radius: 24dp
- Handle: 4dp × 32dp, #8BA3B8, 2dp radius
- Entry: slide up 300ms ease-out
- Drag dismiss: velocity threshold 300dp/s

### Tab Bar

#### Android (Material 3)
- Height: 48dp
- Indicator: 2dp height, Primary color, smooth slide transition
- Active: Primary, 500 weight
- Inactive: Secondary, 400 weight

#### iOS (Cupertino)
- Navigation bar with large title
- Bottom tab bar (if 3+ sections)
- Segmented control for toggles

#### HarmonyOS
- ArkUI native tab component
- Adaptive to system theme

### Pull-to-Refresh
- Trigger: Pull down past 64dp
- Indicator: Circular progress, Primary
- Release: at 80dp
- Haptic: Light at threshold (iOS)
- Snap back: 200ms

### Swipe Actions
- Width: 72dp per action
- Bg: Semantic color
- Icon: 24dp, white
- Full swipe: auto-execute

### Android Quick Settings Tile
- Size: 1×1 standard tile
- Active: Primary bg, white icon
- Inactive: Surface bg, secondary icon
- Label: "Helix VPN" / "Helix VPN: ON"
- Long-press → opens app
- Requires `TILE_SERVICE` in manifest

### Android Widget (4×1)
- Connection status icon + server name
- Toggle connect/disconnect
- Updates via RemoteViews

### iOS Widget (Small + Medium)
- Small: Status icon + connection state
- Medium: Status + server name + toggle
- Updates via TimelineProvider

const sidebarItems = [
  { id: 'design-system', label: 'Design System', badge: 'ready' },
  { id: 'design-files', label: 'Design Files', badge: '2' },
  { id: 'preview', label: 'Preview', badge: 'html' },
];

const sidebarStyles = {
  wrap: { width: 280, minHeight: 640, borderRight: '1px solid var(--color-border, #dfe3e8)', background: 'var(--color-background-soft, #fff)', padding: 16 },
  header: { display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 18 },
  mark: { width: 34, height: 34, borderRadius: 10, background: 'var(--color-primary, #00b96b)', color: '#fff', display: 'grid', placeItems: 'center', fontWeight: 700 },
  item: { display: 'grid', gridTemplateColumns: '1fr auto', gap: 10, alignItems: 'center', padding: '11px 12px', borderRadius: 10, marginBottom: 8, border: '1px solid transparent' },
  active: { borderColor: 'var(--color-primary, #00b96b)', background: 'var(--color-primary-soft, rgba(0,185,107,.1))' },
  badge: { fontSize: 11, color: 'var(--color-text-secondary, #73777f)' },
};

function Sidebar({ title = 'Helix VPN', activeId = 'design-system', items = sidebarItems }) {
  return (
    <nav style={sidebarStyles.wrap} aria-label={title}>
      <div style={sidebarStyles.header}>
        <div style={sidebarStyles.mark}>{title.slice(0, 1)}</div>
        <strong>{title}</strong>
      </div>
      {items.map((item) => (
        <button key={item.id} type="button" style={{ ...sidebarStyles.item, ...(item.id === activeId ? sidebarStyles.active : {}) }}>
          <span>{item.label}</span>
          <span style={sidebarStyles.badge}>{item.badge}</span>
        </button>
      ))}
    </nav>
  );
}

window.Sidebar = Sidebar;

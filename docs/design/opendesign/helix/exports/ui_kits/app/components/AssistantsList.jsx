const assistantItems = [
  { id: 'default', name: 'Helix VPN reviewer', meta: 'Design review workspace', active: true },
  { id: 'tokens', name: 'Token specialist', meta: 'Colors, type, spacing, and states', active: false },
  { id: 'components', name: 'Component reviewer', meta: 'Cards, inputs, messages, and navigation', active: false },
];

const assistantsListStyles = {
  panel: { width: 280, borderRight: '1px solid var(--color-border, #dfe3e8)', background: 'var(--color-surface, #fff)', padding: 14, display: 'grid', alignContent: 'start', gap: 10 },
  header: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 },
  row: { display: 'grid', gridTemplateColumns: '32px 1fr', gap: 10, alignItems: 'center', padding: 10, borderRadius: 10, border: '1px solid transparent' },
  active: { borderColor: 'var(--color-primary, #00b96b)', background: 'var(--color-primary-soft, rgba(0,185,107,.1))' },
  avatar: { width: 32, height: 32, borderRadius: 10, background: 'var(--color-background-soft, #f7f8fa)', display: 'grid', placeItems: 'center', fontWeight: 700 },
  meta: { color: 'var(--color-text-secondary, #73777f)', fontSize: 12 },
};

function AssistantsList({ items = assistantItems }) {
  return (
    <aside style={assistantsListStyles.panel} aria-label="Assistants">
      <header style={assistantsListStyles.header}>
        <strong>Assistants</strong>
        <button type="button">New</button>
      </header>
      {items.map((item) => (
        <button key={item.id} type="button" style={{ ...assistantsListStyles.row, ...(item.active ? assistantsListStyles.active : {}) }}>
          <span style={assistantsListStyles.avatar}>{item.name.slice(0, 1)}</span>
          <span>
            <strong>{item.name}</strong>
            <small style={assistantsListStyles.meta}>{item.meta}</small>
          </span>
        </button>
      ))}
    </aside>
  );
}

window.AssistantsList = AssistantsList;

const inputActions = ['Attach', 'Source', 'Revise'];

const inputBarStyles = {
  wrap: { border: '1px solid var(--color-border, #dfe3e8)', borderRadius: 14, background: 'var(--color-surface, #fff)', padding: 12, display: 'grid', gap: 10 },
  field: { minHeight: 82, border: 0, outline: 0, resize: 'vertical', font: 'inherit', color: 'var(--color-text, #202124)' },
  toolbar: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 12 },
  actions: { display: 'flex', flexWrap: 'wrap', gap: 8 },
  chip: { border: '1px solid var(--color-border, #dfe3e8)', borderRadius: 999, padding: '6px 10px', background: 'var(--color-background-soft, #f7f8fa)' },
  send: { border: 0, borderRadius: 10, padding: '9px 14px', background: 'var(--color-primary, #00b96b)', color: '#fff', fontWeight: 700 },
};

function InputBar({ title = 'Helix VPN prompt', actions = inputActions }) {
  return (
    <form style={inputBarStyles.wrap} aria-label={title}>
      <textarea style={inputBarStyles.field} placeholder="Describe the design revision, evidence to inspect, or preview card to improve." />
      <div style={inputBarStyles.toolbar}>
        <div style={inputBarStyles.actions}>
          {actions.map((action) => <button key={action} type="button" style={inputBarStyles.chip}>{action}</button>)}
        </div>
        <button type="submit" style={inputBarStyles.send}>Send</button>
      </div>
    </form>
  );
}

window.InputBar = InputBar;

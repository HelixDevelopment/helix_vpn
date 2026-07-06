const messageBubbleStyles = {
  bubble: { maxWidth: 680, border: '1px solid var(--color-border, #dfe3e8)', borderRadius: 14, background: 'var(--color-surface, #fff)', padding: 14, display: 'grid', gap: 8 },
  user: { marginLeft: 'auto', background: 'var(--color-primary-soft, rgba(0,185,107,.1))', borderColor: 'var(--color-primary, #00b96b)' },
  meta: { display: 'flex', justifyContent: 'space-between', gap: 12, color: 'var(--color-text-secondary, #73777f)', fontSize: 12 },
  text: { margin: 0, lineHeight: 1.55 },
  status: { justifySelf: 'start', borderRadius: 999, padding: '4px 8px', background: 'var(--color-background-soft, #f7f8fa)', fontSize: 12 },
};

function MessageBubble({ role = 'Helix VPN', text = 'Source-backed design-system guidance belongs in compact, reviewable message surfaces.', status = 'grounded', fromUser = false }) {
  return (
    <article style={{ ...messageBubbleStyles.bubble, ...(fromUser ? messageBubbleStyles.user : {}) }}>
      <div style={messageBubbleStyles.meta}>
        <strong>{role}</strong>
        <span>{status}</span>
      </div>
      <p style={messageBubbleStyles.text}>{text}</p>
      <span style={messageBubbleStyles.status}>Uses captured evidence</span>
    </article>
  );
}

window.MessageBubble = MessageBubble;

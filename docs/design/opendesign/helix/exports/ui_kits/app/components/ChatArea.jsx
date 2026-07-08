const chatMessages = [
  { id: 'user', role: 'You', text: 'Create a compact review surface from the captured source evidence.' },
  { id: 'assistant', role: 'Helix VPN', text: 'The system uses focused preview cards, source-backed tokens, and reusable app-kit components.' },
];

const chatAreaStyles = {
  wrap: { minHeight: 640, background: 'var(--color-background, #f7f8fa)', display: 'grid', gridTemplateRows: 'auto 1fr auto' },
  header: { minHeight: 54, borderBottom: '1px solid var(--color-border, #dfe3e8)', padding: '0 18px', display: 'flex', alignItems: 'center', justifyContent: 'space-between', background: 'var(--color-surface, #fff)' },
  stream: { padding: 22, display: 'grid', alignContent: 'start', gap: 14, overflow: 'auto' },
  note: { border: '1px solid var(--color-border, #dfe3e8)', borderRadius: 12, background: 'var(--color-surface, #fff)', padding: 14 },
  composerSlot: { borderTop: '1px solid var(--color-border, #dfe3e8)', background: 'var(--color-surface, #fff)', padding: 16 },
};

function ChatArea({ title = 'Helix VPN review', messages = chatMessages }) {
  const InputBar = window.InputBar;
  const MessageBubble = window.MessageBubble;
  return (
    <section style={chatAreaStyles.wrap} aria-label={title}>
      <header style={chatAreaStyles.header}>
        <strong>{title}</strong>
        <button type="button">Open source context</button>
      </header>
      <div style={chatAreaStyles.stream}>
        {messages.map((message) => (
          <MessageBubble key={message.id} role={message.role} text={message.text} fromUser={message.id === 'user'} />
        ))}
      </div>
      <div style={chatAreaStyles.composerSlot}><InputBar title={title + ' prompt'} /></div>
    </section>
  );
}

window.ChatArea = ChatArea;

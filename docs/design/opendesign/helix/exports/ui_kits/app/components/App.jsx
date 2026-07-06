const reviewModules = [
  { id: 'colors', label: 'Color review', summary: 'Primary, theme, and semantic color cards' },
  { id: 'type', label: 'Typography review', summary: 'Specimens, scale, and dense metadata rhythm' },
  { id: 'components', label: 'Component review', summary: 'Buttons, inputs, cards, and feedback states' },
];

const appStyles = {
  shell: { display: 'grid', gridTemplateColumns: '280px minmax(240px, 300px) 1fr', minHeight: '720px', background: 'var(--color-background, #f7f8fa)', color: 'var(--color-text, #202124)' },
  workspace: { padding: '24px', display: 'grid', gap: '16px', alignContent: 'start' },
  card: { border: '1px solid var(--color-border, #dfe3e8)', borderRadius: 12, background: 'var(--color-surface, #fff)', padding: '16px' },
  eyebrow: { color: 'var(--color-text-secondary, #73777f)', fontSize: 12, fontWeight: 600, textTransform: 'uppercase', letterSpacing: 0 },
};

function App({ title = 'Helix VPN', modules = reviewModules, summary = 'Source-backed design-system workspace' }) {
  const Sidebar = window.Sidebar;
  const AssistantsList = window.AssistantsList;
  const ChatArea = window.ChatArea;
  return (
    <main style={appStyles.shell}>
      <Sidebar title={title} />
      <AssistantsList />
      <section style={appStyles.workspace}>
        <span style={appStyles.eyebrow}>Review surface</span>
        <h1>{title}</h1>
        <p>{summary}</p>
        <ChatArea title={title + ' workspace'} />
        <section style={{ display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', gap: 12 }}>
          {modules.map((module) => (
            <article key={module.id} style={appStyles.card}>
              <strong>{module.label}</strong>
              <p>{module.summary}</p>
            </article>
          ))}
        </section>
      </section>
    </main>
  );
}

window.App = App;

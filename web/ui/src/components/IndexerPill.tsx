function indexerHue(name: string): number {
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }
  return ((hash % 360) + 360) % 360;
}

export default function IndexerPill({ name }: { name: string }) {
  const hue = indexerHue(name);
  return (
    <span
      style={{
        display: "inline-block",
        padding: "2px 6px",
        borderRadius: 4,
        fontSize: 10,
        fontWeight: 600,
        letterSpacing: "0.03em",
        background: `hsla(${hue}, 60%, 55%, 0.15)`,
        color: `hsl(${hue}, 70%, 65%)`,
        whiteSpace: "nowrap",
      }}
    >
      {name}
    </span>
  );
}

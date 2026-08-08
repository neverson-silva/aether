import React from "react";

function renderBlock(block: string, i: number): React.ReactNode {
  if (block.startsWith("```")) {
    const lines = block.split("\n");
    const code = lines.slice(1, lines.length - 1).join("\n");
    return (
      <pre key={i} className="bg-surface-container-lowest border border-outline-variant rounded-md p-sm overflow-x-auto font-code-md text-code-md my-sm">
        {code}
      </pre>
    );
  }
  if (block.startsWith("### ")) return <h4 key={i} className="font-label-caps text-label-caps text-primary uppercase mt-md mb-sm">{block.slice(4)}</h4>;
  if (block.startsWith("## ")) return <h3 key={i} className="font-headline-sm text-headline-sm text-on-surface mt-lg mb-sm">{block.slice(3)}</h3>;
  if (block.startsWith("# ")) return <h2 key={i} className="font-headline-md text-headline-md text-on-surface mt-lg mb-sm">{block.slice(2)}</h2>;
  if (block.startsWith("- ") || block.startsWith("* ")) {
    return (
      <ul key={i} className="list-disc list-inside my-sm font-body-sm text-body-sm text-on-surface-variant">
        {block.split("\n").filter((l) => l.trim()).map((l, j) => (
          <li key={j}>{l.replace(/^[-*] /, "")}</li>
        ))}
      </ul>
    );
  }
  if (block.startsWith("> ")) return <blockquote key={i} className="border-l-2 border-primary/50 pl-sm my-sm font-body-sm text-body-sm text-on-surface-variant italic">{block.slice(2)}</blockquote>;
  const text = block
    .replace(/\*\*(.+?)\*\*/g, (_, m) => `**${m}**`)
    .replace(/`([^`]+)`/g, (_, m) => `<c>${m}</c>`);
  const parts = text.split(/(\*\*.*?\*\*|<c>.*?<\/c>)/g);
  return (
    <p key={i} className="font-body-sm text-body-sm text-on-surface-variant my-sm">
      {parts.map((part, j) => {
        if (part.startsWith("**") && part.endsWith("**")) return <strong key={j} className="text-on-surface">{part.slice(2, -2)}</strong>;
        if (part.startsWith("<c>") && part.endsWith("</c>")) return <code key={j} className="bg-surface-container-lowest border border-outline-variant rounded px-1 font-code-md text-code-md text-primary">{part.slice(3, -4)}</code>;
        return <React.Fragment key={j}>{part}</React.Fragment>;
      })}
    </p>
  );
}

export function Markdown({ text }: { text: string }) {
  const blocks = text.split(/\n{2,}/);
  return <div>{blocks.map((b, i) => renderBlock(b.trim(), i))}</div>;
}

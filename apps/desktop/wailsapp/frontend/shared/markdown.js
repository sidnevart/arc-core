import { escapeHTML } from "./ux.js";

function renderInline(text) {
  return escapeHTML(text)
    .replace(/`([^`]+)`/g, "<code>$1</code>")
    .replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>")
    .replace(/\*([^*]+)\*/g, "<em>$1</em>")
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" target="_blank" rel="noreferrer">$1</a>');
}

export function renderMarkdown(text) {
  const lines = String(text || "").replace(/\r\n/g, "\n").split("\n");
  const html = [];
  let listItems = [];
  let quoteLines = [];
  let codeFence = null;
  const flushList = () => {
    if (!listItems.length) return;
    html.push(`<ul>${listItems.map((item) => `<li>${renderInline(item)}</li>`).join("")}</ul>`);
    listItems = [];
  };
  const flushQuote = () => {
    if (!quoteLines.length) return;
    html.push(`<blockquote>${quoteLines.map((line) => `<p>${renderInline(line)}</p>`).join("")}</blockquote>`);
    quoteLines = [];
  };
  const flushCode = () => {
    if (!codeFence) return;
    html.push(`<pre class="code-block"><code>${escapeHTML(codeFence.body.join("\n"))}</code></pre>`);
    codeFence = null;
  };
  for (const rawLine of lines) {
    const line = rawLine.replace(/\t/g, "  ");
    const trimmed = line.trim();
    if (trimmed.startsWith("```")) {
      if (codeFence) {
        flushCode();
      } else {
        flushList();
        flushQuote();
        codeFence = { body: [] };
      }
      continue;
    }
    if (codeFence) {
      codeFence.body.push(line);
      continue;
    }
    if (!trimmed) {
      flushList();
      flushQuote();
      continue;
    }
    if (/^>/.test(trimmed)) {
      flushList();
      quoteLines.push(trimmed.replace(/^>\s?/, ""));
      continue;
    }
    flushQuote();
    if (/^#{1,4}\s+/.test(trimmed)) {
      flushList();
      const level = Math.min(4, trimmed.match(/^#+/)[0].length);
      html.push(`<h${level + 2}>${renderInline(trimmed.replace(/^#{1,4}\s+/, ""))}</h${level + 2}>`);
      continue;
    }
    if (/^[-*]\s+/.test(trimmed) || /^\d+\.\s+/.test(trimmed)) {
      listItems.push(trimmed.replace(/^[-*]\s+/, "").replace(/^\d+\.\s+/, ""));
      continue;
    }
    if (/^---+$/.test(trimmed)) {
      flushList();
      html.push("<hr />");
      continue;
    }
    flushList();
    html.push(`<p>${renderInline(trimmed)}</p>`);
  }
  flushList();
  flushQuote();
  flushCode();
  return html.join("") || `<p>${renderInline(String(text || ""))}</p>`;
}

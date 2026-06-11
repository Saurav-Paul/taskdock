// Tiny markdown renderer for read-only contexts (comments).
// Supports headings, bold, italic, inline code, code blocks, links, images, lists, tables.
// ```mermaid fences are rendered as live diagrams via <MermaidDiagram>.

import { MermaidDiagram } from "./MermaidDiagram";

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

function inline(s: string): string {
  return s
    .replace(/!\[([^\]]*)\]\(([^)\s]+)\)/g, '<img src="$2" alt="$1" />')
    .replace(/`([^`]+)`/g, "<code>$1</code>")
    .replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>")
    .replace(/(^|\W)\*([^*]+)\*/g, "$1<em>$2</em>")
    .replace(/(^|\W)_([^_]+)_/g, "$1<em>$2</em>")
    .replace(
      /\[([^\]]+)\]\((https?:\/\/[^)\s]+)\)/g,
      '<a href="$2" target="_blank" rel="noopener noreferrer">$1</a>'
    );
}

// GFM table header separator: | --- | :---: | ---: | (leading/trailing pipes optional)
const TABLE_SEPARATOR = /^\s*\|?\s*:?-+:?\s*(\|\s*:?-+:?\s*)*\|?\s*$/;

function isTableRow(line: string): boolean {
  return line.trim().startsWith("|");
}

function splitTableRow(line: string): string[] {
  let s = line.trim();
  if (s.startsWith("|")) s = s.slice(1);
  if (s.endsWith("|")) s = s.slice(0, -1);
  return s.split("|").map((cell) => cell.trim());
}

function renderTable(rows: string[]): string {
  const header = splitTableRow(rows[0]).map((c) => `<th>${inline(c)}</th>`).join("");
  const out = [`<table><thead><tr>${header}</tr></thead>`];
  const body = rows.slice(2);
  if (body.length) {
    out.push("<tbody>");
    for (const row of body) {
      out.push(`<tr>${splitTableRow(row).map((c) => `<td>${inline(c)}</td>`).join("")}</tr>`);
    }
    out.push("</tbody>");
  }
  out.push("</table>");
  return out.join("");
}

export function markdownToHtml(md: string): string {
  const lines = escapeHtml(md).split("\n");
  const out: string[] = [];
  let inCode = false;
  let listType: "ul" | "ol" | null = null;
  let para: string[] = [];

  const flushPara = () => {
    if (para.length) {
      out.push(`<p>${inline(para.join(" "))}</p>`);
      para = [];
    }
  };
  const closeList = () => {
    if (listType) {
      out.push(`</${listType}>`);
      listType = null;
    }
  };

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    if (line.trim().startsWith("```")) {
      flushPara();
      closeList();
      out.push(inCode ? "</code></pre>" : "<pre><code>");
      inCode = !inCode;
      continue;
    }
    if (inCode) {
      out.push(line);
      continue;
    }
    // GFM table: a | row followed by a header separator line.
    if (isTableRow(line) && i + 1 < lines.length && isTableRow(lines[i + 1]) && TABLE_SEPARATOR.test(lines[i + 1])) {
      flushPara();
      closeList();
      const rows: string[] = [];
      while (i < lines.length && isTableRow(lines[i])) {
        rows.push(lines[i]);
        i++;
      }
      i--; // step back; the for-loop increments past the table
      out.push(renderTable(rows));
      continue;
    }
    const heading = line.match(/^(#{1,6})\s+(.*)$/);
    const bullet = line.match(/^\s*[-*]\s+(.*)$/);
    const ordered = line.match(/^\s*\d+\.\s+(.*)$/);
    if (heading) {
      flushPara();
      closeList();
      const level = heading[1].length;
      out.push(`<h${level}>${inline(heading[2])}</h${level}>`);
    } else if (bullet || ordered) {
      flushPara();
      const type = bullet ? "ul" : "ol";
      if (listType !== type) {
        closeList();
        out.push(`<${type}>`);
        listType = type;
      }
      out.push(`<li>${inline((bullet ?? ordered)![1])}</li>`);
    } else if (line.trim() === "") {
      flushPara();
      closeList();
    } else {
      closeList();
      para.push(line.trim());
    }
  }
  flushPara();
  closeList();
  if (inCode) out.push("</code></pre>");
  return out.join("\n");
}

// Split markdown into plain segments and ```mermaid fences so diagrams can be
// React-rendered islands while everything else keeps the string→HTML pipeline.
type Segment = { type: "md"; text: string } | { type: "mermaid"; code: string };

const MERMAID_FENCE = /^```\s*mermaid\s*$/;
const FENCE = /^```/;

function splitMermaidSegments(md: string): Segment[] {
  const lines = md.split("\n");
  const segments: Segment[] = [];
  let plain: string[] = [];
  let inCode = false;

  const flushPlain = () => {
    if (plain.length) {
      segments.push({ type: "md", text: plain.join("\n") });
      plain = [];
    }
  };

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    if (!inCode && MERMAID_FENCE.test(line.trim())) {
      flushPlain();
      const code: string[] = [];
      i++;
      while (i < lines.length && !FENCE.test(lines[i].trim())) {
        code.push(lines[i]);
        i++;
      }
      // i now sits on the closing fence (or EOF) — consumed by the loop.
      segments.push({ type: "mermaid", code: code.join("\n") });
      continue;
    }
    if (FENCE.test(line.trim())) inCode = !inCode;
    plain.push(line);
  }
  flushPlain();
  return segments;
}

export function Markdown({ source }: { source: string }) {
  const segments = splitMermaidSegments(source);
  if (!segments.some((s) => s.type === "mermaid")) {
    return <div className="markdown" dangerouslySetInnerHTML={{ __html: markdownToHtml(source) }} />;
  }
  return (
    <div className="markdown">
      {segments.map((seg, i) =>
        seg.type === "mermaid" ? (
          <MermaidDiagram key={i} source={seg.code} />
        ) : (
          <div key={i} dangerouslySetInnerHTML={{ __html: markdownToHtml(seg.text) }} />
        )
      )}
    </div>
  );
}

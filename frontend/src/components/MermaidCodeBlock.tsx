// CodeBlock extension whose NodeView keeps the code fully editable but, when
// the block's language is "mermaid", shows a live rendered preview directly
// beneath it (debounced). Only the in-editor DOM changes — parse/serialize
// behavior is inherited untouched, so tiptap-markdown round-trips ```mermaid
// fences byte-identically.

import { useEffect, useState } from "react";
import CodeBlock from "@tiptap/extension-code-block";
import {
  NodeViewContent,
  NodeViewWrapper,
  ReactNodeViewRenderer,
  type NodeViewProps,
} from "@tiptap/react";
import { MermaidDiagram } from "./MermaidDiagram";

function CodeBlockView({ node }: NodeViewProps) {
  const isMermaid = node.attrs.language === "mermaid";
  const source = node.textContent;
  // Debounce re-renders while typing; first value renders immediately.
  const [debounced, setDebounced] = useState(source);

  useEffect(() => {
    if (!isMermaid) return;
    const t = setTimeout(() => setDebounced(source), 300);
    return () => clearTimeout(t);
  }, [source, isMermaid]);

  return (
    <NodeViewWrapper className="code-block-view">
      <pre>
        <NodeViewContent as="code" />
      </pre>
      {isMermaid && <MermaidDiagram source={debounced} />}
    </NodeViewWrapper>
  );
}

export const MermaidCodeBlock = CodeBlock.extend({
  addNodeView() {
    return ReactNodeViewRenderer(CodeBlockView);
  },
});

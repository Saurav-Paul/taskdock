// Shared mermaid renderer used by the comments markdown renderer and the
// editor's live preview. Mermaid (~heavy) is lazy-loaded on first use so it
// stays out of the main bundle.

import { useEffect, useRef, useState } from "react";

type MermaidApi = typeof import("mermaid").default;

let mermaidPromise: Promise<MermaidApi> | null = null;

function loadMermaid(): Promise<MermaidApi> {
  if (!mermaidPromise) {
    mermaidPromise = import("mermaid").then((mod) => {
      const mermaid = mod.default;
      mermaid.initialize({
        startOnLoad: false,
        theme: "dark",
        darkMode: true,
        // Render errors are surfaced by <MermaidDiagram> itself — don't let
        // mermaid inject its own error bomb into the document.
        suppressErrorRendering: true,
        themeVariables: {
          background: "#16171b",
          primaryColor: "#26282e",
          primaryTextColor: "#e2e3e5",
          primaryBorderColor: "#5e6ad2",
          secondaryColor: "#1c1d22",
          tertiaryColor: "#0e0f11",
          lineColor: "#8a8f98",
          textColor: "#e2e3e5",
          mainBkg: "#26282e",
          nodeBorder: "#5e6ad2",
          clusterBkg: "#16171b",
          clusterBorder: "#26282e",
          edgeLabelBackground: "#16171b",
          fontFamily:
            '-apple-system, BlinkMacSystemFont, "Inter", "Segoe UI", sans-serif',
        },
      });
      return mermaid;
    });
  }
  return mermaidPromise;
}

// Unique render-target ids — mermaid requires one per render call.
let renderSeq = 0;

export function MermaidDiagram({ source }: { source: string }) {
  const ref = useRef<HTMLDivElement>(null);
  const [error, setError] = useState<string | null>(null);
  const [hasSvg, setHasSvg] = useState(false);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const mermaid = await loadMermaid();
        const { svg } = await mermaid.render(`td-mermaid-${++renderSeq}`, source);
        if (cancelled || !ref.current) return;
        ref.current.innerHTML = svg;
        setHasSvg(true);
        setError(null);
      } catch (e) {
        // Keep the last good diagram (if any) and show a quiet error line —
        // invalid intermediate states while typing must never crash anything.
        if (!cancelled) setError((e as Error)?.message ?? String(e));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [source]);

  return (
    <div className="mermaid-block">
      <div className="mermaid-diagram" ref={ref} style={hasSvg ? undefined : { display: "none" }} />
      {error && <div className="mermaid-error">mermaid: {error.split("\n")[0]}</div>}
    </div>
  );
}

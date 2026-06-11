/**
 * Tiny fuzzy scorer — subsequence match with word-boundary and
 * consecutive-run bonuses. No dependencies.
 *
 * Returns a score (higher = better) or null when the query is not a
 * subsequence of the text. Matching is greedy left-to-right, which is
 * not globally optimal but plenty for command-palette-sized lists.
 */
export function fuzzyScore(query: string, text: string): number | null {
  const q = query.toLowerCase();
  const t = text.toLowerCase();
  if (!q) return 0;
  let score = 0;
  let prev = -2; // last matched index; -2 so index 0 never counts as a run
  let from = 0;
  for (const ch of q) {
    const i = t.indexOf(ch, from);
    if (i === -1) return null;
    score += 1;
    if (i === 0 || /[^a-z0-9]/.test(t[i - 1])) score += 8; // word-boundary hit
    if (i === prev + 1) score += 5; // consecutive run
    score -= Math.min(i - from, 3); // capped gap penalty
    prev = i;
    from = i + 1;
  }
  return score;
}

/** Best score across a visible label and optional hidden alias keywords. */
export function fuzzyMatch(query: string, label: string, keywords?: string): number | null {
  const a = fuzzyScore(query, label);
  const b = keywords ? fuzzyScore(query, keywords) : null;
  if (a === null) return b;
  return b === null ? a : Math.max(a, b);
}

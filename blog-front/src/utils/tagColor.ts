// Shared by the article sidebar chips, the knowledge word cloud and its starfield,
// so a tag reads as the same colour everywhere on the blog.

// Ordered by hue: red → orange → … → pink → rose.
export const TAG_PALETTE = [
  "#dc2626", "#ea580c", "#d97706", "#65a30d", "#16a34a", "#0d9488", "#0891b2",
  "#0284c7", "#2563eb", "#4f46e5", "#7c3aed", "#c026d3", "#db2777", "#e11d48",
]

export const tagNameHash = (name: string): number => {
  let hash = 0
  for (const char of name) hash = (hash * 31 + char.codePointAt(0)!) >>> 0
  return hash
}

// A tag's own colour, before any per-article clash avoidance.
export const tagBaseColor = (name: string): string => TAG_PALETTE[tagNameHash(name) % TAG_PALETTE.length]

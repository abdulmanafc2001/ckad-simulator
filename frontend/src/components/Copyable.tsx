import { useEffect, useRef, useState } from 'react'

/**
 * Click-to-copy inline token used inside task/hint text. Renders like code,
 * copies its value to the clipboard on click and flashes a brief "Copied"
 * confirmation.
 */
export function CopyChip({ value, display }: { value: string; display?: string }) {
  const [copied, setCopied] = useState(false)
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => () => {
    if (timer.current) clearTimeout(timer.current)
  }, [])

  async function copy() {
    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(value)
      } else {
        // Fallback for non-secure contexts.
        const ta = document.createElement('textarea')
        ta.value = value
        ta.style.position = 'fixed'
        ta.style.opacity = '0'
        document.body.appendChild(ta)
        ta.select()
        document.execCommand('copy')
        document.body.removeChild(ta)
      }
      setCopied(true)
      if (timer.current) clearTimeout(timer.current)
      timer.current = setTimeout(() => setCopied(false), 1200)
    } catch {
      // Clipboard unavailable — do nothing.
    }
  }

  return (
    <button
      type="button"
      className={`copy-chip ${copied ? 'copied' : ''}`}
      onClick={copy}
      title={`Click to copy: ${value}`}
    >
      {display ?? value}
      <span className="copy-chip-feedback" aria-live="polite">
        {copied ? '✓' : '⧉'}
      </span>
    </button>
  )
}

/** One rendered segment of parsed text: plain prose or a copyable token. */
interface Segment {
  text: string
  /** Clipboard value when this segment is a copyable token. */
  copy?: string
}

// Matches, in priority order:
//  g1: single-quoted values            'web', 'nginx:1.25'
//  g2+g3: "namespace <name>"           namespace ckad-init
//  g4+g5: "named <name>"               ConfigMap named app-config
//  g6: absolute paths                  /usr/share/nginx/html
//  g7: key=value pairs                 tier=frontend, requests.cpu=500m
//  g8: image/ref-like name:tag         nginx:1.25, busybox:1.36
const TOKEN_RE =
  /('(?:[^'\\]|\\.)*')|(\bnamespace\s+)([\w-]+)|(\bnamed\s+)([\w-]+)|(\/[\w][\w\-./]*)|(\b[\w.-]+=[^\s,;)"']+)|(\b[\w][\w.-]*:[\w][\w.-]*)/g

/**
 * Common English words that follow "namespace"/"named" in prose but are not
 * resource names (e.g. "limiting the namespace to 2 Pods").
 */
const NAME_STOPWORDS = new Set([
  'a', 'an', 'the', 'and', 'or', 'to', 'of', 'in', 'on', 'at', 'is', 'are',
  'was', 'were', 'with', 'that', 'this', 'it', 'its', 'has', 'have', 'so',
])

/** Split plain text into prose segments and copyable tokens. */
function tokenize(text: string): Segment[] {
  const segments: Segment[] = []
  let last = 0
  for (const m of text.matchAll(TOKEN_RE)) {
    const idx = m.index ?? 0
    if (idx > last) segments.push({ text: text.slice(last, idx) })

    const [full] = m
    let chipStart = idx
    let chipEnd = idx + full.length
    let value = full

    if (m[1]) {
      // Quoted value: keep the quotes visible but copy the clean inner text.
      value = m[1].slice(1, -1)
    } else if (m[2]) {
      // "namespace <name>": only the name becomes a chip.
      chipEnd = idx + m[2].length + m[3].length
      chipStart = idx + m[2].length
      value = m[3]
    } else if (m[4]) {
      // "named <name>": only the name becomes a chip.
      chipEnd = idx + m[4].length + m[5].length
      chipStart = idx + m[4].length
      value = m[5]
    } else if (m[6]) {
      value = m[6]
    } else if (m[7]) {
      value = m[7]
    } else if (m[8]) {
      value = m[8]
    }

    // Skip stopword "names" ("namespace to 2 Pods") — render as plain text.
    if ((m[2] || m[4]) && NAME_STOPWORDS.has(value)) {
      segments.push({ text: text.slice(idx, idx + full.length) })
      last = idx + full.length
      continue
    }

    // Keep sentence punctuation out of path/kv/image chips ("nginx:1.25."
    // at the end of a sentence should copy as "nginx:1.25").
    while (
      m[1] === undefined &&
      chipEnd > chipStart &&
      /[.,;:!?]/.test(text[chipEnd - 1])
    ) {
      chipEnd--
      value = value.slice(0, -1)
    }

    if (chipStart > last) {
      // Re-split: prose before an offset chip (namespace/named prefix).
      segments.pop()
      segments.push({ text: text.slice(last, chipStart) })
    }
    segments.push({ text: text.slice(chipStart, chipEnd), copy: value })
    last = chipEnd
  }
  if (last < text.length) segments.push({ text: text.slice(last) })
  return segments
}

/**
 * Renders plain text where recognizable values (names, namespaces, paths,
 * key=value pairs, image references) become click-to-copy chips.
 */
export function CopyableText({ text }: { text: string }) {
  return (
    <>
      {tokenize(text).map((seg, i) =>
        seg.copy ? (
          <CopyChip key={i} value={seg.copy} display={seg.text} />
        ) : (
          <span key={i}>{seg.text}</span>
        ),
      )}
    </>
  )
}

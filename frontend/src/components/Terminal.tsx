import { useEffect, useRef } from 'react'
import { Terminal as XTerm } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import { api } from '../api/client'
import { TermEditor, tokenizeKeys, type EditorKind } from './editor'

const PROMPT = 'candidate@ckad-simulator:~$ '
// Same visible text as PROMPT, but with the `user@host:` portion in green.
// ANSI escapes are zero-width, so PROMPT.length stays valid for cursor math.
const PROMPT_COLORED = '\x1b[32mcandidate@ckad-simulator:\x1b[0m~$ '

/** Editor commands handled by the built-in in-terminal emulation. */
const EDITOR_BINS = new Set(['vi', 'vim', 'view', 'nano'])

function editorKind(bin: string): EditorKind {
  return bin === 'nano' ? 'nano' : 'vi'
}

interface TerminalProps {
  /** Shown once above the prompt when the terminal mounts. */
  banner?: string
}

/**
 * Interactive exam terminal (killer.sh style). Commands are sent to the
 * backend which executes them against the live minikube cluster and returns
 * the combined output.
 */
export function ExamTerminal({ banner }: TerminalProps) {
  const hostRef = useRef<HTMLDivElement>(null)
  const termRef = useRef<XTerm | null>(null)
  const lineRef = useRef('')
  /** Cursor position (character index) within the input line. */
  const cursorRef = useRef(0)
  const historyRef = useRef<string[]>([])
  const historyIdx = useRef(-1)
  const busyRef = useRef(false)
  const editorRef = useRef<TermEditor | null>(null)
  /** True while inside a bracketed paste block at the shell prompt. */
  const shellPastingRef = useRef(false)

  useEffect(() => {
    if (!hostRef.current) return

    const term = new XTerm({
      cursorBlink: true,
      fontSize: 13,
      fontFamily: '"JetBrains Mono", "Fira Code", Menlo, monospace',
      theme: {
        background: '#0d1117',
        foreground: '#c9d1d9',
        cursor: '#58a6ff',
        selectionBackground: '#264f78',
        black: '#0d1117',
        green: '#3fb950',
        red: '#f85149',
        yellow: '#d29922',
        blue: '#58a6ff',
      },
    })
    const fit = new FitAddon()
    term.loadAddon(fit)
    term.open(hostRef.current)
    fit.fit()
    termRef.current = term

    // Intercept Ctrl+Shift+V (paste) and Ctrl+Shift+C (copy). These are
    // browser-reserved shortcuts (paste-as-plain-text / DevTools inspect) that
    // the browser handles itself, so xterm never receives them. We handle them
    // manually and stop the browser from acting on them.
    const onKeyDown = (e: KeyboardEvent) => {
      if (!e.ctrlKey || !e.shiftKey) return
      const k = e.key.toLowerCase()
      if (k === 'v') {
        e.preventDefault()
        navigator.clipboard
          ?.readText()
          .then((text) => {
            if (text) term.paste(text)
          })
          .catch(() => {})
      } else if (k === 'c') {
        const sel = term.getSelection()
        if (sel) {
          e.preventDefault()
          navigator.clipboard?.writeText(sel).catch(() => {})
        }
      }
    }
    hostRef.current.addEventListener('keydown', onKeyDown)

    // Bracketed paste: pasted text arrives wrapped in \x1b[200~ … \x1b[201~
    // so multi-line pastes are handled deliberately instead of being
    // interpreted key-by-key (which mangled YAML in vi).
    term.write('\x1b[?2004h')

    const writePrompt = () => term.write('\r\n' + PROMPT_COLORED)

    if (banner) {
      term.writeln(banner)
    }
    term.write(PROMPT_COLORED)

    /** Moves the screen cursor to logical index `idx` of the prompt+input
     * region and updates cursorRef. Computes the delta from the CURRENT
     * position first — callers must NOT mutate cursorRef beforehand, or the
     * delta collapses to zero and the cursor never moves on screen. */
    const moveCursorTo = (idx: number) => {
      const cols = term.cols
      const from = PROMPT.length + cursorRef.current
      const to = PROMPT.length + Math.max(0, Math.min(idx, lineRef.current.length))
      const fromRow = Math.floor(from / cols)
      const toRow = Math.floor(to / cols)
      let out = ''
      if (toRow > fromRow) out += `\x1b[${toRow - fromRow}B`
      else if (toRow < fromRow) out += `\x1b[${fromRow - toRow}A`
      const c = to % cols
      out += '\r' + (c > 0 ? `\x1b[${c}C` : '')
      term.write(out)
      cursorRef.current = to - PROMPT.length
    }

    /** Redraws prompt + input line, clearing any stale characters on the
     * current row AND rows below (wrapped lines), then puts the cursor back
     * at cursorRef. The cursor first returns to the START of the input
     * region — otherwise only the cursor's own row gets cleared and long
     * (soft-wrapped) commands would duplicate on every keystroke. */
    const redrawInput = () => {
      const cols = term.cols
      const total = PROMPT.length + lineRef.current.length
      const cur = PROMPT.length + cursorRef.current
      let out = ''
      const upToStart = Math.floor(cur / cols)
      if (upToStart > 0) out += `\x1b[${upToStart}A`
      out += '\r\x1b[J' + PROMPT_COLORED + lineRef.current
      const up = Math.floor((total - cur) / cols)
      if (up > 0) out += `\x1b[${up}A`
      const c = cur % cols
      out += '\r' + (c > 0 ? `\x1b[${c}C` : '')
      term.write(out)
    }

    /** Inserts text at the cursor position (not just at end of line). */
    const insertText = (t: string) => {
      const p = cursorRef.current
      lineRef.current = lineRef.current.slice(0, p) + t + lineRef.current.slice(p)
      cursorRef.current = p + t.length
      redrawInput()
    }

    /** Full clear: erases screen AND scrollback including the current line,
     * then re-prints a single prompt with the input restored.
     * (term.clear() would keep the current line, duplicating the prompt.) */
    const clearScreen = () => {
      const cols = term.cols
      const cur = PROMPT.length + cursorRef.current
      const total = PROMPT.length + lineRef.current.length
      let out = '\x1b[2J\x1b[3J\x1b[H' + PROMPT_COLORED + lineRef.current
      const up = Math.floor((total - cur) / cols)
      if (up > 0) out += `\x1b[${up}A`
      const c = cur % cols
      out += '\r' + (c > 0 ? `\x1b[${c}C` : '')
      term.write(out)
    }

    /** Shell-style Tab completion: commands for the first word, sandbox
     * paths for arguments. Single match completes; multiple matches extend
     * the common prefix and list the options. */
    const completeInput = async () => {
      const line = lineRef.current
      try {
        const res = await api.complete(line)
        const matches = res.matches
        if (matches.length === 0) return
        const toks = line.split(/\s+/)
        const lastTok = toks[toks.length - 1] ?? ''

        if (matches.length === 1) {
          const m = matches[0]
          const addition = m.startsWith(lastTok) ? m.slice(lastTok.length) : m
          lineRef.current += addition + (m.endsWith('/') ? '' : ' ')
          term.write(addition + (m.endsWith('/') ? '' : ' '))
          cursorRef.current = lineRef.current.length
          return
        }

        // Extend to the longest common prefix, then show the options.
        let cp = matches[0]
        for (const m of matches) {
          while (!m.startsWith(cp)) cp = cp.slice(0, -1)
        }
        if (cp.length > lastTok.length && cp.startsWith(lastTok)) {
          lineRef.current += cp.slice(lastTok.length)
          term.write(cp.slice(lastTok.length))
        }
        term.write('\r\n\x1b[90m' + matches.join('   ') + '\x1b[0m')
        term.write('\r\n' + PROMPT_COLORED + lineRef.current)
        cursorRef.current = lineRef.current.length
      } catch {
        // Completion is best-effort; ignore failures.
      }
    }

    const runCommand = async (cmd: string) => {
      // Built-in vi/nano emulation: load the file from the exam sandbox and
      // hand the terminal screen over to the editor until it exits.
      const bin = cmd.split(/\s+/)[0]
      if (EDITOR_BINS.has(bin)) {
        const rest = cmd.slice(bin.length).trim()
        const file =
          rest
            .split(/\s+/)
            .filter((a) => a.length > 0 && !a.startsWith('-'))[0] ?? 'untitled.yaml'
        busyRef.current = true
        term.write('\r\n')
        let content = ''
        try {
          content = (await api.readFile(file)).content
        } catch {
          // New file — start empty.
        }
        term.writeln(`\x1b[90mopening built-in ${bin} (emulated) for ${file} — :wq saves, Ctrl+X exits nano\x1b[0m`)
        editorRef.current = new TermEditor(term, {
          file,
          content,
          kind: editorKind(bin),
          onSave: async (c) => {
            await api.writeFile(file, c)
          },
          onClose: () => {
            editorRef.current = null
            busyRef.current = false
            writePrompt()
          },
        })
        editorRef.current.start()
        return
      }

      busyRef.current = true
      try {
        const res = await api.exec({ command: cmd })
        if (res.output) {
          // Normalize newlines for xterm (\n -> \r\n).
          term.write('\r\n' + res.output.replace(/\n/g, '\r\n'))
        }
        if (res.exitCode !== 0) {
          term.write(`\r\n\x1b[90mexit ${res.exitCode}\x1b[0m`)
        }
      } catch (err) {
        const msg = err instanceof Error ? err.message : 'terminal error'
        term.write(`\r\n\x1b[31m${msg}\x1b[0m`)
      } finally {
        writePrompt()
        busyRef.current = false
      }
    }

    const disposers = [
      term.onData(async (data) => {
        // While an editor session is active, all keys go to the editor.
        if (editorRef.current) {
          editorRef.current.handle(data)
          return
        }
        if (busyRef.current) {
          // Still track bracketed-paste markers so a paste spanning a busy
          // window doesn't leave the pasting flag stuck on.
          for (const key of tokenizeKeys(data)) {
            if (key === '\x1b[200~') shellPastingRef.current = true
            else if (key === '\x1b[201~') shellPastingRef.current = false
          }
          return
        }

        let pasting = shellPastingRef.current
        for (const key of tokenizeKeys(data.replace(/\r\n/g, '\r'))) {
          if (key === '\x1b[200~') {
            pasting = true
            shellPastingRef.current = true
            continue
          }
          if (key === '\x1b[201~') {
            pasting = false
            shellPastingRef.current = false
            continue
          }

          if (pasting) {
            // Literal insert while a bracketed paste is in flight: newlines
            // collapse to a single space so a multi-line paste never
            // half-executes, and Enter is deferred until after the paste.
            if (key === '\r' || key === '\n') {
              const l = lineRef.current
              if (l.length > 0 && !/\s$/.test(l)) insertText(' ')
            } else if (key >= ' ') {
              insertText(key)
            }
            continue
          }

          switch (key) {
            case '\r':
            case '\n':
              {
                const cmd = lineRef.current.trim()
                // Snap the screen cursor to the end of the echoed line so
                // output starts cleanly below it even if the cursor sat
                // mid-line or on a wrapped row.
                moveCursorTo(lineRef.current.length)
                lineRef.current = ''
                cursorRef.current = 0
                historyIdx.current = -1
                if (cmd === 'clear') {
                  clearScreen()
                  break
                }
                if (cmd.length > 0) {
                  historyRef.current.unshift(cmd)
                  if (historyRef.current.length > 100) historyRef.current.pop()
                  await runCommand(cmd)
                } else {
                  writePrompt()
                }
              }
              break
            case '\x7f': // Backspace — crosses wrapped rows correctly
              if (cursorRef.current > 0) {
                const p = cursorRef.current
                lineRef.current = lineRef.current.slice(0, p - 1) + lineRef.current.slice(p)
                cursorRef.current = p - 1
                redrawInput()
              }
              break
            case '\x1b[3~': // Delete — removes the char at the cursor
              if (cursorRef.current < lineRef.current.length) {
                const p = cursorRef.current
                lineRef.current = lineRef.current.slice(0, p) + lineRef.current.slice(p + 1)
                redrawInput()
              }
              break
            case '\x1b[D': // Left
              moveCursorTo(cursorRef.current - 1)
              break
            case '\x1b[C': // Right
              moveCursorTo(cursorRef.current + 1)
              break
            case '\x1b[H':
            case '\x1b[1~':
            case '\x1bOH':
            case '\x01': // Ctrl+A — home
              moveCursorTo(0)
              break
            case '\x1b[F':
            case '\x1b[4~':
            case '\x1bOF':
            case '\x05': // Ctrl+E — end
              moveCursorTo(lineRef.current.length)
              break
            case '\x17': // Ctrl+W — delete word before cursor
              {
                const p = cursorRef.current
                const kept = lineRef.current.slice(0, p).replace(/\S+\s*$/, '')
                lineRef.current = kept + lineRef.current.slice(p)
                cursorRef.current = kept.length
                redrawInput()
              }
              break
            case '\x0b': // Ctrl+K — kill from cursor to end of line
              lineRef.current = lineRef.current.slice(0, cursorRef.current)
              redrawInput()
              break
            case '\x15': // Ctrl+U — clear the whole line
              lineRef.current = ''
              cursorRef.current = 0
              redrawInput()
              break
            case '\u0003': // Ctrl+C
              moveCursorTo(lineRef.current.length)
              term.write('^C')
              lineRef.current = ''
              cursorRef.current = 0
              writePrompt()
              break
            case '\u000c': // Ctrl+L
              clearScreen()
              break
            case '\t': // Tab — shell-style completion
              cursorRef.current = lineRef.current.length
              void completeInput()
              break
            case '\x1b[A': // Arrow up
              if (historyIdx.current < historyRef.current.length - 1) {
                historyIdx.current++
                lineRef.current = historyRef.current[historyIdx.current]
                cursorRef.current = lineRef.current.length
                redrawInput()
              }
              break
            case '\x1b[B': // Arrow down
              if (historyIdx.current > 0) {
                historyIdx.current--
                lineRef.current = historyRef.current[historyIdx.current]
                cursorRef.current = lineRef.current.length
                redrawInput()
              } else if (historyIdx.current === 0) {
                historyIdx.current = -1
                lineRef.current = ''
                cursorRef.current = 0
                redrawInput()
              }
              break
            default:
              if (key >= ' ') insertText(key)
          }
        }
      }),
    ]

    const onResize = () => {
      fit.fit()
      redrawInput()
    }
    window.addEventListener('resize', onResize)

    return () => {
      window.removeEventListener('resize', onResize)
      hostRef.current?.removeEventListener('keydown', onKeyDown)
      disposers.forEach((d) => d.dispose())
      term.dispose()
      termRef.current = null
    }
  }, [banner])

  return <div ref={hostRef} className="exam-terminal" aria-label="Exam terminal" />
}

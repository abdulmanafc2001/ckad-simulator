import { useEffect, useRef } from 'react'
import { Terminal as XTerm } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import { api } from '../api/client'
import { TermEditor, type EditorKind } from './editor'

const PROMPT = 'candidate@ckad-simulator:~$ '

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
  const historyRef = useRef<string[]>([])
  const historyIdx = useRef(-1)
  const busyRef = useRef(false)
  const editorRef = useRef<TermEditor | null>(null)

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

    const writePrompt = () => term.write('\r\n' + PROMPT)

    if (banner) {
      term.writeln(banner)
    }
    term.write(PROMPT)

    const redrawLine = () => {
      // Clear the current line then re-print prompt + buffered text.
      term.write('\r\x1b[K' + PROMPT + lineRef.current)
    }

    /** Full clear: erases screen AND scrollback including the current line,
     * then re-prints a single prompt. (term.clear() would keep the current
     * line, duplicating the prompt.) */
    const clearScreen = () => {
      term.write('\x1b[2J\x1b[3J\x1b[H' + PROMPT + lineRef.current)
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
        term.write('\r\n' + PROMPT + lineRef.current)
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
        if (busyRef.current) return
        // Treat CR, LF, and CRLF all as Enter (paste/tools may send LF).
        if (data === '\r\n') data = '\r'
        switch (data) {
          case '\r':
          case '\n':
            {
              const cmd = lineRef.current.trim()
              lineRef.current = ''
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
          case '\u007f': // Backspace
            if (lineRef.current.length > 0) {
              lineRef.current = lineRef.current.slice(0, -1)
              term.write('\b \b')
            }
            break
          case '\u0003': // Ctrl+C
            term.write('^C')
            lineRef.current = ''
            writePrompt()
            break
          case '\u000c': // Ctrl+L
            clearScreen()
            break
          case '\t': // Tab — shell-style completion
            void completeInput()
            break
          case '\x1b[A': // Arrow up
            if (historyIdx.current < historyRef.current.length - 1) {
              historyIdx.current++
              lineRef.current = historyRef.current[historyIdx.current]
              redrawLine()
            }
            break
          case '\x1b[B': // Arrow down
            if (historyIdx.current > 0) {
              historyIdx.current--
              lineRef.current = historyRef.current[historyIdx.current]
              redrawLine()
            } else if (historyIdx.current === 0) {
              historyIdx.current = -1
              lineRef.current = ''
              redrawLine()
            }
            break
          default:
            if (data >= ' ' || data === '\t') {
              lineRef.current += data
              term.write(data)
            }
        }
      }),
    ]

    const onResize = () => fit.fit()
    window.addEventListener('resize', onResize)

    return () => {
      window.removeEventListener('resize', onResize)
      disposers.forEach((d) => d.dispose())
      term.dispose()
      termRef.current = null
    }
  }, [banner])

  return <div ref={hostRef} className="exam-terminal" aria-label="Exam terminal" />
}

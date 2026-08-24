import type { Terminal } from '@xterm/xterm'

/**
 * In-terminal vi / nano emulation for the exam terminal.
 *
 * The backend exec endpoint is request/response (no PTY), so real editors
 * cannot run. Instead the browser terminal takes over the screen and
 * provides a built-in editor with the familiar keybindings:
 *
 *   vi   — modal editing: h/j/k/l, i/a/o/A, x, dd, u, 0/$/G/gg, w/b,
 *          `:w` `:wq` `:q!` `:x`
 *   nano — direct typing with ^O write out, ^X exit, ^K cut line,
 *          ^U uncut, arrow/Home/End navigation
 *
 * Files are loaded from and saved to the exam sandbox on the backend via
 * api.readFile/api.writeFile, so `kubectl apply -f <file>` works afterwards.
 */

export type EditorKind = 'vi' | 'nano'

export interface EditorOptions {
  file: string
  content: string
  kind: EditorKind
  onSave: (content: string) => Promise<void>
  onClose: () => void
}

const CSI = '\x1b['

export class TermEditor {
  private lines: string[]
  private row = 0
  private col = 0
  private scrollTop = 0
  private dirty = false
  /** vi normal mode when false; nano is always in "insert" mode. */
  private insert: boolean
  private pendingD = false // vi dd
  private pendingG = false // vi gg
  private colon = false // vi : command line
  private cmdline = ''
  private status = ''
  private savePrompt = false // nano ^X with unsaved changes
  private undoStack: string[][] = []
  private killRing: string[] = []
  private exiting = false
  private term: Terminal
  private o: EditorOptions

  constructor(term: Terminal, o: EditorOptions) {
    this.term = term
    this.o = o
    this.lines = o.content.length > 0 ? o.content.replace(/\n$/, '').split('\n') : ['']
    this.insert = o.kind === 'nano'
  }

  /** Takes over the terminal screen. */
  start() {
    this.render()
  }

  /** Shows a transient status message (e.g. save results). */
  notify(msg: string) {
    this.status = msg
    if (!this.exiting) this.render()
  }

  handle(data: string) {
    if (this.exiting) return
    if (this.savePrompt) {
      this.handleSavePrompt(data)
      return
    }
    if (this.o.kind === 'nano') {
      this.handleNano(data)
    } else {
      this.handleVi(data)
    }
    this.clampCursor()
    this.render()
  }

  // ------------------------------------------------------------------ vi

  private handleVi(data: string) {
    if (this.colon) {
      switch (data) {
        case '\r':
        case '\n': {
          const cmd = this.cmdline.trim()
          this.colon = false
          this.cmdline = ''
          this.execEx(cmd)
          break
        }
        case '\x7f':
          this.cmdline = this.cmdline.slice(0, -1)
          break
        case '\x1b':
          this.colon = false
          this.cmdline = ''
          break
        default:
          if (data >= ' ') this.cmdline += data
      }
      return
    }

    if (this.insert) {
      // Escape returns to normal mode before any other handling.
      if (data === '\x1b') {
        this.insert = false
        this.status = ''
        return
      }
      this.handleEditKey(data)
      return
    }

    // Normal mode.
    if (this.pendingD) {
      this.pendingD = false
      if (data === 'd') {
        this.pushUndo()
        if (this.lines.length > 1) {
          this.lines.splice(this.row, 1)
          if (this.row >= this.lines.length) this.row = this.lines.length - 1
        } else {
          this.lines[0] = ''
        }
        this.col = 0
        this.dirty = true
      }
      return
    }
    if (this.pendingG) {
      this.pendingG = false
      if (data === 'g') this.row = 0
      return
    }

    switch (data) {
      case 'h':
      case '\x1b[D':
        this.moveCol(-1)
        break
      case 'l':
      case '\x1b[C':
        this.moveCol(1)
        break
      case 'j':
      case '\x1b[B':
        this.moveRow(1)
        break
      case 'k':
      case '\x1b[A':
        this.moveRow(-1)
        break
      case '0':
      case '\x1b[H':
      case '\x1b[1~':
        this.col = 0
        break
      case '$':
      case '\x1b[F':
      case '\x1b[4~':
        this.col = Math.max(0, this.line().length - (this.insert ? 0 : 1))
        break
      case 'G':
        this.row = this.lines.length - 1
        break
      case 'g':
        this.pendingG = true
        break
      case 'w':
        this.nextWord()
        break
      case 'b':
        this.prevWord()
        break
      case 'x':
        this.pushUndo()
        if (this.line().length > 0) {
          const l = this.line()
          this.lines[this.row] = l.slice(0, this.col) + l.slice(this.col + 1)
          if (this.col >= this.lines[this.row].length) this.col = Math.max(0, this.lines[this.row].length - 1)
          this.dirty = true
        }
        break
      case 'i':
        this.insert = true
        break
      case 'a':
        this.insert = true
        this.moveCol(1)
        break
      case 'A':
        this.insert = true
        this.col = this.line().length
        break
      case 'I':
        this.insert = true
        this.col = 0
        break
      case 'o':
        this.pushUndo()
        this.lines.splice(this.row + 1, 0, '')
        this.row++
        this.col = 0
        this.insert = true
        this.dirty = true
        break
      case 'O':
        this.pushUndo()
        this.lines.splice(this.row, 0, '')
        this.col = 0
        this.insert = true
        this.dirty = true
        break
      case 'd':
        this.pendingD = true
        break
      case 'u':
        this.undo()
        break
      case ':':
        this.colon = true
        this.cmdline = ''
        break
      default:
        // Ignore other keys in normal mode.
    }
  }

  private execEx(cmd: string) {
    switch (cmd) {
      case '':
      case 'w':
        void this.save()
        break
      case 'wq':
      case 'x':
        void this.save().then(() => this.close())
        break
      case 'q':
        if (this.dirty) {
          this.status = 'No write since last change (:q! overrides)'
        } else {
          this.close()
        }
        break
      case 'q!':
        this.close()
        break
      default:
        this.status = `Not an editor command: ${cmd}`
    }
  }

  // ---------------------------------------------------------------- nano

  private handleNano(data: string) {
    switch (data) {
      case '\x0f': // Ctrl+O — write out
        void this.save()
        break
      case '\x18': // Ctrl+X — exit
        if (this.dirty) {
          this.savePrompt = true
          this.status = ''
        } else {
          this.close()
        }
        break
      case '\x0b': // Ctrl+K — cut line
        this.killRing.push(this.line())
        if (this.killRing.length > 100) this.killRing.shift()
        if (this.lines.length > 1) {
          this.lines.splice(this.row, 1)
          if (this.row >= this.lines.length) this.row = this.lines.length - 1
        } else {
          this.lines[0] = ''
        }
        this.col = 0
        this.dirty = true
        break
      case '\x15': // Ctrl+U — uncut
        if (this.killRing.length > 0) {
          this.lines.splice(this.row, 0, this.killRing.pop() as string)
          this.dirty = true
        }
        break
      default:
        this.handleEditKey(data)
    }
  }

  private handleSavePrompt(data: string) {
    switch (data.toLowerCase()) {
      case 'y':
        this.savePrompt = false
        void this.save().then(() => this.close())
        break
      case 'n':
        this.savePrompt = false
        this.close()
        break
      case '\x03': // Ctrl+C cancels
        this.savePrompt = false
        this.status = ''
        break
      default:
      // wait for y/n/^C
    }
  }

  // ------------------------------------------------------- shared editing

  private handleEditKey(data: string) {
    switch (true) {
      case data === '\r' || data === '\n': {
        const l = this.line()
        this.lines.splice(this.row, 1, l.slice(0, this.col), l.slice(this.col))
        this.row++
        this.col = 0
        this.dirty = true
        break
      }
      case data === '\x7f': // Backspace
        if (this.col > 0) {
          const l = this.line()
          this.lines[this.row] = l.slice(0, this.col - 1) + l.slice(this.col)
          this.col--
          this.dirty = true
        } else if (this.row > 0) {
          const prev = this.lines[this.row - 1]
          const cur = this.line()
          this.lines.splice(this.row - 1, 2, prev + cur)
          this.row--
          this.col = prev.length
          this.dirty = true
        }
        break
      case data === '\x1b[3~': // Delete
        if (this.col < this.line().length) {
          const l = this.line()
          this.lines[this.row] = l.slice(0, this.col) + l.slice(this.col + 1)
          this.dirty = true
        }
        break
      case data === '\x1b[A':
        this.moveRow(-1)
        break
      case data === '\x1b[B':
        this.moveRow(1)
        break
      case data === '\x1b[C':
        this.moveCol(1)
        break
      case data === '\x1b[D':
        this.moveCol(-1)
        break
      case data === '\x1b[H' || data === '\x1bOH' || data === '\x1b[1~':
        this.col = 0
        break
      case data === '\x1b[F' || data === '\x1bOF' || data === '\x1b[4~':
        this.col = this.line().length
        break
      case data === '\t':
        this.typeText('  ')
        break
      case data >= ' ':
        this.typeText(data)
        break
      default:
        // Ignore control keys we do not handle.
    }
  }

  private typeText(text: string) {
    const l = this.line()
    this.lines[this.row] = l.slice(0, this.col) + text + l.slice(this.col)
    this.col += text.length
    this.dirty = true
  }

  // ------------------------------------------------------------ movement

  private line(): string {
    return this.lines[this.row] ?? ''
  }

  private moveRow(d: number) {
    this.row = Math.min(Math.max(this.row + d, 0), this.lines.length - 1)
    if (this.col > this.line().length) this.col = this.line().length
  }

  private moveCol(d: number) {
    const max = this.insert ? this.line().length : Math.max(0, this.line().length - 1)
    this.col = Math.min(Math.max(this.col + d, 0), max)
  }

  private nextWord() {
    let l = this.line()
    let c = this.col
    while (c < l.length && /\S/.test(l[c])) c++
    while (c < l.length && /\s/.test(l[c])) c++
    if (c >= l.length && this.row < this.lines.length - 1) {
      this.row++
      this.col = 0
      return
    }
    this.col = c
  }

  private prevWord() {
    const l = this.line()
    let c = this.col
    while (c > 0 && /\s/.test(l[c - 1])) c--
    while (c > 0 && /\S/.test(l[c - 1])) c--
    this.col = c
  }

  private clampCursor() {
    if (this.row >= this.lines.length) this.row = this.lines.length - 1
    if (this.col > this.line().length) this.col = this.line().length
  }

  // ---------------------------------------------------------- undo / io

  private pushUndo() {
    this.undoStack.push([...this.lines])
    if (this.undoStack.length > 100) this.undoStack.shift()
  }

  private undo() {
    const prev = this.undoStack.pop()
    if (prev) {
      this.lines = prev
      this.clampCursor()
      this.dirty = true
      this.status = 'undone'
    }
  }

  private async save() {
    try {
      await this.o.onSave(this.lines.join('\n') + '\n')
      this.dirty = false
      this.status = `"${this.o.file}" written, ${this.lines.length} lines`
    } catch (err) {
      this.status = `write failed: ${err instanceof Error ? err.message : 'error'}`
    }
  }

  private close() {
    this.exiting = true
    this.term.write(`${CSI}2J${CSI}H`)
    this.o.onClose()
  }

  // ------------------------------------------------------------ rendering

  private render() {
    const { rows, cols } = this.term
    const chromeRows = this.o.kind === 'nano' ? 3 : 1
    const bodyRows = Math.max(1, rows - chromeRows)

    // Keep the cursor inside the visible window.
    if (this.row < this.scrollTop) this.scrollTop = this.row
    if (this.row >= this.scrollTop + bodyRows) this.scrollTop = this.row - bodyRows + 1

    let out = `${CSI}H${CSI}J`
    for (let r = 0; r < bodyRows; r++) {
      if (r > 0) out += '\r\n'
      out += `${CSI}K`
      const idx = this.scrollTop + r
      if (idx < this.lines.length) {
        out += clip(this.lines[idx], cols - 1)
      } else if (this.o.kind === 'vi') {
        out += '\x1b[38;5;240m~\x1b[0m'
      }
    }

    if (this.o.kind === 'nano') {
      const title = pad(` nano (emulated) — ${this.o.file}${this.dirty ? ' [modified]' : ''}`, cols - 1)
      out += `\r\n${CSI}K\x1b[7m${title}\x1b[0m`
      const msg = this.savePrompt
        ? ' Save modified buffer?  Y = yes, N = no, ^C = cancel '
        : this.status || ` line ${this.row + 1}/${this.lines.length}, col ${this.col + 1} `
      out += `\r\n${CSI}K${clip(msg, cols - 1)}`
      const bar = ' ^O Write Out   ^X Exit   ^K Cut Line   ^U Uncut   ^G Get Help'
      out += `\r\n${CSI}K\x1b[7m${pad(bar, cols - 1)}\x1b[0m`
    } else {
      let statusLine: string
      if (this.colon) {
        statusLine = `:${this.cmdline}`
      } else if (this.status) {
        statusLine = this.status
      } else {
        const mode = this.insert ? '-- INSERT -- ' : ''
        statusLine = `${mode}"${this.o.file}" ${this.lines.length}L${this.dirty ? ' [modified]' : ''}`
      }
      out += `\r\n${CSI}K${clip(statusLine, cols - 1)}`
    }

    // Place the cursor.
    const cursorScreenRow = Math.min(this.row - this.scrollTop, bodyRows - 1)
    out += `${CSI}${cursorScreenRow + 1};${Math.min(this.col + 1, cols)}H`

    this.term.write(out)
  }
}

function clip(s: string, n: number): string {
  return s.length <= n ? s : s.slice(0, n)
}

function pad(s: string, n: number): string {
  return s.length <= n ? s + ' '.repeat(n - s.length) : clip(s, n)
}

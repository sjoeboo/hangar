import { useEffect, useRef } from 'react'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import '@xterm/xterm/css/xterm.css'

interface TerminalViewProps {
  sessionId: string
  className?: string
}

function getStreamURL(sessionId: string): string {
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = import.meta.env.DEV ? 'localhost:47437' : window.location.host
  return `${proto}//${host}/api/v1/sessions/${sessionId}/stream`
}

export function TerminalView({ sessionId, className }: TerminalViewProps) {
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!containerRef.current) return

    const term = new Terminal({
      fontFamily: '"Symbols Nerd Font Mono", "Cascadia Code", "JetBrains Mono", "Fira Code", monospace',
      fontSize: 13,
      lineHeight: 1.2,
      theme: {
        background: '#020617',
        foreground: '#e2e8f0',
        cursor: '#94a3b8',
        black: '#0f172a',
        red: '#f87171',
        green: '#4ade80',
        yellow: '#facc15',
        blue: '#60a5fa',
        magenta: '#c084fc',
        cyan: '#22d3ee',
        white: '#e2e8f0',
        brightBlack: '#475569',
        brightRed: '#fca5a5',
        brightGreen: '#86efac',
        brightYellow: '#fde047',
        brightBlue: '#93c5fd',
        brightMagenta: '#d8b4fe',
        brightCyan: '#67e8f9',
        brightWhite: '#f8fafc',
      },
      cursorBlink: true,
      scrollback: 5000,
    })

    const fitAddon = new FitAddon()
    term.loadAddon(fitAddon)
    term.loadAddon(new WebLinksAddon())
    term.open(containerRef.current)

    // Fit after layout is settled, then open the stream WS
    requestAnimationFrame(() => {
      fitAddon.fit()
      openStream()
    })

    // ResizeObserver → fit + notify server of new dimensions
    const ro = new ResizeObserver(() => {
      fitAddon.fit()
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }))
      }
    })
    ro.observe(containerRef.current!)

    let ws: WebSocket | null = null
    const decoder = new TextDecoder()

    function openStream() {
      ws = new WebSocket(getStreamURL(sessionId))
      ws.binaryType = 'arraybuffer'

      ws.onopen = () => {
        // Send actual terminal size so tmux renders at the right width
        ws!.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }))
      }

      ws.onmessage = (event) => {
        if (event.data instanceof ArrayBuffer) {
          // Raw PTY bytes — decode as UTF-8 stream and write directly to xterm
          term.write(decoder.decode(new Uint8Array(event.data), { stream: true }))
        }
        // Text frames are control messages (errors, status) — ignore for now
      }

      ws.onclose = () => {
        term.write('\r\n\x1b[33m[stream closed]\x1b[0m\r\n')
      }

      ws.onerror = () => {
        term.write('\r\n\x1b[31m[stream error — session may not be running]\x1b[0m\r\n')
      }
    }

    // Forward keystrokes to server via the stream WS
    term.onData((data) => {
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'input', data }))
      }
    })

    return () => {
      ws?.close()
      ro.disconnect()
      term.dispose()
    }
  }, [sessionId])

  return (
    <div
      ref={containerRef}
      className={className}
      style={{ width: '100%', height: '100%' }}
    />
  )
}

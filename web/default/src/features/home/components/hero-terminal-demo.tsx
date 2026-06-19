/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'

// Gateway base URL + key shown in the demo. Change these to match your deployment.
const GATEWAY_URL = 'https://model-bay.com'
const API_KEY = 'modelbay'

type LineKind = 'command' | 'success' | 'pending'

interface TermLine {
  kind: LineKind
  text: string
}

interface CliDemo {
  id: string
  label: string
  /** Tailwind classes for the active tab. */
  activeTab: string
  lines: TermLine[]
}

function buildClis(t: (key: string) => string): CliDemo[] {
  return [
    {
      id: 'claude',
      label: 'Claude Code',
      activeTab: 'border-orange-400 text-orange-500 dark:text-orange-400',
      lines: [
        { kind: 'command', text: 'npm i -g @anthropic-ai/claude-code' },
        { kind: 'success', text: 'installed claude-code@latest' },
        { kind: 'command', text: `export ANTHROPIC_BASE_URL="${GATEWAY_URL}"` },
        { kind: 'command', text: `export ANTHROPIC_AUTH_TOKEN="${API_KEY}"` },
        {
          kind: 'command',
          text: `claude "${t('Refactor this component and add unit tests')}"`,
        },
        { kind: 'pending', text: 'planning edits · 1.2s' },
        { kind: 'success', text: '4 files changed · tests passing' },
      ],
    },
    {
      id: 'codex',
      label: 'Codex CLI',
      activeTab: 'border-emerald-400 text-emerald-600 dark:text-emerald-400',
      lines: [
        { kind: 'command', text: 'npm i -g @openai/codex' },
        { kind: 'success', text: 'installed codex@latest' },
        { kind: 'command', text: `export OPENAI_BASE_URL="${GATEWAY_URL}"` },
        { kind: 'command', text: `export OPENAI_API_KEY="${API_KEY}"` },
        {
          kind: 'command',
          text: `codex "${t('Fix the concurrency issue in the login flow')}"`,
        },
        { kind: 'pending', text: 'reasoning · 0.9s' },
        { kind: 'success', text: 'patch applied · 2 tests added' },
      ],
    },
    {
      id: 'gemini',
      label: 'Gemini CLI',
      activeTab: 'border-sky-400 text-sky-600 dark:text-sky-400',
      lines: [
        { kind: 'command', text: 'npm i -g @google/gemini-cli' },
        { kind: 'success', text: 'installed gemini-cli@latest' },
        {
          kind: 'command',
          text: `export GOOGLE_GEMINI_BASE_URL="${GATEWAY_URL}"`,
        },
        { kind: 'command', text: `export GEMINI_API_KEY="${API_KEY}"` },
        {
          kind: 'command',
          text: `gemini "${t('Explain the execution plan of this SQL')}"`,
        },
        { kind: 'pending', text: 'analysing query · 0.8s' },
        { kind: 'success', text: '3 hot paths identified · index suggested' },
      ],
    },
  ]
}

interface HeroTerminalDemoProps {
  className?: string
}

export function HeroTerminalDemo(props: HeroTerminalDemoProps) {
  const { t, i18n } = useTranslation()
  const clis = useMemo(() => buildClis(t), [t, i18n.language])

  const [tab, setTab] = useState(0)
  const [lineIdx, setLineIdx] = useState(0)
  const [charIdx, setCharIdx] = useState(0)
  const [done, setDone] = useState(false)
  const timerRef = useRef<ReturnType<typeof setTimeout>>(undefined)

  const cli = clis[tab]

  useEffect(() => {
    const reduced = window.matchMedia(
      '(prefers-reduced-motion: reduce)'
    ).matches
    const lines = clis[tab].lines
    let cancelled = false

    setLineIdx(0)
    setCharIdx(0)
    setDone(false)

    const schedule = (fn: () => void, ms: number) => {
      timerRef.current = setTimeout(() => {
        if (!cancelled) fn()
      }, ms)
    }

    if (reduced) {
      setLineIdx(lines.length)
      setDone(true)
      // Reduced motion: render the full session, no animation, no looping.
      return () => {
        cancelled = true
        if (timerRef.current) clearTimeout(timerRef.current)
      }
    }

    let li = 0
    let ci = 0

    const step = () => {
      const line = lines[li]
      if (!line) {
        // Finished — stop here (no auto-advance to the next CLI).
        setDone(true)
        setLineIdx(lines.length)
        return
      }
      if (line.kind === 'command' && ci < line.text.length) {
        ci += 1
        setLineIdx(li)
        setCharIdx(ci)
        schedule(step, 18 + Math.random() * 34)
        return
      }
      // Line fully revealed — hold briefly, then advance.
      setLineIdx(li)
      setCharIdx(line.text.length)
      const pause =
        line.kind === 'pending' ? 880 : line.kind === 'command' ? 360 : 280
      schedule(() => {
        li += 1
        ci = 0
        step()
      }, pause)
    }

    schedule(step, 380)

    return () => {
      cancelled = true
      if (timerRef.current) clearTimeout(timerRef.current)
    }
  }, [tab, clis])

  const lines = cli.lines
  const visible: ReactNode[] = []
  const lastIndex = Math.min(lineIdx, lines.length - 1)
  for (let i = 0; i <= lastIndex; i++) {
    const line = lines[i]
    const isCurrent = i === lineIdx && !done
    const isTypingCommand = isCurrent && line.kind === 'command'
    const text = isTypingCommand ? line.text.slice(0, charIdx) : line.text
    visible.push(
      <TerminalLine
        key={i}
        kind={line.kind}
        text={text}
        showCursor={isTypingCommand}
      />
    )
  }

  return (
    <div className={cn('mx-auto w-full max-w-2xl', props.className)}>
      <div
        className={cn(
          'overflow-hidden rounded-2xl border-2 backdrop-blur-sm',
          'border-border bg-white/95 shadow-[0_20px_50px_-25px_rgba(15,23,42,0.18)] ring-1 ring-black/5',
          'dark:border-white/15 dark:bg-[#0b0f17]/95 dark:shadow-[0_20px_60px_-25px_rgba(0,0,0,0.7)] dark:ring-white/5'
        )}
      >
        {/* Title bar with traffic lights */}
        <div
          className={cn(
            'flex items-center gap-2 border-b px-4 py-2.5',
            'border-border/50 dark:border-white/[0.05]'
          )}
        >
          <div className='flex items-center gap-1.5'>
            <span className='size-3 rounded-full bg-[#ff5f57]' />
            <span className='size-3 rounded-full bg-[#febc2e]' />
            <span className='size-3 rounded-full bg-[#28c840]' />
          </div>
          <span className='text-foreground/40 mx-auto font-mono text-[11px] tracking-wide'>
            {cli.label.toLowerCase().replace(/\s+/g, '-')} — zsh
          </span>
          <div className='flex items-center gap-2'>
            <span className='inline-block size-1.5 rounded-full bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.45)]' />
            <span className='text-foreground/40 font-mono text-[10px] tracking-wider uppercase'>
              live
            </span>
          </div>
        </div>

        {/* CLI tab strip */}
        <div
          className={cn(
            'flex items-center gap-1 border-b px-2 sm:gap-1.5 sm:px-3',
            'border-border/50 dark:border-white/[0.05]'
          )}
        >
          {clis.map((item, index) => {
            const isActive = index === tab
            return (
              <button
                key={item.id}
                type='button'
                onClick={() => setTab(index)}
                className={cn(
                  'relative -mb-px flex items-center gap-1.5 border-b-2 px-2.5 py-2.5 text-[11px] font-medium tracking-wide transition-colors sm:px-3 sm:text-xs',
                  isActive
                    ? item.activeTab
                    : 'text-foreground/40 hover:text-foreground/70 border-transparent'
                )}
              >
                {item.label}
              </button>
            )
          })}
        </div>

        {/* Terminal body — fixed height so it never reflows */}
        <div className='h-[360px] overflow-hidden px-5 py-4 font-mono text-[12.5px] leading-[1.7]'>
          <div className='flex flex-col'>
            {visible}
            {done && <PromptCursor />}
          </div>
        </div>

        {/* Footer */}
        <div
          className={cn(
            'flex items-center justify-between border-t px-5 py-2.5',
            'border-border/40 bg-muted/30 dark:border-white/[0.05] dark:bg-white/[0.02]'
          )}
        >
          <span className='text-foreground/40 font-mono text-[10px] tracking-wider'>
            {GATEWAY_URL.replace(/^https?:\/\//, '')}
          </span>
          <span className='text-foreground/30 font-mono text-[10px] tracking-wider uppercase'>
            one url · every cli
          </span>
        </div>
      </div>
    </div>
  )
}

function TerminalLine(props: {
  kind: LineKind
  text: string
  showCursor: boolean
}) {
  const { kind, text, showCursor } = props

  if (kind === 'command') {
    return (
      <div className='flex'>
        <span className='mr-2 shrink-0 select-none text-blue-500 dark:text-blue-400'>
          $
        </span>
        <span className='text-foreground/90 break-all'>
          {text}
          {showCursor && <Caret />}
        </span>
      </div>
    )
  }

  if (kind === 'success') {
    return (
      <div className='flex'>
        <span className='mr-2 shrink-0 select-none text-emerald-500 dark:text-emerald-400'>
          ✓
        </span>
        <span className='text-foreground/55 break-all'>{text}</span>
      </div>
    )
  }

  // pending
  return (
    <div className='flex'>
      <span className='mr-2 shrink-0 animate-pulse select-none text-amber-500 dark:text-amber-400'>
        ⌁
      </span>
      <span className='text-foreground/55 break-all'>{text}</span>
    </div>
  )
}

function PromptCursor() {
  return (
    <div className='flex'>
      <span className='mr-2 shrink-0 select-none text-blue-500 dark:text-blue-400'>
        $
      </span>
      <Caret />
    </div>
  )
}

function Caret() {
  return (
    <span className='ml-0.5 inline-block h-[1.05em] w-[0.5ch] translate-y-[0.18em] animate-pulse rounded-[1px] bg-foreground/70 align-middle' />
  )
}

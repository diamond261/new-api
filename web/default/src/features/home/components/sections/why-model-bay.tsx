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
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'

interface WhyModelBayProps {
  className?: string
}

interface FeatureCard {
  number: string
  ascii: string
  titleKey: string
  descriptionKey: string
}

const CARDS: FeatureCard[] = [
  {
    number: '01',
    ascii: `┌──┐  ┌──┐  ┌──┐
│HK│  │HK│  │HK│
└──┘  └──┘  └──┘
   ↘   ↓   ↙
   ┌──────┐
   │ flow │
   └──────┘`,
    titleKey: 'Hong Kong Local',
    descriptionKey: 'Operated by a Hong Kong-based company',
  },
  {
    number: '02',
    ascii: `┌─────────────┐
│  ░░░░░░░░░  │
│  ░ no logs  │
│  ░░░░░░░░░  │
└──────┬──────┘
       ▼
    /dev/null`,
    titleKey: 'Data Security',
    descriptionKey:
      'We commit that none of your data is stored on our site in any form',
  },
  {
    number: '03',
    ascii: `p50  ▰▰▱▱▱▱
p90  ▰▰▰▱▱▱
p99  ▰▰▰▰▱▱
     ─────────
     32ms  ✓`,
    titleKey: 'Fast Speed',
    descriptionKey:
      'Latency stays stable under 100ms, paired with Hong Kong-class speed',
  },
  {
    number: '04',
    ascii: `CN ──────╮
         │
   ╭─────┤
   │  ModelBay
   ╰─────╯│
         ▼▼
   OpenAI · Claude
   Gemini · Qwen ...`,
    titleKey: 'Rich Selection',
    descriptionKey: 'All the most popular models are available',
  },
]

export function WhyModelBay(props: WhyModelBayProps) {
  const { t } = useTranslation()

  return (
    <section
      className={cn(
        'relative z-10 mx-auto w-full max-w-7xl px-6 pb-20 md:px-10',
        props.className
      )}
    >
      {/* Mono divider: "// why model bay" centered with thin rules */}
      <div className='mb-12 flex items-center justify-center gap-4 md:mb-16'>
        <span
          aria-hidden
          className='border-border/60 h-px flex-1 max-w-[240px] border-t'
        />
        <span className='text-muted-foreground/70 font-mono text-xs tracking-wider whitespace-nowrap'>
          {t('// why model bay')}
        </span>
        <span
          aria-hidden
          className='border-border/60 h-px flex-1 max-w-[240px] border-t'
        />
      </div>

      {/* Heading */}
      <div className='mx-auto mb-10 max-w-2xl text-center md:mb-12'>
        <h2 className='text-[clamp(1.6rem,3.4vw,2.4rem)] leading-[1.15] font-medium tracking-tight'>
          {t('Designed for engineers, not for slide decks.')}
        </h2>
      </div>

      {/* Cards grid */}
      <div className='grid grid-cols-1 gap-4 sm:grid-cols-2 md:gap-5 lg:grid-cols-4'>
        {CARDS.map((card) => (
          <div
            key={card.number}
            className={cn(
              'group bg-card/60 border-border/60 hover:border-border relative flex flex-col rounded-xl border p-5 backdrop-blur-sm transition-colors',
              'dark:bg-white/[0.02] dark:border-white/10 dark:hover:border-white/20'
            )}
          >
            {/* Numbered box */}
            <div className='mb-4 inline-flex items-center gap-2'>
              <span className='border-border/70 text-foreground/80 inline-flex h-7 min-w-[2rem] items-center justify-center rounded-md border px-2 font-mono text-xs font-medium tracking-wider dark:border-white/15'>
                {card.number}
              </span>
              <span className='text-muted-foreground/40 h-px flex-1 border-t border-dashed' />
            </div>

            {/* ASCII art */}
            <pre className='text-muted-foreground/70 mb-5 overflow-hidden font-mono text-[10.5px] leading-[1.4] whitespace-pre select-none'>
              {card.ascii}
            </pre>

            {/* Title */}
            <h3 className='mb-2 text-xl font-medium tracking-tight'>
              {t(card.titleKey)}
            </h3>

            {/* Description */}
            <p className='text-muted-foreground text-sm leading-relaxed'>
              {t(card.descriptionKey)}
            </p>
          </div>
        ))}
      </div>
    </section>
  )
}

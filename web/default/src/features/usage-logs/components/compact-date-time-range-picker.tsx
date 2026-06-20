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
import { useMemo, useState } from 'react'
import { CalendarDays } from 'lucide-react'
import type { DateRange } from 'react-day-picker'
import { enUS, fr, ja, ru, vi, zhCN, zhTW } from 'react-day-picker/locale'
import { useTranslation } from 'react-i18next'
import dayjs from '@/lib/dayjs'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Calendar } from '@/components/ui/calendar'
import { Input } from '@/components/ui/input'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'

const calendarLocales = {
  en: enUS,
  zh: zhCN,
  'zh-TW': zhTW,
  fr,
  ru,
  ja,
  vi,
} as const

interface CompactDateTimeRangePickerProps {
  start?: Date
  end?: Date
  onChange: (range: { start?: Date; end?: Date }) => void
  className?: string
}

function toTimeStr(date?: Date): string {
  if (!date) return '00:00'
  return `${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}`
}

function applyTime(date: Date | undefined, timeStr: string): Date | undefined {
  if (!date) return undefined
  const [h, m] = timeStr.split(':').map(Number)
  const d = new Date(date)
  d.setHours(h, m, 0, 0)
  return d
}

export function CompactDateTimeRangePicker({
  start,
  end,
  onChange,
  className,
}: CompactDateTimeRangePickerProps) {
  const { t, i18n } = useTranslation()
  const [open, setOpen] = useState(false)
  const [range, setRange] = useState<DateRange | undefined>(() =>
    start || end ? { from: start, to: end } : undefined
  )
  const [startTime, setStartTime] = useState(() => toTimeStr(start))
  const [endTime, setEndTime] = useState(() => toTimeStr(end))

  const calendarLocale =
    calendarLocales[i18n.language as keyof typeof calendarLocales] ??
    calendarLocales[
      i18n.language.split('-')[0] as keyof typeof calendarLocales
    ] ??
    enUS

  const label = useMemo(() => {
    if (!start && !end) return t('Date Range')
    const startText = start ? dayjs(start).format('YYYY-MM-DD HH:mm') : '-'
    const endText = end ? dayjs(end).format('YYYY-MM-DD HH:mm') : '-'
    return `${startText} ~ ${endText}`
  }, [end, start, t])

  const handleOpenChange = (nextOpen: boolean) => {
    if (nextOpen) {
      setRange(start || end ? { from: start, to: end } : undefined)
      setStartTime(toTimeStr(start))
      setEndTime(toTimeStr(end))
    }
    setOpen(nextOpen)
  }

  const applyDraft = () => {
    onChange({
      start: applyTime(range?.from, startTime),
      end: applyTime(range?.to ?? range?.from, endTime),
    })
    setOpen(false)
  }

  const applyPreset = (kind: 'today' | '7d' | 'week' | '30d' | 'month') => {
    const now = dayjs()
    const presets = {
      today: {
        start: now.startOf('day').toDate(),
        end: now.endOf('day').toDate(),
      },
      '7d': {
        start: now.subtract(6, 'day').startOf('day').toDate(),
        end: now.endOf('day').toDate(),
      },
      week: {
        start: now.startOf('week').toDate(),
        end: now.endOf('week').toDate(),
      },
      '30d': {
        start: now.subtract(29, 'day').startOf('day').toDate(),
        end: now.endOf('day').toDate(),
      },
      month: {
        start: now.startOf('month').toDate(),
        end: now.endOf('month').toDate(),
      },
    }
    const p = presets[kind]
    onChange(p)
    setOpen(false)
  }

  const presetLabels: Record<string, string> = {
    today: t('Today'),
    '7d': t('7 Days'),
    week: t('This week'),
    '30d': t('30 Days'),
    month: t('This month'),
  }

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger
        render={
          <Button
            type='button'
            variant='outline'
            className={cn(
              'w-full justify-start gap-2 px-2.5 text-sm leading-5 font-normal tabular-nums',
              !start && !end && 'text-muted-foreground',
              className
            )}
          />
        }
      >
        <CalendarDays className='text-muted-foreground size-4 shrink-0' />
        <span className='truncate'>{label}</span>
      </PopoverTrigger>
      <PopoverContent align='start' className='w-auto p-3'>
        <div className='space-y-3'>
          <Calendar
            mode='range'
            selected={range}
            onSelect={setRange}
            locale={calendarLocale}
            numberOfMonths={2}
            captionLayout='label'
            className='p-0'
          />

          <div className='grid grid-cols-2 gap-2 border-t pt-3'>
            <div className='space-y-1'>
              <div className='text-muted-foreground text-xs'>
                {t('Start Time')}
              </div>
              <Input
                type='time'
                value={startTime}
                onChange={(e) => setStartTime(e.target.value)}
                className='h-7 text-xs appearance-none [&::-webkit-calendar-picker-indicator]:hidden'
              />
            </div>
            <div className='space-y-1'>
              <div className='text-muted-foreground text-xs'>
                {t('End Time')}
              </div>
              <Input
                type='time'
                value={endTime}
                onChange={(e) => setEndTime(e.target.value)}
                className='h-7 text-xs appearance-none [&::-webkit-calendar-picker-indicator]:hidden'
              />
            </div>
          </div>

          <div className='flex flex-wrap gap-1.5'>
            {(['today', '7d', 'week', '30d', 'month'] as const).map((kind) => (
              <Button
                key={kind}
                type='button'
                variant='secondary'
                size='sm'
                className='h-7 flex-1 px-2 text-xs'
                onClick={() => applyPreset(kind)}
              >
                {presetLabels[kind]}
              </Button>
            ))}
          </div>

          <div className='flex justify-end'>
            <Button size='sm' className='h-8' onClick={applyDraft}>
              {t('Confirm')}
            </Button>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  )
}

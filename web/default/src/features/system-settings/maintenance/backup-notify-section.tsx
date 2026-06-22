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
import { useEffect, useRef } from 'react'
import * as z from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useTranslation } from 'react-i18next'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import {
  SettingsControlGroup,
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'

const schema = z.object({
  AutoBackupTelegramEnabled: z.boolean(),
  AutoBackupTelegramBotToken: z.string(),
  AutoBackupHour: z.number().int().min(0).max(23),
})

type FormValues = z.infer<typeof schema>

type Props = {
  defaultValues: FormValues
}

const HOURS = Array.from({ length: 24 }, (_, i) => i)

export function BackupNotifySection({ defaultValues }: Props) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const submittingRef = useRef(false)
  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues,
  })

  useEffect(() => {
    if (!submittingRef.current) form.reset(defaultValues)
  }, [defaultValues, form])

  const onSubmit = async (values: FormValues) => {
    submittingRef.current = true
    try {
      for (const entry of [
        { key: 'AutoBackupTelegramEnabled', value: values.AutoBackupTelegramEnabled },
        { key: 'AutoBackupTelegramBotToken', value: values.AutoBackupTelegramBotToken },
        { key: 'AutoBackupHour', value: String(values.AutoBackupHour) },
      ]) {
        await updateOption.mutateAsync(entry)
      }
    } finally {
      submittingRef.current = false
    }
  }

  return (
    <SettingsSection title={t('Data Safety')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
            saveLabel={t('Save data safety settings')}
          />

          <FormField
            control={form.control}
            name='AutoBackupTelegramEnabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Telegram backup notification')}</FormLabel>
                  <FormDescription>
                    {t('Send the backup file to a Telegram chat after each automatic backup.')}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
                <FormMessage />
              </SettingsSwitchItem>
            )}
          />

          <SettingsControlGroup className='space-y-4'>
            <FormField
              control={form.control}
              name='AutoBackupTelegramBotToken'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Telegram bot token')}</FormLabel>
                  <FormControl>
                    <Input
                      placeholder='123456:ABCdefGhIjklMnoPQRsTUVwxyz'
                      type='password'
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='AutoBackupHour'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Backup upload time')}</FormLabel>
                  <FormDescription>
                    {t('Hour of day (0–23) to send the backup (UTC)')}
                  </FormDescription>
                  <Select
                    value={String(field.value)}
                    onValueChange={(v) => field.onChange(Number(v))}
                  >
                    <FormControl>
                      <SelectTrigger className='w-[120px]'>
                        <SelectValue />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectGroup>
                        {HOURS.map((h) => (
                          <SelectItem key={h} value={String(h)}>
                            {String(h).padStart(2, '0')}:00
                          </SelectItem>
                        ))}
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />
          </SettingsControlGroup>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}

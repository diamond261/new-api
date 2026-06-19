/*
 * Generate the Traditional Chinese (zh-TW) locale from the Simplified Chinese
 * (zh) locale using OpenCC's Simplified -> Traditional (Taiwan, with phrase
 * conversion) ruleset.
 *
 * Only the translation VALUES are converted; the keys are English source
 * strings and must stay byte-for-byte identical to the other locale files so
 * i18next can resolve them.
 *
 * Usage (from web/default/):
 *   bun run i18n:gen-zhtw
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import * as OpenCC from 'opencc-js'

const here = dirname(fileURLToPath(import.meta.url))
const localesDir = join(here, '..', 'src', 'i18n', 'locales')
const srcPath = join(localesDir, 'zh.json')
const outPath = join(localesDir, 'zh-TW.json')

// cn  = Simplified Chinese (Mainland)
// twp = Traditional Chinese (Taiwan) with idiom/phrase conversion
const convert = OpenCC.Converter({ from: 'cn', to: 'twp' })

/** Recursively convert only string values; keep object keys untouched. */
function deepConvert(node) {
  if (typeof node === 'string') return convert(node)
  if (Array.isArray(node)) return node.map(deepConvert)
  if (node && typeof node === 'object') {
    const out = {}
    for (const [key, value] of Object.entries(node)) {
      out[key] = deepConvert(value)
    }
    return out
  }
  return node
}

const source = JSON.parse(readFileSync(srcPath, 'utf8'))
const converted = deepConvert(source)
writeFileSync(outPath, JSON.stringify(converted, null, 2) + '\n', 'utf8')

const count = Object.keys(converted.translation ?? {}).length
console.log(`Wrote ${outPath} (${count} keys) from ${srcPath}`)

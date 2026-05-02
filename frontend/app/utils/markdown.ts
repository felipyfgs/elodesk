import MarkdownIt from 'markdown-it'
import DOMPurify from 'dompurify'

const md = new MarkdownIt({
  html: false,
  breaks: true,
  linkify: true,
  typographer: false
})

const waBoldRE = /\*([^\s*](?:[^\n*]*?[^\s*])?)\*/g
const waStrikeRE = /~([^\s~](?:[^\n~]*?[^\s~])?)~/g
const hardBreakRE = /\\(\r?\n)/g
const backslashEscapeRE = /\\([!"#$%&'()*+,\-./:;<=>?@[\\\]^_`{|}~])/g

const PH_TRIPLE_AST = '\x00ta'
const PH_DOUBLE_AST = '\x00da'
const PH_TRIPLE_TIL = '\x00tt'
const PH_DOUBLE_TIL = '\x00dt'

function normalizeForRender(s: string): string {
  s = s.replace(hardBreakRE, '$1')

  s = s.replace(backslashEscapeRE, '$1')

  s = s.replaceAll('***', PH_TRIPLE_AST)
  s = s.replaceAll('**', PH_DOUBLE_AST)
  s = s.replaceAll('~~~', PH_TRIPLE_TIL)
  s = s.replaceAll('~~', PH_DOUBLE_TIL)

  s = s.replace(waBoldRE, '**$1**')
  s = s.replace(waStrikeRE, '~~$1~~')

  s = s.replaceAll(PH_TRIPLE_AST, '***')
  s = s.replaceAll(PH_DOUBLE_AST, '**')
  s = s.replaceAll(PH_TRIPLE_TIL, '~~~')
  s = s.replaceAll(PH_DOUBLE_TIL, '~~')

  return s
}

const ALLOWED_TAGS = ['p', 'strong', 'em', 's', 'code', 'pre', 'a', 'ul', 'ol', 'li', 'br', 'blockquote', 'hr']
const ALLOWED_INLINE_TAGS = ['strong', 'em', 's', 'code', 'a', 'br']
const ALLOWED_ATTR = ['href', 'target', 'rel']

export function renderMarkdown(input: string): string {
  if (!input) return ''
  const html = md.render(normalizeForRender(input))
  return DOMPurify.sanitize(html, { ALLOWED_TAGS, ALLOWED_ATTR }).trim()
}

export function renderMarkdownInline(input: string): string {
  if (!input) return ''
  const html = md.renderInline(normalizeForRender(input))
  return DOMPurify.sanitize(html, { ALLOWED_TAGS: ALLOWED_INLINE_TAGS, ALLOWED_ATTR })
}

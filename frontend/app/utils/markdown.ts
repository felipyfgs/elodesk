import MarkdownIt from 'markdown-it'
import DOMPurify from 'dompurify'

const md = new MarkdownIt({
  html: false,
  breaks: true,
  linkify: true,
  typographer: false
})

// Normalização aplicada ao conteúdo antes de passar pro markdown-it. Cobre:
//
//  1. Hard-breaks `\\\n` → `\n` (chatwoot#13669, prosemirror-markdown).
//  2. Desescape `\X` → `X` para caracteres especiais do CommonMark
//     (tiptap#7258 — escape excessivo do tiptap-markdown).
//  3. WA dialect → Markdown padrão:
//       *X*  → **X**   (bold)        — seguro porque italic do nosso editor
//                                       sai como _X_ via ItalicUnderscore;
//                                       conteúdo gravado pelo tiptap não
//                                       usa `*X*` para italic.
//       ~X~  → ~~X~~  (strike)       — single-tilde não tem significado em
//                                       CommonMark, é puro upgrade.
//     `_X_` já é italic em ambos formatos, não precisa converter.
//
// Sequências literais `**`/`***`/`~~`/`~~~` são protegidas com placeholders
// pra que a conversão single-delim não as fragmente.

const waBoldRE = /\*([^\s*](?:[^\n*]*?[^\s*])?)\*/g
const waStrikeRE = /~([^\s~](?:[^\n~]*?[^\s~])?)~/g
const hardBreakRE = /\\(\r?\n)/g
const backslashEscapeRE = /\\([!"#$%&'()*+,\-./:;<=>?@[\\\]^_`{|}~])/g

const PH_TRIPLE_AST = '\x00ta'
const PH_DOUBLE_AST = '\x00da'
const PH_TRIPLE_TIL = '\x00tt'
const PH_DOUBLE_TIL = '\x00dt'

function normalizeForRender(s: string): string {
  // 1. Hard-breaks. Antes do desescape pra não consumir o backslash do
  //    hard break como escape genérico.
  s = s.replace(hardBreakRE, '$1')

  // 2. Desescape de caracteres especiais do CommonMark.
  s = s.replace(backslashEscapeRE, '$1')

  // 3. Proteger delimitadores duplos/triplos literais (ordem: triple antes
  //    de double).
  s = s.replaceAll('***', PH_TRIPLE_AST)
  s = s.replaceAll('**', PH_DOUBLE_AST)
  s = s.replaceAll('~~~', PH_TRIPLE_TIL)
  s = s.replaceAll('~~', PH_DOUBLE_TIL)

  // 4. Conversão WA → Markdown.
  s = s.replace(waBoldRE, '**$1**')
  s = s.replace(waStrikeRE, '~~$1~~')

  // 5. Restaurar literais.
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
  // Trim because markdown-it appends a trailing newline. Bubbles use
  // `whitespace-pre-wrap`, so that `\n` would render as a visible blank line.
  return DOMPurify.sanitize(html, { ALLOWED_TAGS, ALLOWED_ATTR }).trim()
}

export function renderMarkdownInline(input: string): string {
  if (!input) return ''
  const html = md.renderInline(normalizeForRender(input))
  return DOMPurify.sanitize(html, { ALLOWED_TAGS: ALLOWED_INLINE_TAGS, ALLOWED_ATTR })
}

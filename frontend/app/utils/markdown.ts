import MarkdownIt from 'markdown-it'
import DOMPurify from 'dompurify'

const md = new MarkdownIt({
  html: false,
  breaks: true,
  linkify: true,
  typographer: false
})

export function renderMarkdown(input: string): string {
  if (!input) return ''
  const html = md.render(input)
  // Trim because markdown-it appends a trailing newline. Bubbles use
  // `whitespace-pre-wrap`, so that `\n` would render as a visible blank line.
  return DOMPurify.sanitize(html, {
    ALLOWED_TAGS: ['p', 'strong', 'em', 'code', 'pre', 'a', 'ul', 'ol', 'li', 'br', 'blockquote', 'hr'],
    ALLOWED_ATTR: ['href', 'target', 'rel']
  }).trim()
}

export function renderMarkdownInline(input: string): string {
  if (!input) return ''
  const html = md.renderInline(input)
  return DOMPurify.sanitize(html, {
    ALLOWED_TAGS: ['strong', 'em', 'code', 'a', 'br'],
    ALLOWED_ATTR: ['href', 'target', 'rel']
  })
}

// apiAdapter normalizes the backend response shape to the camelCase + ISO
// timestamp form the frontend was originally written against.
//
// Backend now serves Chatwoot-shape (snake_case keys, epoch-second timestamps)
// for conversation/contact/message/inbox payloads. Rather than rewriting every
// store/component, we walk the response and:
//   1. rename keys snake_case -> camelCase (recursive)
//   2. for epoch-second numeric timestamps (keys ending in *_at / *At), expand
//      to milliseconds so `new Date(value)` keeps working
//   3. preserve already-camelCase keys, JSON.RawMessage objects, and arrays
//
// We intentionally do NOT mutate strings, ints that aren't timestamps, or
// arbitrary JSONB blobs (additional_attributes, custom_attributes, content_attributes)
// — those are passed through verbatim so the backend's opaque schema stays opaque.

const EPOCH_KEY_SUFFIXES = ['_at', 'At']
const EPOCH_KEYS_FORCE = new Set([
  'timestamp', 'waiting_since', 'snoozed_until',
  'first_reply_created_at', 'agent_last_seen_at',
  'assignee_last_seen_at', 'contact_last_seen_at'
])
// Keys whose value is opaque JSONB — leave untouched (don't recurse, don't
// transform). The backend serializes these as raw json.RawMessage.
const OPAQUE_KEYS = new Set([
  'additional_attributes', 'additionalAttributes',
  'custom_attributes', 'customAttributes',
  'content_attributes', 'contentAttributes',
  'metadata'
])

function snakeToCamel(s: string): string {
  if (!s.includes('_')) return s
  return s.replace(/_([a-z0-9])/g, (_, c) => c.toUpperCase())
}

function looksLikeEpoch(key: string): boolean {
  if (EPOCH_KEYS_FORCE.has(key)) return true
  return EPOCH_KEY_SUFFIXES.some(suffix => key.endsWith(suffix))
}

function maybeExpandEpoch(key: string, value: unknown): unknown {
  if (!looksLikeEpoch(key)) return value
  if (typeof value !== 'number') return value
  // Heuristic: epoch seconds have ~10 digits (until year 2286); epoch ms have
  // ~13. Anything bigger than 1e12 is already ms — leave it alone.
  if (value === 0) return value
  if (value > 1_000_000_000_000) return value
  return value * 1000
}

export function normalizeApiResponse<T = unknown>(input: unknown): T {
  if (input === null || input === undefined) return input as T
  if (Array.isArray(input)) return input.map(item => normalizeApiResponse(item)) as T
  if (typeof input !== 'object') return input as T
  // Date and other built-ins shouldn't be walked.
  if (input instanceof Date) return input as T
  const out: Record<string, unknown> = {}
  for (const [key, value] of Object.entries(input as Record<string, unknown>)) {
    const camelKey = snakeToCamel(key)
    if (OPAQUE_KEYS.has(key)) {
      // Opaque JSONB: copy verbatim under both keys so consumers reading the
      // snake_case form (e.g. additional_attributes.is_group) still find it.
      out[camelKey] = value
      out[key] = value
      continue
    }
    const expanded = maybeExpandEpoch(key, value)
    out[camelKey] = expanded === value
      ? normalizeApiResponse(value)
      : expanded
  }
  return out as T
}

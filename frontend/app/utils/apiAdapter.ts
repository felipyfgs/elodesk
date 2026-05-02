const EPOCH_KEY_SUFFIXES = ['_at', 'At']
const EPOCH_KEYS_FORCE = new Set([
  'timestamp', 'waiting_since', 'snoozed_until',
  'first_reply_created_at', 'agent_last_seen_at',
  'assignee_last_seen_at', 'contact_last_seen_at'
])
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
  if (value === 0) return value
  if (value > 1_000_000_000_000) return value
  return value * 1000
}

export function normalizeApiResponse<T = unknown>(input: unknown): T {
  if (input === null || input === undefined) return input as T
  if (Array.isArray(input)) return input.map(item => normalizeApiResponse(item)) as T
  if (typeof input !== 'object') return input as T
  if (input instanceof Date) return input as T
  const out: Record<string, unknown> = {}
  for (const [key, value] of Object.entries(input as Record<string, unknown>)) {
    const camelKey = snakeToCamel(key)
    if (OPAQUE_KEYS.has(key)) {
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

// parseJsonAttrs coerces a JSONB blob field (additional_attributes,
// custom_attributes, content_attributes) into a Record. The backend serializes
// these as `json.RawMessage`, so apiAdapter passes them through verbatim and
// they may arrive already parsed as objects or as legacy JSON strings depending
// on the producer. Returns {} on missing/invalid input.
export function parseJsonAttrs(raw: unknown): Record<string, unknown> {
  if (raw == null) return {}
  if (typeof raw === 'object') return raw as Record<string, unknown>
  if (typeof raw === 'string') {
    if (!raw) return {}
    try {
      const parsed = JSON.parse(raw) as unknown
      return typeof parsed === 'object' && parsed !== null ? parsed as Record<string, unknown> : {}
    } catch {
      return {}
    }
  }
  return {}
}

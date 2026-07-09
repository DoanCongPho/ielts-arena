// Content columns from the API (test.content_data, submission.payload,
// score.details) are already parsed JSON objects in normal responses, but
// this stays defensive in case a caller passes the raw string form.
export function safeParse(raw) {
  if (raw == null) return null;
  try {
    return typeof raw === 'string' ? JSON.parse(raw) : raw;
  } catch {
    return null;
  }
}

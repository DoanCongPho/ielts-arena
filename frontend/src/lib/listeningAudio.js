// Timing helpers for a listening test's recording: sections either share
// one file (content.audio_url, each a time range in it) or each have their
// own (section.audio_url, timed within that file).

// playlist lists what to play, in order: each section's own recording, or
// the test's one shared recording.
export function playlist(content) {
  const sections = content?.sections || [];
  if (sections.length > 0 && sections.every((s) => s.audio_url)) {
    return sections.map((s) => ({ src: s.audio_url, length: sectionLength(s) }));
  }
  return content?.audio_url ? [{ src: content.audio_url, length: totalSeconds(content) }] : [];
}

function sectionLength(s) {
  return Math.max(0, (s.section_end_time || 0) - (s.section_start_time || 0));
}

// totalSeconds is how long the whole recording runs.
export function totalSeconds(content) {
  return (content?.sections || []).reduce((sum, s) => sum + sectionLength(s), 0);
}

// sectionAt finds which section a position in a shared recording is in.
export function sectionAt(content, seconds) {
  const sections = content?.sections || [];
  const i = sections.findIndex((s) => seconds < (s.section_end_time || 0));
  return i === -1 ? Math.max(0, sections.length - 1) : i;
}

import { useRef } from 'react';

// useSectionAudio drives the single <audio> player across a listening
// test's sections, which either share the test's recording (content
// .audio_url, each section a time range in it) or each have their own
// (section.audio_url, times counted within that file). Spread `audioProps`
// onto the <audio>; seek(i, t) moves playback to t seconds into section i's
// file — call it before switching the active section, and if that swaps
// the file, the seek waits until the new file has loaded.
export function useSectionAudio(content, activeIndex) {
  const audioRef = useRef(null);
  const pendingSeek = useRef(null);

  const srcOf = (i) => content?.sections?.[i]?.audio_url || content?.audio_url;

  function seek(i, seconds) {
    const t = seconds ?? 0;
    if (srcOf(i) === srcOf(activeIndex) && audioRef.current) {
      audioRef.current.currentTime = t;
    } else {
      pendingSeek.current = t;
    }
  }

  function onLoadedMetadata() {
    if (pendingSeek.current != null && audioRef.current) {
      audioRef.current.currentTime = pendingSeek.current;
      pendingSeek.current = null;
    }
  }

  function play() {
    audioRef.current?.play()?.catch(() => {});
  }

  return { seek, play, audioProps: { ref: audioRef, src: srcOf(activeIndex), onLoadedMetadata } };
}

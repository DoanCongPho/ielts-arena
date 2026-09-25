// Recording answers and playing the examiner, for the speaking runner.

// Containers in order of preference. Chrome and Firefox record Opus in
// WebM/Ogg; Safari only records AAC in MP4. The server accepts all of them.
const RECORDING_TYPES = [
  { mime: 'audio/webm;codecs=opus', ext: 'webm' },
  { mime: 'audio/ogg;codecs=opus', ext: 'ogg' },
  { mime: 'audio/mp4', ext: 'mp4' },
];

export function recordingFormat() {
  if (typeof MediaRecorder === 'undefined') return null;
  return RECORDING_TYPES.find((t) => MediaRecorder.isTypeSupported(t.mime)) || null;
}

// openMicrophone asks for the mic once per attempt, with the processing a
// voice call uses, so answers are clear on a laptop's built-in mic.
export function openMicrophone() {
  return navigator.mediaDevices.getUserMedia({
    audio: { channelCount: 1, echoCancellation: true, noiseSuppression: true, autoGainControl: true },
  });
}

// startRecording records from stream until the returned stop() is called,
// which resolves to {blob, durationSec}.
export function startRecording(stream, format) {
  const recorder = new MediaRecorder(stream, { mimeType: format.mime, audioBitsPerSecond: 32000 });
  const chunks = [];
  const startedAt = performance.now();
  recorder.ondataavailable = (e) => {
    if (e.data.size > 0) chunks.push(e.data);
  };
  recorder.start(1000);

  return function stop() {
    return new Promise((resolve) => {
      recorder.onstop = () => {
        resolve({
          blob: new Blob(chunks, { type: format.mime.split(';')[0] }),
          durationSec: (performance.now() - startedAt) / 1000,
        });
      };
      if (recorder.state !== 'inactive') recorder.stop();
    });
  };
}

// speakLine plays the examiner saying text: the recorded examiner voice
// when there is one, else the browser's own British English voice. It
// resolves when the line has been spoken, and rejects if cancelled via
// the signal.
export function speakLine(line, signal) {
  if (line.audio_url) {
    return playRecording(line.audio_url, signal).catch((err) => {
      if (signal?.aborted) throw err;
      return speakWithBrowser(line.text, signal);
    });
  }
  return speakWithBrowser(line.text, signal);
}

function playRecording(url, signal) {
  return new Promise((resolve, reject) => {
    const audio = new Audio(url);
    const abort = () => {
      audio.pause();
      reject(new DOMException('aborted', 'AbortError'));
    };
    signal?.addEventListener('abort', abort, { once: true });
    audio.onended = () => {
      signal?.removeEventListener('abort', abort);
      resolve();
    };
    audio.onerror = () => reject(new Error('examiner audio failed'));
    audio.play().catch(reject);
  });
}

function speakWithBrowser(text, signal) {
  if (typeof speechSynthesis === 'undefined' || !text) return Promise.resolve();
  return new Promise((resolve, reject) => {
    const u = new SpeechSynthesisUtterance(text);
    u.lang = 'en-GB';
    u.rate = 0.95;
    const voice = speechSynthesis.getVoices().find((v) => v.lang === 'en-GB');
    if (voice) u.voice = voice;
    const abort = () => {
      speechSynthesis.cancel();
      reject(new DOMException('aborted', 'AbortError'));
    };
    signal?.addEventListener('abort', abort, { once: true });
    u.onend = () => {
      signal?.removeEventListener('abort', abort);
      resolve();
    };
    // A voice failing mid-line must not stall the test.
    u.onerror = () => resolve();
    speechSynthesis.speak(u);
  });
}

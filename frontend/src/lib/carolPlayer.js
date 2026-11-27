import { useSyncExternalStore } from 'react';
import { CAROLS } from './carols';

// carolPlayer plays the carols in lib/carols.js as a music box, synthesised
// with Web Audio — no audio files. It's a module singleton, so music keeps
// playing while pages change; the MusicPlayer component is only its remote
// control. Notes are scheduled a little ahead on the AudioContext clock
// (the usual lookahead pattern), so timing doesn't depend on React or on
// setInterval's jitter. Pausing suspends the context, which freezes that
// clock, so a song resumes exactly where it stopped.

const PREF_KEY = 'ielts.music';
const CHANGE_EVENT = 'carol-player:change';
const LOOKAHEAD_S = 0.3;
const TICK_MS = 60;
const GAP_BETWEEN_SONGS_S = 1.8;

// --- State the UI sees (persisted, except whether audio is unlocked) ----

function loadPrefs() {
  try {
    const p = JSON.parse(localStorage.getItem(PREF_KEY) || '{}');
    return {
      on: !!p.on,
      track: Number.isInteger(p.track) && p.track >= 0 && p.track < CAROLS.length ? p.track : 0,
      volume: typeof p.volume === 'number' ? Math.min(1, Math.max(0, p.volume)) : 0.5,
    };
  } catch {
    return { on: false, track: 0, volume: 0.5 };
  }
}

let state = { ...loadPrefs(), blocked: false, playing: false };

function setState(patch) {
  state = { ...state, ...patch };
  try {
    localStorage.setItem(PREF_KEY, JSON.stringify({ on: state.on, track: state.track, volume: state.volume }));
  } catch {
    // Storage blocked: settings just won't survive a reload.
  }
  window.dispatchEvent(new Event(CHANGE_EVENT));
}

function subscribe(listener) {
  window.addEventListener(CHANGE_EVENT, listener);
  return () => window.removeEventListener(CHANGE_EVENT, listener);
}

// --- Music theory -------------------------------------------------------

const SEMITONES = { C: 0, D: 2, E: 4, F: 5, G: 7, A: 9, B: 11 };

function midiOf(note) {
  const m = /^([A-G])([#b]?)(-?\d)$/.exec(note);
  if (!m) return null;
  const accidental = m[2] === '#' ? 1 : m[2] === 'b' ? -1 : 0;
  return 12 * (Number(m[3]) + 1) + SEMITONES[m[1]] + accidental;
}

const freqOf = (midi) => 440 * 2 ** ((midi - 69) / 12);

function chordTones(name) {
  const m = /^([A-G][#b]?)(m|7|maj7)?$/.exec(name);
  if (!m) return null;
  const root = midiOf(`${m[1]}3`);
  const shape = m[2] === 'm' ? [0, 3, 7] : m[2] === '7' ? [0, 4, 7, 10] : [0, 4, 7];
  return { root, upper: shape.slice(1).map((s) => root + 12 + s) };
}

// Turns a carol into timed events: [{ beat, beats, kind, midi(s) }].
function eventsOf(song) {
  const events = [];
  let beat = 0;
  for (const token of song.melody.split(/\s+/)) {
    const [note, len] = token.split(':');
    const beats = Number(len);
    const midi = note === 'R' ? null : midiOf(note);
    if (midi != null) events.push({ beat, beats, kind: 'melody', midi: midi + (song.transpose || 0) });
    beat += beats;
  }
  const melodyEnd = beat;

  song.chords.split(/\s+/).forEach((name, bar) => {
    const tones = chordTones(name);
    if (!tones) return;
    const start = song.pickup + bar * song.beatsPerBar;
    // Oom-pah: the root on the downbeat, the chord on the other beats.
    events.push({ beat: start, beats: song.beatsPerBar, kind: 'bass', midi: tones.root });
    for (let b = 1; b < song.beatsPerBar; b++) {
      events.push({ beat: start + b, beats: 0.6, kind: 'chord', midis: tones.upper });
    }
    if (song.sleighBells) {
      for (let b = 0; b < song.beatsPerBar; b++) events.push({ beat: start + b + 0.5, beats: 0.2, kind: 'bells' });
    }
  });

  events.sort((a, b) => a.beat - b.beat);
  const totalBeats = Math.max(melodyEnd, song.pickup + song.chords.split(/\s+/).length * song.beatsPerBar);
  return { events, totalBeats };
}

// --- Audio --------------------------------------------------------------

let ctx = null;
let master = null;
let noise = null;
let timer = null;
let live = new Set(); // sources still sounding, stopped when skipping
let song = null; // { events, totalBeats, index, start, spb, next }

function ensureContext() {
  if (ctx) return;
  const Ctx = window.AudioContext || window.webkitAudioContext;
  if (!Ctx) return;
  ctx = new Ctx();
  master = ctx.createGain();
  master.gain.value = gainFor(state.volume);
  const comp = ctx.createDynamicsCompressor();
  master.connect(comp).connect(ctx.destination);

  // One second of white noise, reused for every sleigh-bell shake.
  noise = ctx.createBuffer(1, ctx.sampleRate, ctx.sampleRate);
  const data = noise.getChannelData(0);
  for (let i = 0; i < data.length; i++) data[i] = Math.random() * 2 - 1;
}

// Loudness feels roughly logarithmic, so square the slider.
const gainFor = (volume) => volume * volume * 0.9;

function track(node, stopAt) {
  node.start(stopAt[0]);
  node.stop(stopAt[1]);
  live.add(node);
  node.onended = () => live.delete(node);
}

// A music-box note: a sine with a quiet octave and a triangle body,
// struck fast and left to ring down.
function bell(midi, t, dur, level) {
  const f = freqOf(midi);
  const ring = Math.max(0.5, dur) + 0.6;
  const env = ctx.createGain();
  env.gain.setValueAtTime(0, t);
  env.gain.linearRampToValueAtTime(level, t + 0.006);
  env.gain.exponentialRampToValueAtTime(0.0004, t + ring);
  env.connect(master);
  for (const [type, mult, amp] of [['sine', 1, 1], ['sine', 2, 0.28], ['triangle', 1, 0.35]]) {
    const osc = ctx.createOscillator();
    osc.type = type;
    osc.frequency.value = f * mult;
    const g = ctx.createGain();
    g.gain.value = amp;
    osc.connect(g).connect(env);
    track(osc, [t, t + ring + 0.05]);
  }
}

function soft(midi, t, dur, level, type) {
  const env = ctx.createGain();
  env.gain.setValueAtTime(0, t);
  env.gain.linearRampToValueAtTime(level, t + 0.02);
  env.gain.exponentialRampToValueAtTime(0.0004, t + dur);
  env.connect(master);
  const osc = ctx.createOscillator();
  osc.type = type;
  osc.frequency.value = freqOf(midi);
  osc.connect(env);
  track(osc, [t, t + dur + 0.05]);
}

function sleighBells(t) {
  const src = ctx.createBufferSource();
  src.buffer = noise;
  const hp = ctx.createBiquadFilter();
  hp.type = 'highpass';
  hp.frequency.value = 7000;
  const env = ctx.createGain();
  env.gain.setValueAtTime(0.05, t);
  env.gain.exponentialRampToValueAtTime(0.0004, t + 0.14);
  src.connect(hp).connect(env).connect(master);
  track(src, [t, t + 0.16]);
}

function schedule(e, t, spb) {
  const dur = e.beats * spb;
  if (e.kind === 'melody') bell(e.midi, t, dur, 0.32);
  else if (e.kind === 'bass') soft(e.midi, t, Math.min(dur, 1.6), 0.2, 'triangle');
  else if (e.kind === 'chord') e.midis.forEach((m) => soft(m, t, Math.min(dur, 0.5), 0.05, 'sine'));
  else if (e.kind === 'bells') sleighBells(t);
}

function silenceAll() {
  for (const node of live) {
    try {
      node.stop();
    } catch {
      // already stopped
    }
  }
  live = new Set();
}

function startSong(index) {
  silenceAll();
  const carol = CAROLS[index];
  const { events, totalBeats } = eventsOf(carol);
  const spb = 60 / carol.bpm;
  song = { events, totalBeats, index, spb, start: ctx.currentTime + 0.12, next: 0 };
  if (state.track !== index) setState({ track: index });
}

function tick() {
  if (!ctx || !song || ctx.state !== 'running') return;
  const horizon = ctx.currentTime + LOOKAHEAD_S;
  while (song.next < song.events.length) {
    const e = song.events[song.next];
    const t = song.start + e.beat * song.spb;
    if (t > horizon) break;
    if (t >= ctx.currentTime - 0.05) schedule(e, t, song.spb);
    song.next++;
  }
  // Past the end (plus a breath): on to the next carol, looping the list.
  if (ctx.currentTime > song.start + song.totalBeats * song.spb + GAP_BETWEEN_SONGS_S) {
    startSong((song.index + 1) % CAROLS.length);
  }
}

// Brings the audio in line with `state`: plays when on and not blocked.
async function sync() {
  const shouldPlay = state.on && !state.blocked;
  if (!shouldPlay) {
    clearInterval(timer);
    timer = null;
    if (ctx && ctx.state === 'running') await ctx.suspend();
    if (state.playing) setState({ playing: false });
    return;
  }
  ensureContext();
  if (!ctx) return;
  if (ctx.state !== 'running') {
    try {
      await ctx.resume();
    } catch {
      return;
    }
  }
  // A browser keeps audio locked until the page has had a click or key
  // press; until then the context stays suspended and we wait for one.
  if (ctx.state !== 'running') {
    armUnlock();
    return;
  }
  if (!song || song.index !== state.track) startSong(state.track);
  if (!timer) timer = setInterval(tick, TICK_MS);
  tick();
  if (!state.playing) setState({ playing: true });
}

let unlockArmed = false;
function armUnlock() {
  if (unlockArmed) return;
  unlockArmed = true;
  const unlock = () => {
    unlockArmed = false;
    window.removeEventListener('pointerdown', unlock);
    window.removeEventListener('keydown', unlock);
    sync();
  };
  window.addEventListener('pointerdown', unlock);
  window.addEventListener('keydown', unlock);
}

// --- Controls -----------------------------------------------------------

export const carolPlayer = {
  toggle() {
    setState({ on: !state.on });
    sync();
  },
  next() {
    const index = (state.track + 1) % CAROLS.length;
    setState({ track: index });
    if (ctx && state.playing) startSong(index);
    sync();
  },
  prev() {
    const index = (state.track - 1 + CAROLS.length) % CAROLS.length;
    setState({ track: index });
    if (ctx && state.playing) startSong(index);
    sync();
  },
  setVolume(volume) {
    const v = Math.min(1, Math.max(0, volume));
    setState({ volume: v });
    if (master) master.gain.setTargetAtTime(gainFor(v), ctx.currentTime, 0.05);
  },
  // Pages that must stay quiet (taking a test) block the music while
  // they're open; it resumes when they close if it was on.
  setBlocked(blocked) {
    if (state.blocked === blocked) return;
    setState({ blocked });
    sync();
  },
};

// Music that was on before a reload resumes at the first click or key.
if (state.on) armUnlock();

export function useCarolPlayer() {
  const s = useSyncExternalStore(subscribe, () => state, () => state);
  return { ...s, title: CAROLS[s.track].title, count: CAROLS.length, ...carolPlayer };
}

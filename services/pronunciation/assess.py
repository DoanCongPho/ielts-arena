"""Scoring one recording: each word against its references, then prosody."""

from __future__ import annotations

import numpy as np

import scoring
from audio import SAMPLE_RATE
from model import BLANK, FRAME_S, ID_TO_PHONE, chunk_log_probs
from phonemes import VARIANTS, Reference, all_references, weak_references

# Seconds added around a word's recognised timing for alignment, which
# absorbs recogniser timing error.
WORD_PAD_S = 0.15
# A word scoring below this is reported as not clearly intelligible.
INTELLIGIBLE_AT = 60.0


def frames_for(chunks, start: float, end: float) -> tuple[float, np.ndarray] | None:
    """The log-probability frames covering [start, end], clipped to the
    chunk that holds the word's start (a padded window may run past the
    end of the audio)."""
    for offset, lp in chunks:
        chunk_end = offset + lp.shape[0] * FRAME_S
        if offset - WORD_PAD_S <= start < chunk_end:
            a = max(int((start - offset) / FRAME_S), 0)
            b = min(int(np.ceil((end - offset) / FRAME_S)), lp.shape[0])
            if b > a:
                return offset + a * FRAME_S, lp[a:b]
    return None


def assess_word(chunks, ref: Reference, start: float, end: float) -> dict | None:
    win = frames_for(chunks, max(start - WORD_PAD_S, 0.0), end + WORD_PAD_S)
    if win is None or not ref.ids:
        return None
    offset, lp = win
    path = scoring.ctc_force_align(lp, ref.ids, BLANK)
    if path is None:
        return None
    scores = scoring.phoneme_scores(lp, ref.ids, path, BLANK)
    tight = frames_for(chunks, start, end) or win
    heard = [ID_TO_PHONE.get(i, "") for i in scoring.greedy_decode(tight[1], BLANK)]
    heard_per_phone, edits = scoring.align_phones(ref.phones, heard)
    return {
        "score": float(np.mean(scores)),
        "phonemes": [
            {"expected": p, "heard": h, "score": round(s, 1)}
            for p, h, s in zip(ref.phones, heard_per_phone, scores)
        ],
        "edits": edits,
        "heard": " ".join(heard),
        "spans": scoring.phone_spans(ref.phones, path, offset, FRAME_S, end),
        "stressed": ref.stressed,
    }


def prosody(audio: np.ndarray, assessed: list[dict]) -> dict:
    f0_times, f0 = scoring.pitch_track(audio.astype(np.float64), SAMPLE_RATE)
    voiced = f0 > 0
    pitch_std, pitch_range = scoring.pitch_stats(f0)
    db_times, db = scoring.intensity_db(audio.astype(np.float64), SAMPLE_RATE)

    def mean_in(values, times, a, b, mask=None):
        sel = (times >= a) & (times < b)
        if mask is not None:
            sel &= mask
        return float(values[sel].mean()) if sel.any() else 0.0

    stress_checked = stress_matched = 0
    vowel_durations = []
    for w in assessed:
        vowels = [s for s in w["spans"] if scoring.is_vowel(s.phone)]
        vowel_durations += [s.end - s.start for s in vowels]
        if len(vowels) < 2 or w["stressed"] < 0 or w["stressed"] >= len(vowels):
            continue
        feats = [
            (v.end - v.start, mean_in(db, db_times, v.start, v.end), mean_in(f0, f0_times, v.start, v.end, voiced))
            for v in vowels
        ]
        stress_checked += 1
        stress_matched += scoring.most_prominent(feats) == w["stressed"]

    return {
        "pitch_range_st": round(pitch_range, 2),
        "pitch_std_st": round(pitch_std, 2),
        "stress_match_rate": round(stress_matched / stress_checked, 3) if stress_checked else 0.0,
        "npvi_vowel": round(scoring.npvi(vowel_durations), 1),
    }


def assess_recording(pcm: np.ndarray, timed: list[dict], texts: list[str]) -> dict:
    """Scores every word of one recording and summarises it. timed are the
    recogniser's {word, start, end}; texts their cleaned words ("" to skip)."""
    chunks = chunk_log_probs(pcm, [w for w, t in zip(timed, texts) if t])
    refs = all_references(texts)

    results, chosen, counts = [], [], {v: 0 for v in VARIANTS}
    total_phones = total_edits = 0
    for i, w in enumerate(timed):
        best, best_variant = None, VARIANTS[0]
        candidates = [(v, ref[i]) for (v, _), ref in refs.items()]
        candidates += [(None, r) for r in weak_references(texts[i])]
        for v, ref in candidates if texts[i] else ():
            r = assess_word(chunks, ref, float(w["start"]), float(w["end"]))
            if r is not None and (best is None or r["score"] > best["score"]):
                best, best_variant = r, v
        entry = {"word": texts[i], "start": w["start"], "end": w["end"], "score": 0.0, "phonemes": []}
        if best is not None:
            if best_variant is not None:
                counts[best_variant] += 1
            chosen.append(best)
            total_phones += len(best["phonemes"])
            total_edits += best["edits"]
            entry.update(score=round(best["score"], 1), phonemes=best["phonemes"], heard=best["heard"])
        # One entry per input word, in order, so the caller can map scores
        # back onto its own transcript by index.
        results.append(entry)

    scored = [r for r in results if r["phonemes"]]
    return {
        "intelligibility": round(sum(r["score"] >= INTELLIGIBLE_AT for r in scored) / len(scored), 3) if scored else 0.0,
        "gop_mean": round(float(np.mean([r["score"] for r in scored])), 1) if scored else 0.0,
        "per": round(total_edits / total_phones, 3) if total_phones else 1.0,
        "accent_variant": max(counts, key=counts.get),
        "words": results,
        "prosody": prosody(pcm, chosen),
    }

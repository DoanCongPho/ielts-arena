"""Pure scoring maths for the pronunciation service: CTC forced alignment,
goodness of pronunciation, phoneme matching and rhythm. Kept free of the
model and audio libraries so it can be tested on its own."""

from __future__ import annotations

import math
from dataclasses import dataclass

import numpy as np

# IPA characters that start a vowel phoneme in espeak's output (diphthongs
# such as "aɪ" and "oʊ" start with a vowel too).
VOWEL_STARTS = set("aeiouyɑɐɒæɛəɜɪʊʌɔøœɨʉɯɤʏɚɝ")

# Log-posterior margin floor when turning GOP into a 0-100 score, so one
# terrible frame can't dominate a phoneme's mean.
GOP_FLOOR = -6.0


def is_vowel(phone: str) -> bool:
    return bool(phone) and phone[0] in VOWEL_STARTS


def ctc_force_align(log_probs: np.ndarray, targets: list[int], blank: int) -> list[int] | None:
    """Viterbi CTC alignment of targets to frames.

    Returns, for each frame, the index into targets that frame is aligned
    to, or -1 for a blank frame. Returns None when the frames are too few to
    hold the targets (CTC needs a blank between repeated labels).
    """
    T = log_probs.shape[0]
    L = len(targets)
    if L == 0:
        return [-1] * T
    ext = [blank]
    for t in targets:
        ext += [t, blank]
    S = len(ext)
    repeats = sum(1 for a, b in zip(targets, targets[1:]) if a == b)
    if T < L + repeats:
        return None

    neg = -np.inf
    dp = np.full((T, S), neg)
    back = np.zeros((T, S), dtype=np.int8)  # 0: stay, 1: from s-1, 2: from s-2
    dp[0, 0] = log_probs[0, ext[0]]
    dp[0, 1] = log_probs[0, ext[1]]
    for t in range(1, T):
        for s in range(S):
            best, move = dp[t - 1, s], 0
            if s >= 1 and dp[t - 1, s - 1] > best:
                best, move = dp[t - 1, s - 1], 1
            if s >= 2 and ext[s] != blank and ext[s] != ext[s - 2] and dp[t - 1, s - 2] > best:
                best, move = dp[t - 1, s - 2], 2
            if best == neg:
                continue
            dp[t, s] = best + log_probs[t, ext[s]]
            back[t, s] = move

    s = S - 1 if dp[T - 1, S - 1] >= dp[T - 1, S - 2] else S - 2
    if dp[T - 1, s] == neg:
        return None
    path = [0] * T
    for t in range(T - 1, -1, -1):
        path[t] = (s - 1) // 2 if s % 2 == 1 else -1
        s -= back[t, s]
    return path


def phoneme_scores(log_probs: np.ndarray, targets: list[int], path: list[int], blank: int) -> list[float]:
    """Goodness of pronunciation per target, scaled 0-100.

    For each frame aligned to a phoneme, GOP is the log posterior of the
    expected phoneme minus the best competing (non-blank) one: 0 when the
    model heard exactly that phoneme, negative when it heard another.
    """
    scores = []
    for k, target in enumerate(targets):
        frames = [t for t, p in enumerate(path) if p == k]
        if not frames:
            scores.append(0.0)
            continue
        margins = []
        for t in frames:
            row = log_probs[t].copy()
            expected = row[target]
            row[target] = -np.inf
            row[blank] = -np.inf
            margins.append(max(min(expected - row.max(), 0.0), GOP_FLOOR))
        scores.append(100.0 * math.exp(float(np.mean(margins))))
    return scores


def greedy_decode(log_probs: np.ndarray, blank: int) -> list[int]:
    """Free (unconstrained) CTC decoding: what the model heard."""
    out, prev = [], None
    for t in log_probs.argmax(axis=1):
        t = int(t)
        if t != prev and t != blank:
            out.append(t)
        prev = t
    return out


def align_phones(expected: list[str], heard: list[str]) -> tuple[list[str], int]:
    """Levenshtein-aligns heard phonemes to expected ones.

    Returns what was heard in place of each expected phoneme ("" when it
    was dropped) and the edit distance.
    """
    n, m = len(expected), len(heard)
    d = [[0] * (m + 1) for _ in range(n + 1)]
    for i in range(n + 1):
        d[i][0] = i
    for j in range(m + 1):
        d[0][j] = j
    for i in range(1, n + 1):
        for j in range(1, m + 1):
            d[i][j] = min(
                d[i - 1][j] + 1,
                d[i][j - 1] + 1,
                d[i - 1][j - 1] + (expected[i - 1] != heard[j - 1]),
            )
    out = [""] * n
    i, j = n, m
    while i > 0 and j > 0:
        if d[i][j] == d[i - 1][j - 1] + (expected[i - 1] != heard[j - 1]):
            out[i - 1] = heard[j - 1]
            i, j = i - 1, j - 1
        elif d[i][j] == d[i - 1][j] + 1:
            i -= 1
        else:
            j -= 1
    return out, d[n][m]


def npvi(durations: list[float]) -> float:
    """Normalised pairwise variability index of successive durations."""
    pairs = [(a, b) for a, b in zip(durations, durations[1:]) if a + b > 0]
    if not pairs:
        return 0.0
    return 100.0 * float(np.mean([abs(a - b) / ((a + b) / 2) for a, b in pairs]))


@dataclass
class PhoneSpan:
    phone: str
    start: float  # seconds
    end: float


def phone_spans(phones: list[str], path: list[int], offset: float, frame_s: float, word_end: float) -> list[PhoneSpan]:
    """Turns a CTC path into phoneme time spans. CTC puts each label on a
    frame or two and blanks between, so a phoneme is taken to last from its
    first frame until the next phoneme's first frame."""
    onsets = {}
    for t, k in enumerate(path):
        if k >= 0 and k not in onsets:
            onsets[k] = offset + t * frame_s
    spans = []
    for k, phone in enumerate(phones):
        if k not in onsets:
            continue
        nxt = next((onsets[j] for j in range(k + 1, len(phones)) if j in onsets), word_end)
        spans.append(PhoneSpan(phone, onsets[k], max(nxt, onsets[k] + frame_s)))
    return spans


def most_prominent(values: list[tuple[float, float, float]]) -> int:
    """Index of the most prominent syllable nucleus, given (duration,
    intensity, pitch) per vowel: the sum of each feature's z-score within
    the word, which is how listeners perceive English lexical stress."""
    arr = np.array(values, dtype=float)
    std = arr.std(axis=0)
    std[std == 0] = 1.0
    z = (arr - arr.mean(axis=0)) / std
    return int(z.sum(axis=1).argmax())


# --- Pitch and intensity -----------------------------------------------------
# A small YIN pitch tracker (de Cheveigné & Kawahara, 2002) in numpy, so the
# service needs no native audio library and runs the same on x86 and ARM.


def intensity_db(x: np.ndarray, sr: int, hop_s: float = 0.01, win_s: float = 0.03) -> tuple[np.ndarray, np.ndarray]:
    """Frame times (centres, seconds) and RMS level in dB."""
    hop, win = int(sr * hop_s), int(sr * win_s)
    if len(x) < win:
        return np.zeros(0), np.zeros(0)
    n = 1 + (len(x) - win) // hop
    idx = np.arange(win)[None, :] + hop * np.arange(n)[:, None]
    rms = np.sqrt(np.mean(x[idx] ** 2, axis=1) + 1e-12)
    return (np.arange(n) * hop + win / 2) / sr, 20 * np.log10(rms)


def pitch_track(
    x: np.ndarray,
    sr: int,
    hop_s: float = 0.01,
    fmin: float = 75.0,
    fmax: float = 500.0,
    threshold: float = 0.15,
    silence_db: float = 35.0,
    batch: int = 2000,
) -> tuple[np.ndarray, np.ndarray]:
    """Frame times and f0 in Hz, 0 where unvoiced.

    A frame is unvoiced when YIN finds no period below `threshold`, or when
    it is more than `silence_db` quieter than the loudest frame.
    """
    tau_min, tau_max = int(sr / fmax), int(sr / fmin) + 1
    w = tau_max  # integration window
    frame = w + tau_max
    hop = int(sr * hop_s)
    if len(x) < frame:
        return np.zeros(0), np.zeros(0)
    n_frames = 1 + (len(x) - frame) // hop
    nfft = 1 << int(np.ceil(np.log2(frame + w)))
    lags = np.arange(tau_max + 1)

    f0 = np.zeros(n_frames)
    energy = np.zeros(n_frames)
    for start in range(0, n_frames, batch):
        count = min(batch, n_frames - start)
        idx = np.arange(frame)[None, :] + hop * (start + np.arange(count))[:, None]
        frames = x[idx]
        # r(tau) = sum_j x_j x_{j+tau}, j < w, for every lag at once.
        spec = np.fft.rfft(frames, nfft) * np.conj(np.fft.rfft(frames[:, :w], nfft))
        r = np.fft.irfft(spec, nfft)[:, : tau_max + 1]
        cs = np.concatenate([np.zeros((count, 1)), np.cumsum(frames**2, axis=1)], axis=1)
        e0 = cs[:, w][:, None]
        e_tau = cs[:, lags + w] - cs[:, lags]
        d = np.maximum(e0 + e_tau - 2 * r, 0.0)
        # Cumulative mean normalised difference.
        cmnd = np.ones_like(d)
        running = np.cumsum(d[:, 1:], axis=1)
        cmnd[:, 1:] = d[:, 1:] * lags[1:] / np.maximum(running, 1e-12)
        energy[start : start + count] = e0[:, 0]

        for i in range(count):
            c = cmnd[i]
            below = np.nonzero(c[tau_min:] < threshold)[0]
            if not below.size:
                continue
            tau = tau_min + below[0]
            while tau + 1 <= tau_max and c[tau + 1] < c[tau]:
                tau += 1
            # Parabolic interpolation around the minimum.
            if tau_min < tau < tau_max:
                a, b, cc = c[tau - 1], c[tau], c[tau + 1]
                denom = a - 2 * b + cc
                shift = 0.5 * (a - cc) / denom if denom != 0 else 0.0
            else:
                shift = 0.0
            f0[start + i] = sr / (tau + shift)

    level = 10 * np.log10(energy + 1e-12)
    f0[level < level.max() - silence_db] = 0.0
    times = (np.arange(n_frames) * hop + frame / 2) / sr
    return times, f0


def pitch_stats(f0: np.ndarray) -> tuple[float, float]:
    """Spread (std) and 10th-90th percentile range of voiced pitch, in
    semitones around the speaker's median."""
    voiced = f0[f0 > 0]
    if voiced.size <= 20:
        return 0.0, 0.0
    st = 12 * np.log2(voiced / np.median(voiced))
    return float(st.std()), float(np.percentile(st, 90) - np.percentile(st, 10))

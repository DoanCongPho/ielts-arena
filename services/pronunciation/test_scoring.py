import math

import numpy as np

import scoring

BLANK = 0


def log_probs(frames: list[dict[int, float]], vocab: int = 5) -> np.ndarray:
    """Frames given as {token: probability}; the rest is spread evenly."""
    out = np.zeros((len(frames), vocab))
    for t, probs in enumerate(frames):
        rest = (1 - sum(probs.values())) / (vocab - len(probs))
        for c in range(vocab):
            out[t, c] = math.log(probs.get(c, rest))
    return out


def test_force_align_places_each_target():
    lp = log_probs([{0: 0.9}, {1: 0.9}, {0: 0.9}, {2: 0.9}, {2: 0.9}, {0: 0.9}])
    path = scoring.ctc_force_align(lp, [1, 2], BLANK)
    assert path == [-1, 0, -1, 1, 1, -1]


def test_force_align_needs_a_blank_between_repeats():
    lp = log_probs([{1: 0.9}, {1: 0.9}])
    assert scoring.ctc_force_align(lp, [1, 1], BLANK) is None
    lp = log_probs([{1: 0.9}, {0: 0.9}, {1: 0.9}])
    assert scoring.ctc_force_align(lp, [1, 1], BLANK) == [0, -1, 1]


def test_gop_is_high_when_heard_and_low_when_not():
    heard = log_probs([{0: 0.9}, {1: 0.9}, {0: 0.9}])
    swapped = log_probs([{0: 0.9}, {3: 0.8, 1: 0.1}, {0: 0.9}])
    path = [-1, 0, -1]
    good = scoring.phoneme_scores(heard, [1], path, BLANK)[0]
    bad = scoring.phoneme_scores(swapped, [1], path, BLANK)[0]
    assert good == 100.0
    assert bad < 20


def test_greedy_decode_collapses_repeats_and_blanks():
    lp = log_probs([{1: 0.9}, {1: 0.9}, {0: 0.9}, {1: 0.9}, {2: 0.9}])
    assert scoring.greedy_decode(lp, BLANK) == [1, 1, 2]


def test_align_phones():
    heard, edits = scoring.align_phones(["θ", "ɪ", "ŋ", "k"], ["t", "ɪ", "ŋ"])
    assert heard == ["t", "ɪ", "ŋ", ""]
    assert edits == 2


def test_npvi():
    assert scoring.npvi([0.1, 0.1, 0.1]) == 0
    assert round(scoring.npvi([0.2, 0.1]), 1) == 66.7


def test_most_prominent():
    # duration, intensity (dB), pitch (Hz): the second vowel stands out.
    assert scoring.most_prominent([(0.06, 60, 180), (0.14, 68, 220), (0.07, 61, 175)]) == 1


SR = 16_000


def voice(f0_hz, seconds=1.0):
    """A voiced sound: a fundamental plus two harmonics, like a vowel."""
    t = np.arange(int(SR * seconds)) / SR
    f = np.broadcast_to(np.asarray(f0_hz, dtype=float), t.shape)
    phase = 2 * np.pi * np.cumsum(f) / SR
    return 0.5 * np.sin(phase) + 0.25 * np.sin(2 * phase) + 0.12 * np.sin(3 * phase)


def test_pitch_track_finds_a_steady_pitch():
    _, f0 = scoring.pitch_track(voice(200.0), SR)
    voiced = f0[f0 > 0]
    assert voiced.size > 50
    assert abs(np.median(voiced) - 200) < 2


def test_pitch_track_follows_a_glide_and_ignores_silence():
    t = np.arange(SR) / SR
    glide = voice(150 + 100 * t)  # 150 Hz rising to 250 Hz
    audio = np.concatenate([glide, np.zeros(SR // 2)])
    times, f0 = scoring.pitch_track(audio, SR)
    assert (f0[times > 1.1] == 0).all()  # the silence is unvoiced
    early, late = f0[(times > 0.1) & (times < 0.2)], f0[(times > 0.8) & (times < 0.9)]
    assert 150 < np.median(early) < 185 and 225 < np.median(late) < 255


def test_pitch_stats_separates_flat_from_lively_speech():
    _, flat = scoring.pitch_track(voice(180.0, 2.0), SR)
    t = np.arange(2 * SR) / SR
    _, lively = scoring.pitch_track(voice(180 * 2 ** (4 * np.sin(2 * np.pi * 0.8 * t) / 12), 2.0), SR)
    flat_std, _ = scoring.pitch_stats(flat)
    lively_std, lively_range = scoring.pitch_stats(lively)
    assert flat_std < 0.3
    assert lively_std > 1.5 and lively_range > 4


def test_intensity_db():
    times, db = scoring.intensity_db(np.concatenate([0.5 * np.ones(SR // 2), 0.05 * np.ones(SR // 2)]), SR)
    assert db[times < 0.4].mean() - db[times > 0.6].mean() > 19  # 10x amplitude = 20 dB

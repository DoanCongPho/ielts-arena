"""IELTS Arena pronunciation service.

Takes one recorded answer, its transcript and per-word timings from speech
recognition, and measures:

- intelligibility: goodness of pronunciation (GOP) of every phoneme, by CTC
  forced alignment with a wav2vec2 phoneme recogniser, against both British
  and American reference pronunciations (whichever fits each word better,
  so accent alone costs nothing);
- phoneme error rate: what the recogniser heard without constraints,
  compared with what was expected;
- prosody: pitch variation (intonation), word stress placement and vowel
  rhythm (nPVI) — the "features" the upper IELTS Pronunciation bands ask for.
  Pitch is tracked with YIN in numpy (scoring.py), so there is no native
  audio library and the image builds the same on x86 and ARM.

It does not turn these into a band; the IELTS Arena API does that against
the descriptors.
"""

from __future__ import annotations

import json
import os
import subprocess
import tempfile
from dataclasses import dataclass

import platform

import numpy as np
import torch
from fastapi import Depends, FastAPI, File, Form, Header, HTTPException, UploadFile
from phonemizer.backend import EspeakBackend
from phonemizer.separator import Separator
from transformers import AutoModelForCTC, AutoProcessor

import scoring

MODEL_ID = os.environ.get("MODEL_ID", "facebook/wav2vec2-xlsr-53-espeak-cv-ft")
TOKEN = os.environ.get("PRON_TOKEN", "")
SAMPLE_RATE = 16_000
FRAME_S = 0.02  # wav2vec2 emits one frame per 20 ms
# Model input is cut into chunks this long at word boundaries: self-attention
# memory grows with the square of the length.
CHUNK_S = 12.0
# Seconds added around a word's recognised timing for alignment, which
# absorbs recogniser timing error.
WORD_PAD_S = 0.15
# A word scoring below this is reported as not clearly intelligible.
INTELLIGIBLE_AT = 60.0
VARIANTS = ("en-us", "en-gb")
MAX_AUDIO_BYTES = 25 * 1024 * 1024

torch.set_num_threads(int(os.environ.get("TORCH_THREADS", "2")))
processor = AutoProcessor.from_pretrained(MODEL_ID)
model = AutoModelForCTC.from_pretrained(MODEL_ID).eval()
if os.environ.get("QUANTIZE", "1") == "1":
    # int8 linear layers: about twice as fast on CPU for a negligible
    # change in phoneme posteriors. ARM CPUs (e.g. an Oracle Ampere VM)
    # need the qnnpack backend for it.
    if platform.machine() in ("aarch64", "arm64") and "qnnpack" in torch.backends.quantized.supported_engines:
        torch.backends.quantized.engine = "qnnpack"
    model = torch.quantization.quantize_dynamic(model, {torch.nn.Linear}, dtype=torch.qint8)
tokenizer = processor.tokenizer
BLANK = tokenizer.pad_token_id
UNK = tokenizer.unk_token_id
ID_TO_PHONE = {i: p for p, i in tokenizer.get_vocab().items()}

# Phones and words need different separators; words are merged back into
# one phone list anyway (a number like "2019" can expand to several words).
SEP = Separator(phone=" ", word="_", syllable="")
backends = {v: EspeakBackend(v, with_stress=True, language_switch="remove-flags") for v in VARIANTS}

app = FastAPI(title="IELTS Arena pronunciation")


def require_token(authorization: str = Header(default="")) -> None:
    if TOKEN and authorization != f"Bearer {TOKEN}":
        raise HTTPException(status_code=401, detail="invalid token")


@app.get("/health")
def health() -> dict:
    return {"ok": True, "model": MODEL_ID}


@dataclass
class Reference:
    phones: list[str]
    ids: list[int]
    stressed: int  # index into the word's vowels of the primary stress, -1 if unknown


def to_ids(phones: list[str]) -> tuple[list[str], list[int]]:
    """Maps espeak phonemes to model tokens, splitting any the model's
    vocabulary lacks into characters and dropping what's still unknown."""
    kept, ids = [], []
    for p in phones:
        i = tokenizer.convert_tokens_to_ids(p)
        if i != UNK:
            kept.append(p)
            ids.append(i)
            continue
        for ch in p:
            j = tokenizer.convert_tokens_to_ids(ch)
            if j != UNK:
                kept.append(ch)
                ids.append(j)
    return kept, ids


def phonemize_in_context(words: list[str], variant: str) -> list[str]:
    """One phone string per word, phonemized as a whole utterance so
    function words get their weak forms ("a" as /ə/, not the letter name
    /eɪ/). Falls back to word by word when the utterance doesn't split
    back into the same number of words (a number or symbol read as
    several words)."""
    joined = backends[variant].phonemize([" ".join(words)], separator=SEP, strip=True)[0]
    per_word = [w.strip() for w in joined.split("_")]
    per_word = [w for w in per_word if w]
    if len(per_word) == len(words):
        return per_word
    return backends[variant].phonemize(words, separator=SEP, strip=True)


# Weak forms of English function words. In connected speech these are the
# normal pronunciations ("a" is /ə/, "to" is /tə/), and IELTS descriptors
# reward using them, so they must never count as mispronunciations. espeak
# can't be relied on for them: phonemizing a sentence merges words ("I am"
# becomes one), so its per-word output falls back to citation forms.
WEAK_FORMS = {
    "a": ["ə", "ɐ"], "an": ["ə n", "ɐ n"], "the": ["ð ə", "ð ɪ", "ð i"],
    "to": ["t ə", "t ʊ"], "of": ["ə v", "ə"], "and": ["ə n d", "ə n", "n"],
    "for": ["f ɚ", "f ə"], "at": ["ə t"], "as": ["ə z"], "was": ["w ə z"],
    "were": ["w ɚ", "w ə"], "can": ["k ə n", "k n"], "from": ["f ɹ ə m"],
    "that": ["ð ə t"], "some": ["s ə m"], "are": ["ɚ", "ə"], "you": ["j ə"],
    "your": ["j ɚ", "j ə"], "but": ["b ə t"], "them": ["ð ə m", "ə m"],
    "have": ["h ə v", "ə v"], "has": ["h ə z", "ə z"], "had": ["h ə d", "ə d"],
    "would": ["w ə d", "ə d"], "do": ["d ə"], "does": ["d ə z"], "am": ["ə m"],
    "or": ["ɚ"], "than": ["ð ə n"], "there": ["ð ɚ"], "his": ["ɪ z"], "her": ["h ɚ", "ɚ"],
}


def weak_references(word: str) -> list[Reference]:
    out = []
    for form in WEAK_FORMS.get(word.lower().strip(".,!?;:\"'"), []):
        phones, ids = to_ids(form.split())
        if ids:
            out.append(Reference(phones, ids, -1))
    return out


def references(words: list[str], variant: str, in_context: bool) -> list[Reference]:
    out = []
    texts = phonemize_in_context(words, variant) if in_context else backends[variant].phonemize(words, separator=SEP, strip=True)
    for text in texts:
        phones, stressed_token = [], -1
        for tok in text.replace("_", " ").split():
            if "ˈ" in tok and stressed_token < 0:
                stressed_token = len(phones)
            tok = tok.replace("ˈ", "").replace("ˌ", "")
            if tok:
                phones.append(tok)
        phones, ids = to_ids(phones)
        # The stress mark sits at the stressed syllable's start; its
        # nucleus is the first vowel from there.
        stressed = -1
        if stressed_token >= 0:
            vowel_idx = [k for k, p in enumerate(phones) if scoring.is_vowel(p)]
            after = [n for n, k in enumerate(vowel_idx) if k >= stressed_token]
            stressed = after[0] if after else -1
        out.append(Reference(phones, ids, stressed))
    return out


def decode_audio(data: bytes, filename: str) -> np.ndarray:
    """Any browser recording format to 16 kHz mono float32, via ffmpeg.

    The input goes through a temporary file, not a pipe: an MP4/M4A (what
    Safari records) usually keeps its index at the end of the file, which
    ffmpeg can only reach by seeking.
    """
    suffix = os.path.splitext(filename or "")[1] or ".bin"
    with tempfile.NamedTemporaryFile(suffix=suffix) as f:
        f.write(data)
        f.flush()
        proc = subprocess.run(
            ["ffmpeg", "-nostdin", "-loglevel", "error", "-i", f.name,
             "-ac", "1", "-ar", str(SAMPLE_RATE), "-f", "f32le", "pipe:1"],
            capture_output=True, timeout=120,
        )
    if proc.returncode != 0:
        raise HTTPException(status_code=400, detail="could not decode audio: " + proc.stderr.decode()[:200])
    pcm = np.frombuffer(proc.stdout, dtype=np.float32)
    # Scoring silence would report every word as mispronounced; failing
    # instead lets the caller retry or fall back to an estimate.
    if len(pcm) < SAMPLE_RATE // 2:
        raise HTTPException(status_code=400, detail=f"decoded audio is only {len(pcm) / SAMPLE_RATE:.2f} s long")
    return pcm


def chunk_log_probs(audio: np.ndarray, words: list[dict]) -> list[tuple[float, np.ndarray]]:
    """Runs the model over chunks of about CHUNK_S seconds, cut between
    words. Returns (chunk start in seconds, log-probabilities per frame)."""
    cuts, start = [], 0.0
    for w in words:
        if w["end"] - start > CHUNK_S:
            cuts.append((start, w["start"]))
            start = w["start"]
    cuts.append((start, len(audio) / SAMPLE_RATE))
    out = []
    for a, b in cuts:
        a = max(a - 0.2, 0.0)
        seg = audio[int(a * SAMPLE_RATE): int(b * SAMPLE_RATE) + int(0.2 * SAMPLE_RATE)]
        if len(seg) < SAMPLE_RATE // 10:
            continue
        inputs = processor(seg, sampling_rate=SAMPLE_RATE, return_tensors="pt")
        with torch.inference_mode():
            logits = model(inputs.input_values).logits[0]
        out.append((a, torch.log_softmax(logits, dim=-1).numpy()))
    return out


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


@app.post("/assess", dependencies=[Depends(require_token)])
async def assess(
    audio: UploadFile = File(...),
    transcript: str = Form(""),
    words: str = Form(...),
) -> dict:
    data = await audio.read()
    if len(data) > MAX_AUDIO_BYTES:
        raise HTTPException(status_code=413, detail="audio too large")
    try:
        timed = json.loads(words)
        texts = [str(w.get("word", "")).strip() for w in timed]
    except (ValueError, AttributeError):
        raise HTTPException(status_code=400, detail="words must be a JSON list of {word, start, end}")
    if not any(texts):
        raise HTTPException(status_code=400, detail="no words to assess")

    pcm = decode_audio(data, audio.filename)
    chunks = chunk_log_probs(pcm, [w for w, t in zip(timed, texts) if t])
    # Each word is scored against every reference and keeps the best: two
    # accents, and each with the connected-speech form ("a" as /ə/) and the
    # citation form ("a" as /eɪ/), since both are correct English.
    refs = {
        (v, ctx): references(texts, v, ctx)
        for v in VARIANTS
        for ctx in (True, False)
    }

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

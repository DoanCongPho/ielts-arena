"""IELTS Arena pronunciation service — the HTTP layer.

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

Modules: audio (decoding), model (the recogniser), phonemes (reference
pronunciations), assess (scoring a recording), scoring (the maths).
"""

from __future__ import annotations

import json
import os

from fastapi import Depends, FastAPI, File, Form, Header, HTTPException, UploadFile

from assess import assess_recording
from audio import AudioError, decode_audio
from model import MODEL_ID

TOKEN = os.environ.get("PRON_TOKEN", "")
MAX_AUDIO_BYTES = 25 * 1024 * 1024

app = FastAPI(title="IELTS Arena pronunciation")


def require_token(authorization: str = Header(default="")) -> None:
    if TOKEN and authorization != f"Bearer {TOKEN}":
        raise HTTPException(status_code=401, detail="invalid token")


@app.get("/health")
def health() -> dict:
    return {"ok": True, "model": MODEL_ID}


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
    try:
        pcm = decode_audio(data, audio.filename)
    except AudioError as e:
        raise HTTPException(status_code=400, detail=str(e))
    return assess_recording(pcm, timed, texts)

"""The wav2vec2 phoneme recogniser: loading it, and running it over audio."""

from __future__ import annotations

import os
import platform

import numpy as np
import torch
from transformers import AutoModelForCTC, AutoProcessor

from audio import SAMPLE_RATE

MODEL_ID = os.environ.get("MODEL_ID", "facebook/wav2vec2-xlsr-53-espeak-cv-ft")
FRAME_S = 0.02  # wav2vec2 emits one frame per 20 ms
# Model input is cut into chunks this long at word boundaries: self-attention
# memory grows with the square of the length.
CHUNK_S = 12.0

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

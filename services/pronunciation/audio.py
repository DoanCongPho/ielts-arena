"""Decoding recordings for the model."""

from __future__ import annotations

import os
import subprocess
import tempfile

import numpy as np

SAMPLE_RATE = 16_000


class AudioError(ValueError):
    """The recording can't be scored (undecodable or nearly silent)."""


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
        raise AudioError("could not decode audio: " + proc.stderr.decode()[:200])
    pcm = np.frombuffer(proc.stdout, dtype=np.float32)
    # Scoring silence would report every word as mispronounced; failing
    # instead lets the caller retry or fall back to an estimate.
    if len(pcm) < SAMPLE_RATE // 2:
        raise AudioError(f"decoded audio is only {len(pcm) / SAMPLE_RATE:.2f} s long")
    return pcm

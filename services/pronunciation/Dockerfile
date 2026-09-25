# CPU only, x86 or ARM. The model is downloaded at build time so a restart
# loads it from disk instead of the network.
FROM python:3.11-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ffmpeg espeak-ng \
    && rm -rf /var/lib/apt/lists/*

# Run as an unprivileged user.
RUN useradd -m -u 1000 app
USER app
ENV HOME=/home/app \
    HF_HOME=/home/app/.cache/huggingface \
    PATH=/home/app/.local/bin:$PATH \
    PYTHONUNBUFFERED=1
WORKDIR /home/app/service

COPY --chown=app requirements.txt .
RUN pip install --no-cache-dir --user -r requirements.txt

ARG MODEL_ID=facebook/wav2vec2-xlsr-53-espeak-cv-ft
ENV MODEL_ID=$MODEL_ID
RUN python -c "import os; from transformers import AutoModelForCTC, AutoProcessor; m=os.environ['MODEL_ID']; AutoProcessor.from_pretrained(m); AutoModelForCTC.from_pretrained(m)"

COPY --chown=app *.py ./

EXPOSE 7860
# One worker: the model takes about 1.3 GB and requests are CPU-bound, so a
# second worker would only compete for the same two cores.
CMD ["uvicorn", "app:app", "--host", "0.0.0.0", "--port", "7860", "--workers", "1"]

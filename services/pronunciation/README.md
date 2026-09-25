# IELTS Arena pronunciation service

Measures pronunciation in one recorded IELTS Speaking answer, for the IELTS
Arena API to rate against the Pronunciation band descriptors. It's a plain
Docker container (CPU only, about 2 GB of RAM), deployed for free on an
Oracle Cloud Always Free VM.

## What it measures

| Measure | How | Descriptor feature |
|---|---|---|
| `intelligibility` | Share of words whose mean phoneme GOP is ≥ 60 | "understood throughout without much effort" |
| `gop_mean`, per-word `score` | Goodness of pronunciation: CTC forced alignment with `facebook/wav2vec2-xlsr-53-espeak-cv-ft`, then the log posterior of the expected phoneme minus the best competitor, scaled 0–100 | "individual words or phonemes mispronounced" |
| `per` | Phoneme error rate: unconstrained decoding against the expected phonemes | same |
| `accent_variant` | Each word is scored against **en-us and en-gb** references (espeak-ng), each in its connected-speech form ("a" as /ə/, phonemized in context) and its citation form ("a" as /eɪ/), and keeps the best, so neither accent nor a careful strong form costs anything | "accent has no effect on intelligibility" |
| `prosody.pitch_std_st`, `pitch_range_st` | YIN pitch track (numpy, `scoring.py`), in semitones around the speaker's median | intonation |
| `prosody.stress_match_rate` | In words of two or more syllables, whether the most prominent vowel (duration + intensity + pitch) is the dictionary's primary stress | word stress |
| `prosody.npvi_vowel` | Normalised pairwise variability of vowel durations | rhythm / stress-timing |

Thresholds are starting points. Calibrate them on examiner-rated samples
before trusting bands built on them.

## API

`POST /assess` (multipart), with `Authorization: Bearer $PRON_TOKEN` when
`PRON_TOKEN` is set:

- `audio`: the recording (webm, ogg, mp4, m4a, wav — anything ffmpeg reads)
- `words`: JSON list of `{word, start, end}` from speech recognition
- `transcript`: the full text (informational)

It returns one `words` entry per input word, in order.

`GET /health` answers once the model is loaded.

## Deploy on Oracle Cloud Always Free

The Always Free tier includes Ampere A1 (ARM) capacity of up to 4 OCPUs and
24 GB of RAM, which is far more than this needs. Sign-up asks for a card to
verify identity; Always Free resources are not charged.

1. **Create the VM.** In the console, go to Compute → Instances → Create.
   - Image: **Canonical Ubuntu 24.04** or **Oracle Linux 9** (the script
     handles both; the SSH user is `ubuntu` or `opc`).
   - Shape: **Ampere → VM.Standard.A1.Flex**, 2–4 OCPUs and 12–24 GB.
   - Upload your SSH public key.

   If the console says "Out of capacity", try another availability domain,
   or try again later.
2. **Open ports 80 and 443 in the VCN.** On the instance page, open the
   subnet, then its Security List, and add two ingress rules for source
   `0.0.0.0/0`, TCP, destination ports `80` and `443`.
3. **Copy this directory to the VM and run the setup script:**

   ```sh
   scp -r services/pronunciation <user>@<VM_IP>:~/
   ssh <user>@<VM_IP> 'bash ~/pronunciation/deploy/oracle/setup.sh'
   ```

   The script:
   - installs Docker
   - opens 80 and 443 in the VM's own firewall
   - uses `<VM_IP>.sslip.io` as the hostname (a free wildcard DNS that
     points at your IP), unless you run it with
     `PRON_DOMAIN=pron.example.com` for a domain you own
   - generates a token
   - starts the service behind Caddy, which gets a Let's Encrypt
     certificate on its own

   The first build downloads the model and takes several minutes.
4. **Point the API at it.** Set the two values the script prints on the
   IELTS Arena API:

   ```
   PRON_SERVICE_URL=https://<VM_IP with dashes>.sslip.io
   PRON_SERVICE_TOKEN=<token>
   ```

To update the service after changing the code, copy the directory again and
run `sudo docker compose up -d --build` in `deploy/oracle`. Logs:
`sudo docker compose logs -f pron`.

If the service is ever unreachable, the API retries the grade with backoff.
On the last attempt it estimates pronunciation instead, so an outage delays
a grade but never fails it.

## Run locally

```sh
docker build -t ielts-pron . && docker run -p 7860:7860 ielts-pron
python -m pytest test_scoring.py   # the maths only; needs numpy and pytest
```

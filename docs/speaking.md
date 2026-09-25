# Speaking

Speaking is graded the way an IELTS examiner rates a test, using only free
hosting and the OpenAI API.

## Where the code lives

`internal/feature/ielts_test` owns what every skill shares: tests,
submissions, the grading queue. For speaking that is the content model
(`speaking_content.go`) and submission validation (`speaking_submission.go`).
It reaches grading only through the `SpeakingGrader` interface and never
imports the packages below.

`internal/feature/speaking/`:

| Package | What it does |
|---|---|
| `grading` | The examiner engine. `grader.go` orchestrates it. Measurements are split by dimension: `fluency.go`, `coherence.go`, `language.go`, and `pronunciation.go` (the service's results). `judge.go` and `descriptors.go` are the LLM judges; `corrections.go` handles grammar and word-choice fixes; `caps.go` holds the guardrails; `calibrate.go` is the calibration command. |
| `examiner` | The examiner's script (`script.go`) and voice, synthesised once per line and cached (`audio.go`). |
| `api` | `/api/speaking`: `uploads.go`, `script.go`, `custom.go` (custom tests), `generate.go` ("fill the rest"), `recordings.go` and `handler.go`. |
| `speakingtest` | Test fixtures only. |

Golden files in `grading/testdata/golden` and `examiner/testdata/golden` pin
the stored score JSON, the judge prompts, the evidence and the script. A change
that alters them fails the tests. If the change is intended, regenerate them
with `UPDATE_GOLDEN=1 go test ./internal/feature/speaking/...`.

## How a test runs

1. `GET /api/speaking/tests/{id}/script` returns the examiner's script. It
   uses the standard examiner wording for the greeting, the Part 2 hand-over,
   the rounding-off questions, the Part 3 lead-in and the close.
2. The browser plays each line. Once the examiner stops speaking, it records
   the answer (WebM/Opus, or MP4/AAC on Safari). There are no re-recordings,
   and the next question is not shown ahead of time.
   - Part 2 shows the cue card, with 1 minute to prepare (notes allowed) and
     the talk stopped at 2 minutes.
   - Part 1 and 3 limits are soft, as in the real test.
3. Each clip is uploaded as soon as it's recorded, through links from
   `POST /api/speaking/uploads`. The links point straight at the bucket, or at
   signed `/assets/media/` paths when no bucket is configured. The API never
   handles audio bytes.
4. `POST /api/submissions` with
   `{answers: [{question_id, audio_key, duration_sec}]}` queues the grade. A
   submission may only name recordings under `speaking/<its own user id>/`.

## How it's graded

| Examiner practice | Here |
|---|---|
| One rating for the whole test, not per answer | One grade per submission. Each criterion judge sees every part, labelled. |
| Four criteria, equal weight, whole bands | `Fluency and Coherence`, `Lexical Resource`, `Grammatical Range and Accuracy`, `Pronunciation`, each an int from 1 to 9 |
| Best fit to the band descriptors | Each judge gets that criterion's descriptors (`grading/descriptors.go`). For each band it checks the key features as met, partly met or not met, with evidence, and only then picks a band. |
| Overall = mean, .25 rounds up to .5 and .75 to the next band | `ieltsOverall` in Go. Writing uses it now too. |
| Accent is not penalised | Phonemes are scored against both British and American references, and each word keeps the better one |
| Content-planning pauses are fine at band 8+; word searches are not | Each long pause goes to the judge with its surrounding words, marked as mid-clause or at a boundary |
| Rehearsed or off-topic answers don't count | The judges leave them out and list them. The review shows which ones. |

The pipeline (`grading/grader.go`):

1. **Transcribe** every answer with `whisper-1` (`verbose_json`, word and
   segment timestamps). A prompt full of fillers keeps the "um"s and restarts
   that fluency is rated on.
2. **Measure** (`grading/evidence.go` and a file per dimension, pure Go):
   - speech rate and articulation rate
   - pauses and long pauses per minute, with their context
   - mean length of run
   - fillers
   - repetitions and restarts
   - Part 2 length
   - discourse markers
   - lexical variety (MATTR)
   - sentence complexity
3. **Pronunciation.** A sample of the test (Part 2 first, up to 3 minutes)
   goes to the pronunciation service, which returns:
   - intelligibility, GOP and phoneme error rate
   - pitch variation
   - word-stress placement
   - vowel rhythm (nPVI)
4. **Judge.** Four parallel calls to `SPEAKING_MODEL`, one per criterion.
5. **Guardrails** (`grading/caps.go`). Measurements cap what a judge may
   award:
   - a Part 2 under 45 s caps FC at 5
   - intelligibility under 75% caps P at 5
   - flat intonation caps P at 6
   - under 40 words caps everything at 3

   The judged band, the cap and its reason are all stored and shown.
6. **Fallback.** A failed pronunciation call is retried with the job's
   backoff. On the last attempt, pronunciation is estimated from speech
   recognition confidence, marked `estimated`, and capped at 7.

Single-part practice is graded the same way, but reported as an
**indicative** band, because IELTS never scores one part on its own. Part 1
alone caps FC at 7.

Custom tests earn no XP, because anyone could write easy cards to farm it.

## Examiner voice

The examiner is `gpt-4o-mini-tts` with one fixed voice and a fixed
"neutral British examiner" instruction.

- **Storage:** each line is stored at `examiner/<hash of model, voice,
  instructions and text>.mp3`, so every distinct line is made once and
  shared by every test that uses it.
- **When lines are made:** on demand. Lines are queued when a test is
  created, and again whenever a script is served with lines still missing.
- **Fallback:** the runner uses the browser's `speechSynthesis` (en-GB) for
  any line that isn't recorded yet, so a test never waits for audio.

## Custom cards

`/practice/speaking/custom/new` builds a test.

- **Mode:** a full test (Parts 1–3) or one part.
- **Sources:** each part comes from an official test or is written by the
  user.
- **Fill the rest:** writes Part 1 and/or Part 3 around the user's cue card,
  in the style of the official books, for the user to edit before saving.
- **Moderation:** the user's text goes through OpenAI moderation (free).
- **Style check:** a quick LLM check returns non-blocking advice when a card
  doesn't read like the real test.
- **Visibility:** custom tests have `owner_id` set (migration 000010). Only
  their owner can see them, and they can't be edited or deleted once
  attempted.

API:

| Method and path | Purpose |
|---|---|
| `GET /api/speaking/custom` | the user's tests |
| `POST /api/speaking/custom` | `{title, part1?, part2?, part3?}`, each part `{bank_test_id}` or `{custom}` |
| `PUT` or `DELETE /api/speaking/custom/{id}` | edit or delete one |
| `POST /api/speaking/custom/generate` | `{part2, part1: bool, part3: bool}`, returns drafted parts |
| `GET /api/speaking/submissions/{id}/recordings` | playback links for the user's own answers |

## Deploying on free services

- **Pronunciation service:** deploy `services/pronunciation` on an Oracle
  Cloud Always Free ARM VM with `deploy/oracle/setup.sh`. The script sets up
  Docker and HTTPS through Caddy and sslip.io. See the service's README. Set
  `PRON_SERVICE_URL` and `PRON_SERVICE_TOKEN` on the API. If the service is
  down, grades are delayed, never failed. (Hugging Face Docker Spaces now
  need a paid plan, so they're no longer an option.)
- **Bucket:** the existing assets bucket also holds `speaking/` and
  `examiner/` objects. Browsers PUT recordings to it directly, so add a CORS
  rule allowing `PUT` and `GET` from the frontend's origin. For example, on
  Backblaze B2 with `b2 bucket update --cors-rules`, allow origin
  `https://your-frontend`, operations `s3_put` and `s3_get`, and header
  `content-type`. Without `S3_BUCKET`, everything is kept in `MEDIA_DIR`
  (`/app/data/media` in compose).
- **Costs:** OpenAI only.
  - Whisper: about $0.006 per minute, so about $0.08 per full test.
  - Judging: four calls, cents.
  - Examiner lines: synthesised once, ever.

## Calibration

The guardrail thresholds are starting points. Before bands are presented as
more than estimates, check the pipeline against examiner-rated samples:

```sh
go run ./cmd/api calibrate-speaking samples/manifest.json
```

The manifest lists samples with published examiner bands, such as
British Council, IDP or Cambridge samples with examiner commentary, covering
bands 4–8:

```json
[{
  "id": "bc-band6-a",
  "source": "where the recording and its bands come from",
  "bands": { "overall": 6.0, "Fluency and Coherence": 6, "Pronunciation": 6 },
  "answers": [
    { "part": 1, "question": "Do you work or study?", "audio": "bc-band6-a/p1-1.mp3" },
    { "part": 2, "question": "Describe a book you enjoyed reading.", "audio": "bc-band6-a/p2.mp3" },
    { "part": 3, "question": "Why do people read less today?", "audio": "bc-band6-a/p3-1.mp3" }
  ]
}]
```

The command runs the real pipeline, so it needs `OPENAI_API_KEY` and ideally
the pronunciation service. It prints each sample's examiner band next to
ours, and exits non-zero until both targets are met:

- the overall band is within ±0.5 of the examiner's for at least 80% of
  samples
- each criterion matches exactly for at least 70% of samples

Tune `grading/caps.go` (and the pronunciation service's `INTELLIGIBLE_AT`)
against it. Sample recordings are copyrighted, so keep them out of the repo
the way `tests_bank/` is.

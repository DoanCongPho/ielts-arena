# IELTS Arena

IELTS Arena is a web app for IELTS practice, starting with Writing, built on a pure-Go (no framework) backend with a separate frontend that talks to it over REST and WebSocket. Learners pick a Task 1 or Task 2 prompt, write within a timer, and get automated band scores across the four IELTS criteria plus corrections and improvement suggestions; a background worker keeps the question bank current by fetching and normalizing tests from configured sources. On top of solo practice, a real-time Dual Match mode pits multiple users against the same prompt where the highest overall band wins, with a future 4v4 team mode that assigns each player a skill role and combines their results into a team overall. Progress is gamified through levels (from practice XP) and competitive rank (ELO-style), unlocking badges and titles so users feel their growth is rewarded.

## Running the project

```bash
cp .env.example .env     # fill in DB_USER / DB_PASSWORD / DB_NAME / SECRET_KEY / OPENAI_API_KEY
docker compose up --build
```

Open http://localhost:8081 — that is the only port you need. nginx serves the
SPA and proxies `/api` and `/assets` to the API, so every request is
same-origin and there is no CORS anywhere in the stack.

Running the two halves separately during development:

```bash
make migrate-up && make run     # API on :8080
cd frontend && npm install && npm run dev
```

Vite's dev server proxies `/api` to `localhost:8080`, exactly as nginx does in
production, so application code never knows the API's host.

## Architecture

- **Backend** — plain Go (only `gorilla/mux` for routing), `database/sql` with
  no ORM. `internal/platform` is shared infrastructure, `internal/feature` is
  business logic. Within a feature, `Handler` owns the HTTP side and `Service`
  owns the business rules — `Service` never refers to a router, a request or a
  status code.
- **Background grading** — `POST /api/submissions` returns `202` with a
  submission in the `pending` state; a worker pool
  (`internal/feature/ielts_test/worker.go`) claims work from the queue, grades
  it, then settles it as `graded` / `failed`. Clients poll
  `GET /api/submissions/{id}`. Transient failures are retried with increasing
  backoff; if a worker dies mid-grade its lease expires and another worker
  picks the job up.
- **Speaking** — graded the way an IELTS examiner rates: the whole 3-part
  test, four criteria in whole bands by best fit to the band descriptors, and
  IELTS rounding for the overall. OpenAI transcribes (Whisper, with word
  timings), judges each criterion and voices the examiner; a self-hosted
  wav2vec2 service on an Oracle Cloud Always Free VM (`services/pronunciation`)
  measures phonemes, intonation, stress and rhythm. Users can also write their
  own cue cards and practise a full test or a single part. See
  [docs/speaking.md](docs/speaking.md).
- **Frontend** — React + Vite, with no state manager: `lib/api.js` is the only
  API layer and pages fetch for themselves.

## Status

**Shipped:** JWT sign-up / sign-in (access + refresh), practice across the four
skills with Reading/Listening auto-graded over 18 question types, Writing
graded by an LLM against the four IELTS criteria with corrections and a model
answer, server-side answer redaction, submission history, a level/XP system
with 100 avatar frames, and an admin page for authoring tests.

**Built, needs calibration:** Speaking — a spoken exam with an examiner voice,
graded against the band descriptors, plus user-written cue cards and part
practice. Until the pipeline agrees with examiner-rated samples
(`api calibrate-speaking`, see [docs/speaking.md](docs/speaking.md)), its bands
should be read as estimates.

**Planned:** real-time Dual Match over WebSocket, 4v4 team mode, ELO-style
ranking, badges and titles, and a worker that collects tests from external
sources on its own.

# PRD — IELTS Arena (v0.2)

Product requirements document. Backend: pure Go (no framework), REST API (`net/http` + `gorilla/mux`). Frontend: separate React (Vite) app talking to the backend over REST. This document defines *what* and *why*, not *how* to implement it.

> Supersedes `PRD_IELTS_App.md.pdf` (Draft v0.1, Writing-only). This revision reflects the app as built: **Reading, Writing, and Listening are implemented and live; Speaking is scoped but not yet built.**

## 1. Overview

**Name:** IELTS Arena.

**Problem:** IELTS learners lack (a) fast, specific feedback across all four skills, (b) a steadily refreshed bank of practice tests, (c) motivation to practice consistently.

**Solution:** A web app for IELTS practice covering Reading, Writing, and Listening today, with automated grading, progress tracking, and a gamification layer (levels, XP, unlockable avatar frames) to drive motivation. Speaking, competitive real-time matches, and team modes are planned extensions of the same grading + gamification core.

**Scoping principle:** Every skill is a "skill module" — a `tests.skill` value with its own content shape, grading logic, and attempt UI — plugged into one shared framework for submissions, scoring, XP, and (in the future) competitive play.

## 2. Current state vs. roadmap

### Shipped

- **Auth:** email/password registration, login, JWT access + refresh tokens (`/auth/register`, `/auth/login`, `/auth/refresh`).
- **Writing:** Task 1 / Task 2 prompts (optionally with a chart/image), free-text submission, LLM-graded across the 4 IELTS criteria with band scores, per-criterion feedback, corrections, and a model answer.
- **Reading:** passage-based tests with 14 supported question types, auto-graded (exact-match / accepted-answers / multi-select-set rules), band derived from a raw-score-to-band table.
- **Listening:** section-based tests sharing the reading question-type engine, plus a shared audio file per test and per-question timestamp hints for review.
- **Submissions & history:** every attempt is persisted with its graded result; users can list past submissions and revisit a graded submission's score/detail.
- **Gamification (partial):** lifetime XP and a derived level (1–100, compounding XP curve) awarded once per user per test on first graded attempt; 100 unlockable avatar frames (frame *N* unlocks at level *N*), equippable via profile settings.
- **Admin authoring:** admin-only `POST /api/tests` for creating tests of any skill, with validation (contiguous question ordering, gap/question-count matching, required fields per question type, etc. — see [data_schema.md](data_schema.md)).

### Not yet built (roadmap)

| Priority | Feature | Notes |
|---|---|---|
| Next | **Speaking** | Needs audio capture/upload and either ASR + LLM grading or a different grading pipeline. Content shape already reserves a `part` field (1/2/3) in `content_data`. |
| Next | **Test source worker** | Background job to crawl/import new Writing/Reading/Listening prompts from configured sources, normalize into the existing `content_data` shapes, and dedupe. Today all tests are created manually via the admin API. |
| Future | **Dual Match** | Real-time head-to-head: N users get the same prompt and timer, highest overall band wins. Requires WebSocket infra and match state machine — neither exists yet. |
| Future | **Competitive rank (ELO-style)** | `rank_score` column already exists on `users` but nothing currently writes to it; rank only makes sense once Dual Match exists. |
| Future | **Team 4v4** | Two 4-person teams, one member per skill (Listening/Reading/Writing/Speaking), team overall = average of the four bands. Hard-blocked on Speaking + Dual Match both existing first. |

## 3. Target users

- **Self-study learners** — want fast, specific feedback with no teacher in the loop.
- **Competitive learners** — want to compare scores and climb a rank (once Dual Match ships).
- **Group/class learners** — want a team mode (future).

## 4. Feature detail

### 4.1 Writing — grading + improvement (shipped)

**Flow:** pick Task 1 or Task 2 → write against a suggested (non-enforced, in solo practice) timer → word-count guidance (T1 ≥150, T2 ≥250) → submit → backend grades → result returned.

**Functional requirements:**
- Grade against the 4 IELTS Writing criteria (Task Achievement/Response, Coherence & Cohesion, Lexical Resource, Grammatical Range & Accuracy), 0–9 in 0.5 steps, plus an overall band.
- Per-criterion feedback (strengths/weaknesses).
- Inline corrections (span + issue + suggestion) and a model answer.
- Persist every submission + result for progress history.

**Backend note:** grading calls an external LLM (`internal/platform/llm`); this is I/O-bound and should stay off the request-handling hot path as the feature grows (see §7).

### 4.2 Reading & Listening — auto-grading (shipped)

**Flow:** pick a test → answer questions per group (multiple choice, matching, completion types, etc. — see [ielts-rl-data-structure.md](ielts-rl-data-structure.md)) → submit → backend auto-grades, no LLM call.

**Functional requirements:**
- Support the full IELTS question-type catalogue: 14 types for Reading, 10 for Listening (6 shared).
- Grade deterministically: single-key exact match, accepted-answers membership (for fill-in-blank styles), or order-independent set match (`multiple-choice-multi`).
- Map raw correct/total to an IELTS-style band.
- Never leak answer keys before grading (`GET /api/tests*` redacts `answer`/`accepted_answers`; they reappear only in a graded submission's score).
- Listening additionally carries one shared audio file per test with per-section start/end offsets and per-question timestamp hints (review convenience only, not used in live attempts or grading).

### 4.3 Gamification — level & avatar frames (shipped)

**Functional requirements:**
- XP is the lifetime source of truth; level is a denormalized cache recomputed via a compounding curve (each level costs 4% more XP than the last, 1–100).
- XP is granted exactly once per `(user, test)` pair, on that pair's first graded submission — enforced at the database level via a composite primary key, not an application-level check, so it's race-safe under concurrent duplicate submissions.
- 100 avatar frames, one per level; frame *N* is unlocked iff the user's level ≥ *N* (fully derived, no separate unlocks table). Users can equip any unlocked frame independent of their current level.
- Resubmitting an already-attempted test is allowed for practice and still gets graded, just doesn't grant XP again.

**Acceptance criteria:**
- XP/level only change through a graded first-attempt submission — no client-supplied delta is ever trusted.
- The profile view always reflects `level = f(xp)` live, so a stale cached `level` column can never be shown to the user.

### 4.4 Speaking (planned, not yet built)

**Description:** same submission/grading/history framework as Writing, but the answer is spoken rather than typed. `content_data` for this skill already reserves `prompt`, `image_url`, and a `part` (1/2/3) field, matching Writing's shape.

**Open questions to resolve before building:**
- Audio capture format and max length per part.
- Whether grading transcribes via ASR and then reuses the Writing-style LLM rubric, or uses a dedicated audio-aware grading path.
- Storage/lifecycle for audio recordings (retention, size limits).

### 4.5 Test source worker (planned, not yet built)

**Description:** background process that periodically fetches new Writing/Reading/Listening prompts from configured sources, normalizes them into the existing `content_data` shapes per skill, and inserts them via the same validation path `POST /api/tests` already enforces.

**Functional requirements:**
- Scheduled, per-source workers; one source's failure must not affect others or corrupt the existing bank.
- Dedupe by content before insert.
- Mark newly-imported tests as "current" where relevant.
- Respect each source's terms of use / `robots.txt`.

**Open question:** concrete source(s) to integrate first, and their licensing terms — must be confirmed before building.

### 4.6 Dual Match (planned, not yet built)

**Description:** N users attempt the identical prompt under an identical, server-enforced timer; highest overall band wins.

**Functional requirements (target):**
- Matchmaking pairs users by level/rank proximity.
- Server is the single source of truth for the countdown; submission is force-locked at time-up.
- Basic anti-cheat: prompt is assigned only at match start, no edits accepted after lock, anomalous paste patterns flagged (not hard-blocked in v1).
- Match results update every participant's rank consistently, even if grading for different participants finishes at different times.

**Backend note:** needs a WebSocket layer (none exists today) and a per-match state machine (`waiting → in_progress → grading → finished`).

### 4.7 Team 4v4 (future, blocked on Speaking + Dual Match)

**Description:** two 4-person teams; each teammate takes one skill (Listening/Reading/Writing/Speaking); team overall = the four bands combined using IELTS-style rounding (nearest 0.5).

Not scoped in detail until Speaking and Dual Match both exist — listed here only to keep the data model (`content_data`, scoring) from precluding it later.

## 5. Non-functional requirements

- **Performance:** Writing grading depends on external LLM latency — should not block the HTTP request thread indefinitely; needs an explicit target once measured against the current model.
- **Concurrency:** grading and (future) crawling should not serialize on a single global lock; Dual Match will need one goroutine/state machine per match.
- **Consistency:** XP/level writes are already race-safe via a DB-level uniqueness constraint (§4.3); the same discipline applies to rank updates once Dual Match ships.
- **Security:** all `/api/*` routes require a valid JWT (`middleware.RequireAuth`); test creation is additionally gated to `role = admin` (`middleware.RequireAdmin`, checked from JWT claims, no DB round-trip); answer keys are never exposed pre-grading.
- **Extensibility:** new skills plug in as a `tests.skill` value + a grading strategy, without changing the submission/score/XP framework.
- **Observability:** grading calls, admin test creation, and (future) crawl/match jobs should be logged for debugging.

## 6. Architecture (as built)

**Backend (pure Go):**
- `net/http` + `gorilla/mux` for REST routing (`cmd/api/main.go`).
- Feature packages: `internal/feature/ielts_test` (tests, submissions, scores, grading, auto-grading), `internal/feature/profile` (level/XP/frame reads — no table of its own, reuses `auth`'s `users` row).
- Platform packages: `internal/platform/auth` (users, JWT), `internal/platform/middleware` (auth/admin/CORS), `internal/platform/llm` (external grading model client), `internal/platform/leveling` (XP↔level curve), `internal/platform/database` (Postgres + migrations).
- Static file serving for test assets (charts, images) under `/assets/`.

**Not yet in the architecture:** WebSocket server, background worker/scheduler, match state machine — all required for §4.5–4.7.

**Frontend (React, separate app):**
- Auth pages (login/register), dashboard, per-skill practice/attempt pages (Writing/Reading/Listening), submission history + detail view, admin test-creation page.
- Profile HUD showing level/XP/equipped avatar frame.

**Communication:** REST only, today. WebSocket is planned for Dual Match's real-time state and push-graded results.

## 7. API surface (current)

**Auth**
- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/refresh`

**Tests**
- `GET /api/tests?skill=&task_type=` — list, answer keys redacted.
- `GET /api/tests/{id}` — detail, answer keys redacted.
- `POST /api/tests` — admin only; create a test for any skill.

**Submissions & scores**
- `POST /api/submissions` — submit an answer payload for a test.
- `GET /api/submissions` — list the current user's submissions.
- `GET /api/submissions/{id}` — a specific submission.
- `GET /api/submissions/{id}/score` — graded result for a submission.

**Profile**
- `GET /api/profile` — `name`, `level`, `xp`, `current_level_xp`, `xp_to_next_level`, `image_url`, `equipped_frame_level`, `unlocked_max_frame_level`.
- `PUT /api/profile/frame` — `{ "frame_level": N }`, must satisfy `1 ≤ N ≤ level`.

**Planned, not yet implemented:** `POST /api/match/queue`, match WebSocket events (`match_found`, `test_assigned`, `timer_tick`, `submit_locked`, `match_result`), rank endpoints.

## 8. Data model

See [data_schema.md](data_schema.md) for the full, current ER diagram and field-level detail (generated from migrations `000001`–`000006`). Summary:

- **`users`** — auth fields + `level`, `xp`, `rank_score` (unused so far), `role`, `image_url`, `equipped_frame_level`.
- **`tests`** — `skill` (`writing | reading | listening | speaking`), `task_type`, polymorphic `content_data` JSON (shape depends on `skill`), `xp_gain`, `is_current`.
- **`submissions`** — `payload` JSON (shape depends on skill), `status` (`pending | submitted | graded | failed`).
- **`scores`** — one per submission, `overall_band`, polymorphic `details` JSON (LLM criteria breakdown for Writing/Speaking, correctness breakdown for Reading/Listening).
- **`submission_xp_grants`** — `(user_id, test_id)` primary key; makes "XP only on first graded attempt" atomic under concurrent submissions.

**Not yet modeled:** matches, rank history — to be added when Dual Match is scoped in detail.

## 9. Success metrics

- % of submissions that reach a graded state successfully.
- Average time from submission to graded result (per skill).
- Practice attempts per user per week, across all three live skills.
- Retention / return rate — signal for whether the level/frame system is working.
- (Future) completed Dual Match count, once shipped.

## 10. Risks & open questions

- **Grading accuracy:** LLM-graded Writing bands may diverge from a real examiner — needs calibration and should be labeled as indicative, not official.
- **Test source licensing (§4.5):** copyright/terms of use for any crawled source must be checked before the worker is built.
- **Speaking grading approach (§4.4):** ASR + reused Writing rubric vs. a dedicated audio pipeline — undecided.
- **Anti-cheat depth for Dual Match:** how strict should v1 be?
- **Rank formula:** `rank_score` exists on `users` but its update formula (ELO or otherwise) is undecided — needed before Dual Match ships.
- **Grading cost:** any per-day/per-user cap on LLM-graded submissions?

## 11. Decisions to confirm before building the next feature

For **Speaking**: audio format/length limits, transcription approach, storage/retention.
For **the test source worker**: concrete source(s) and their licensing terms.
For **Dual Match**: rank formula, tie-break rule when overall bands are equal, anti-cheat strictness for v1.

---
v0.2 — reflects Reading/Writing/Listening as shipped; update again once Speaking or Dual Match moves from planned to in-progress.

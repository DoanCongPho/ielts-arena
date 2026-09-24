# IELTS Test Data Structure — Developer Spec

## Shared shape (Reading & Listening)

```
Test
 └─ passages[] (Reading) / sections[] (Listening)
     └─ question_groups[]        // 1 group = one "Questions X-Y" block of the real test
         ├─ group_order
         ├─ question_type        // see the tables below
         ├─ instructions         // the text shown above the group
         ├─ shared_options[]?    // when the whole group picks from one list
         └─ questions[]
             ├─ question_order   // GLOBAL numbering across the whole test
             ├─ text
             ├─ answer
             ├─ accepted_answers[]?  // the spellings that count as correct
             ├─ explanation?     // why the answer is right (written in Vietnamese)
             └─ evidence[]?      // {paragraph, quote} — where the answer is found
```

**Explanations & evidence** (`explanation`, `evidence`) are stripped from
`GET /api/tests` like `answer` is, and only served by
`GET /api/tests/{id}/answer-key` once the user has a graded attempt at that
test (admins can always read it):

- `explanation`: plain text, `\n` between lines. Follow three steps:
  understand the question → the keywords paraphrased in the text → compare and
  conclude.
- `evidence[]`: `paragraph` is a **0-based index** into the `paragraphs[]` of
  the passage holding the question, or into the listening section's
  `transcript[]`; `quote` is a **verbatim excerpt** of that paragraph
  (whitespace and quote style `'` / `'` don't have to match). `POST /api/tests`
  rejects a quote that isn't in its paragraph.

**Rules:**

- `question_order` must run 1→40 across the WHOLE test, not restarting per
  passage/section. A `multiple-choice-multi` question covers `select_count`
  consecutive numbers.
- Every entry in `questions[]` must be a real blank / answer to grade — no
  display-only "questions" (headers, labels) mixed into the array.
- When several questions share one list of choices (matching, word bank), put
  it in the group's `shared_options` and do NOT repeat it per question.
- A fill-in-the-blank `answer` always comes with `accepted_answers[]`, the
  **array** of spellings that count, not a single fixed string.

---

## 📖 READING — 11 types

| # | question_type | Type-specific fields | Grading notes |
|---|---|---|---|
| 1 | `true-false-not-given` | — | answer ∈ {TRUE, FALSE, NOT GIVEN}, compared case-insensitively |
| 2 | `yes-no-not-given` | — | answer ∈ {YES, NO, NOT GIVEN} |
| 3 | `multiple-choice` | `questions[].options[]` ({id, text}) | one correct answer |
| 4 | `multiple-choice-multi` | `select_count`, `options[]` | answer is an **array of 2+ keys**, order-independent. One question **covers `select_count` numbers** (`question_order: 23` + `select_count: 2` = questions 23-24, the next question is 25) and scores **one mark per correct key** |
| 5 | `matching-headings` | `shared_options[]` (the list of headings), one question per paragraph | answer is a key into `shared_options` |
| 6 | `matching-information` | `shared_options[]` (paragraphs A-F, declared once) | one paragraph may answer more than one question (`allow_reuse: true`) |
| 7 | `matching-features` | `shared_options[]` (people / places) | each choice is usually used once (`allow_reuse: false`) |
| 8 | `matching-sentence-endings` | `options[]` per question, or `shared_options[]` | join the first half of a sentence to its ending |
| 9 | `sentence-completion` | `word_limit` | free-text blank, no word bank, with a word limit |
| 10 | `summary-completion` | `has_word_bank`, `word_bank[]?`, `summary_text` (holds `{{gap}}`) | without a word bank → free text; with one → pick from the box |
| 11 | `table-completion` | `table_structure` (`columns[]` as metadata, NOT questions; `rows[][]`) | only a cell holding `{{gap}}` is a real question |
| — | `short-answer` | `word_limit` | like sentence completion, but phrased as a Wh- question |
| — | `diagram-label-completion` | `diagram_image_url`, `diagram_image_urls[]?` (further images when one group labels several diagrams) | the answer is a word/phrase from the passage; `text` may be empty when the numbers are printed on the image |
| — | `flow-chart-completion` | `flow_structure.steps[]` (holds `{{gap}}`) | like a table, but as a chain of steps |

---

## 🎧 LISTENING — 10 types (plus the audio-specific fields)

**Extra fields:**

```
test.audio_url            // one recording for the whole test; optional when
                          // every section brings its own
section.audio_url         // this section's own recording; its times and its
                          // questions' timestamp_hints are then seconds into
                          // this file
section.section_start_time / section_end_time   // seconds within the recording
section.transcript[]      // what is said, as paragraphs — what evidence quotes;
                          // hidden until the attempt is graded
question.timestamp_hint   // seconds into the section's recording, for the
                          // review UI only — NEVER used for grading
```

| # | question_type | Type-specific fields | Notes |
|---|---|---|---|
| 1 | `form-completion` | `form_structure.fields[]` | filling in a form (name, date, phone…) |
| 2 | `note-completion` | `note_structure.items[]` (holds `{{gap}}`) | like a table, but as bulleted notes |
| 3 | `table-completion` | `table_structure.rows[]` | as in Reading |
| 4 | `flow-chart-completion` | `flow_structure.steps[]` | as in Reading |
| 5 | `summary-completion` | `summary_text`, `word_bank[]?` | as in Reading |
| 6 | `sentence-completion` | — | a single-sentence blank |
| 7 | `multiple-choice` | `options[]` | one answer |
| 8 | `multiple-choice-multi` | `select_count` | pick 2+ answers |
| 9 | `matching` | `shared_options[]` (opinions / features) | common in Part 3 — match each speaker to an opinion |
| 10 | `map-plan-labelling` | `map_image_url` (**required**), `location_key[]` describing each letter / place | must not be empty — without the image or the key the question is unusable (a bug hit in an earlier version) |

Note structures (`note_structure.items`, `form_structure.fields`, and each
`table_structure` cell, which may hold several `\n`-separated lines) carry
light markup that the frontend renders: `## text` is a subheading, `- text` a
bullet, one level deeper per leading tab (`\t- text`); anything else is a
plain line.

---

## Checklist for authoring a new test (so every type is covered)

- [ ] Reading: at least one group of each of the 11 types above, spread across the 3 passages
- [ ] Listening: at least one group of each of the 10 types above, spread across the 4 sections — `map-plan-labelling` especially needs a real `map_image_url`
- [ ] `question_order` totals exactly 40 marks per skill, with no duplicates and no gaps
- [ ] Every `shared_options` / `word_bank` is declared once at group level, never copied into each question
- [ ] Every fill-in-the-blank answer carries `accepted_answers` (even when only one spelling is correct)

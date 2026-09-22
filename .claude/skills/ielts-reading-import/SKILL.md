---
name: ielts-reading-import
description: Turn a Cambridge IELTS book (PDF) or a typed reading test (docx / text) into an IELTS Arena reading test — a POST /api/tests JSON body with the answers from the book's key, a Vietnamese explanation and passage evidence for every question — validate it, and load it into the local stack. Use when the user asks to import, add or convert a reading test from a document, e.g. "/ielts-reading-import ~/Downloads/cam18.pdf test 2".
---

# Import an IELTS reading test from a document

Input: a file path and which test to take (e.g. "test 2"). A Cambridge book
holds four tests plus answer keys at the back. If the file or the test
number is missing or ambiguous, ask before reading anything.

Schema details, the Cambridge-wording → `question_type` table and a worked
example are in [reference.md](reference.md). Read it before step 2.

## 1. Read the source

- **PDF**: Read with `pages` (at most 20 pages per call). Find the test's
  Reading section (Passage 1–3 with their questions) — the contents page or
  the running headers give page numbers — and the test's reading answer key
  near the back. Scanned pages are fine: read them as images.
- **docx**: `textutil -convert txt -stdout FILE.docx` (macOS), or
  `pandoc FILE.docx -t plain` if available.
- **Answers come only from the book's key.** Never work an answer out
  yourself; if the key for this test is missing, stop and tell the user.

## 2. Build the JSON

Write `tests_bank/reading/cambridge<volume>_test<N>.json` following
reference.md. The essentials:

- Top level: `skill: "reading"`, `task_type: "test<N>"`, `series:
  "cambridge"`, `volume`, `test_number`, `source: "cambridge-book"`,
  `is_current: true`, `xp_gain: 50`.
- One paragraph per printed paragraph, text **verbatim** — only undo PDF
  artefacts (line-break hyphenation, stray page headers / numbers). Lettered
  sections get `label` ("A"…); a standfirst goes first and a glossary
  footnote last, both unlabelled.
- `question_order` runs 1–40 across the whole test. A "Choose TWO" item is
  one question: `select_count: 2`, `question_order` = its first number,
  `answer` = both keys; the next question continues after both numbers.
- Fill-in answers: `answer` is the key's main form, `accepted_answers` every
  form the key allows — split "/" alternatives and expand optional words in
  brackets: "(the) soil" → `["soil", "the soil"]`.
- `{{gap}}` markers in a structure must equal that group's question count.
- `instructions`: the book's instruction text for the group, as printed.

## 3. Explanation and evidence — every question

- `explanation` (Vietnamese, plain text, `\n` between lines):
  ```
  Bước 1: Hiểu câu hỏi: <what the question asks, in Vietnamese>
  Bước 2: Từ khoá được paraphrase:
  • <question wording> ~ <passage wording>
  Bước 3: Đối chiếu: <what the passage says and why that gives the answer>
  Đáp án: <answer>
  ```
  NOT GIVEN: say what the passage does and doesn't say. FALSE / NO: name the
  contradiction. Matching / multiple choice: say why the tempting wrong
  options are wrong when it helps.
- `evidence`: `[{"paragraph": <0-based index into that passage's
  paragraphs>, "quote": "<exact excerpt>"}]` — the shortest clause or
  sentence that proves the answer, **copied character for character from
  the paragraph text you wrote** (whitespace and ' vs ’ don't matter,
  everything else does). One entry per key for "Choose TWO". For NOT GIVEN,
  cite the sentence closest to the topic, or leave evidence out.

## 4. Validate until clean

```
go run ./cmd/api validate-test tests_bank/reading/cambridge<V>_test<N>.json
```

It prints `valid: …` with counts, or the **first** problem — fix it and run
again until valid. The summary must show 40 marks and every question
explained and cited. Then check yourself what the validator can't: every
answer equals the book's key, and question numbers match the book.

## 5. Review, then import

Show the user a short summary: passages, question types per passage, and
anything you were unsure of (paragraph breaks, OCR'd words, key variants).
Import only after they say so:

```
docker compose up -d --build api     # if the api image predates this code
docker compose exec -T api ./api import-test - < tests_bank/reading/cambridge<V>_test<N>.json
```

To re-import corrected content into an existing test, use
`import-test --replace <id>`; keep the question numbering unchanged, since
past submissions are keyed by it. `tests_bank/` is gitignored — the content
is copyrighted, never commit it.

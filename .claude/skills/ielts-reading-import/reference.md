# Reading test JSON — reference

The request body of `POST /api/tests` (`CreateTestRequest` in
`internal/feature/ielts_test/request.go`); the content types are in
`models.go` and the rules in `autograde.go` (`validateContentData`). Full
field notes: `docs/ielts-rl-data-structure.md`. A complete example with
every question type: `docs/reading_test.json`.

## Shape

```jsonc
{
  "skill": "reading", "task_type": "test2",
  "series": "cambridge", "volume": 18, "test_number": 2,
  "source": "cambridge-book", "is_current": true, "xp_gain": 50,
  "content_data": {
    "passages": [{
      "title": "…",
      "paragraphs": [{ "label": "A", "text": "…" }],       // label "" when unlettered
      "question_groups": [{
        "group_order": 1,                                  // 1.. within the passage
        "question_type": "true-false-not-given",
        "instructions": "…",
        "questions": [{
          "question_order": 1,                             // 1..40 across the test
          "text": "…",
          "answer": "FALSE",
          "explanation": "Bước 1: …\nBước 2: …\n• … ~ …\nBước 3: …\nĐáp án: FALSE",
          "evidence": [{ "paragraph": 2, "quote": "…exact excerpt…" }]
        }]
      }]
    }]
  }
}
```

## Cambridge wording → question_type

| Book instruction | `question_type` | Group fields | Question fields |
|---|---|---|---|
| …agree with the **information** … TRUE / FALSE / NOT GIVEN | `true-false-not-given` | — | `text`, `answer` |
| …agree with the **claims / views of the writer** … YES / NO / NOT GIVEN | `yes-no-not-given` | — | `text`, `answer` |
| Choose the correct letter, A, B, C or D | `multiple-choice` | — | `text`, `options[{id,text}]`, `answer` |
| Choose **TWO** letters, A–E | `multiple-choice-multi` | `select_count: 2` | `text`, `options`, `answer: ["B","D"]` |
| Choose the correct **heading** for each paragraph | `matching-headings` | `shared_options` (`i`, `ii`, … → heading) | `text: "Paragraph B"`, `answer: "iv"` |
| Which **paragraph / section** contains the following information? | `matching-information` | `shared_options` (`A` → `Paragraph A`), `allow_reuse` (true if "NB You may use any letter more than once") | `text`, `answer` |
| Match each statement with the correct **person / place / …** | `matching-features` | `shared_options` (letter → name), `allow_reuse` | `text`, `answer` |
| Complete each sentence with the correct **ending** | `matching-sentence-endings` | `shared_options` (letter → ending) | `text` (sentence start), `answer` |
| Complete the **sentences** below | `sentence-completion` | `word_limit` | `text` with `__` for the blank, `answer`, `accepted_answers` |
| Answer the questions below | `short-answer` | `word_limit` | `text`, `answer`, `accepted_answers` |
| Complete the **summary** (words from the passage) | `summary-completion` | `has_word_bank: false`, `summary_text` with `{{gap}}` | `answer`, `accepted_answers` |
| Complete the summary using the **list of words, A–I** | `summary-completion` | `has_word_bank: true`, `word_bank` (letter → word), `summary_text` | `answer: "C"` (no accepted_answers) |
| Complete the **notes** | `note-completion` | `note_structure: {title, items[]}` with `{{gap}}` | `answer`, `accepted_answers` |
| Complete the **table** | `table-completion` | `table_structure: {columns[], rows[][]}` with `{{gap}}` cells | `answer`, `accepted_answers` |
| Complete the **flow-chart** | `flow-chart-completion` | `flow_structure: {steps[]}` with `{{gap}}` | `answer`, `accepted_answers` |
| Label the **diagram** | `diagram-label-completion` | `diagram_image_url` (required) | `text` (what the label points at), `answer`, `accepted_answers` |

Notes:
- Structured types (summary / notes / table / flow-chart) have no question
  `text`: the i-th `{{gap}}` in document order is answered by the i-th
  question. Headings inside notes are plain items without a gap.
- `diagram-label-completion` needs an image URL. You can't produce one from
  a PDF — ask the user for one, or skip that group and say so.
- `accepted_answers` must include `answer` itself. Matching is exact after
  trimming and lower-casing, so list real variants ("5,000" and "5000";
  "3" and "three" when the key allows either).
- Evidence is checked on import: `paragraph` must be in range and `quote`
  must occur in that paragraph (whitespace and quote style folded).

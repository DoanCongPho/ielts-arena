#!/usr/bin/env python3
"""Convert a YouPass reading quiz export into a POST /api/tests request body.

The input is the raw JSON the YouPass quiz API returns ({code, message,
data: {parts: [...]}}), fetched with the passage text included — i.e. each
part's `vocabs` tree must be populated. The output matches the reading
content_data shape in internal/feature/ielts_test/models.go and is meant to
pass validateContentData (autograde.go) as-is.

YouPass shape, as far as this script relies on it:

    data.parts[]                      one per passage
      .title
      .vocabs[]                       level-1 blocks; value is always ""
        .children[].value             level-2 = one sentence of that block
      .question_sets[]                one per question group
        .description                  HTML instructions
        .content                      HTML with <span class="gap-placeholder">
        .options[] {option, text}     shared options (TF/YN/matching)
        .allow_reuse
        .questions[]
          .order, .text, .question_type
          .correct_answer             single-key types
          .correct_answers[]          fill-in-blank types
          .options[] {option, text}   per-question multiple-choice options
          .explain                    HTML explanation -> `explanation`
          .locate_info                where the answer is, as 1-based
                                      {paragraph: vocab block, sentence,
                                      index: word} ranges -> `evidence`

Stdlib only. Usage:

    python3 scripts/youpass_reading_to_test.py IN.json [-o OUT.json]
        [--task-type test1] [--xp-gain 50] [--source youpass]
        [--thumbnail-url URL] [--not-current]
        [--series cambridge --volume 20 --test-number 1 | --no-series]

The book (series / volume / test_number) is read from the quiz title
("Orange 20 Reading - Test 1" -> cambridge 20, test 1) unless given.
"""

import argparse
import json
import re
import sys
from html.parser import HTMLParser
from pathlib import Path

GAP = "{{gap}}"

# YouPass per-question `question_type` -> our QuestionType. The per-set
# `question_type` (SINGLE_SELECTION, GAP_FILLING, TABLE_SELECTION, ...) only
# describes their widget and is ambiguous, so it's ignored.
# SUMMARY_COMPLETION is resolved further in fill_group_type(): YouPass uses
# it for both note and summary completion.
TYPE_MAP = {
    "TRUE_FALSE": "true-false-not-given",
    "YES_NO": "yes-no-not-given",
    "MULTIPLE_CHOICE_ONE": "multiple-choice",
    "MULTIPLE_CHOICE_MANY": "multiple-choice-multi",
    "MATCHING_HEADING": "matching-headings",
    "MATCHING_INFO": "matching-information",
    "MATCHING_FEATURES": "matching-features",
    "MATCHING_ENDINGS": "matching-sentence-endings",
    "SUMMARY_COMPLETION": "summary-completion",
    "SENTENCE_COMPLETION": "sentence-completion",
}

_NUMBER_WORDS = {"ONE": 1, "TWO": 2, "THREE": 3, "FOUR": 4}

BLOCK_TAGS = {"h1", "h2", "h3", "h4", "h5", "h6", "p", "li", "div"}

_QUOTES = str.maketrans({"‘": "'", "’": "'", "“": '"', "”": '"', "\xa0": " "})


class ConvertError(Exception):
    pass


def clean(s):
    """Collapse whitespace and fold typographic quotes, so answers typed with
    a plain apostrophe still match (normalizeAnswer doesn't fold quotes)."""
    return re.sub(r"\s+", " ", (s or "").translate(_QUOTES)).strip()


class _BlockParser(HTMLParser):
    """Flattens HTML into (tag, text) blocks. Only the innermost block
    element owns its text, so <li><div>x</div></li> yields one block."""

    def __init__(self):
        super().__init__(convert_charrefs=True)
        self.stack = []  # [tag, text_parts, has_child_block]
        self.blocks = []
        self.loose = []  # text outside any block element

    def handle_starttag(self, tag, attrs):
        if tag == "br":
            self._text(" ")
        elif tag in BLOCK_TAGS:
            if self.stack:
                self.stack[-1][2] = True
            self.stack.append([tag, [], False])

    def handle_endtag(self, tag):
        if tag not in BLOCK_TAGS or not self.stack:
            return
        # Tolerate sloppy nesting: close up to the matching tag.
        while self.stack:
            t, parts, _ = self.stack.pop()
            text = clean("".join(parts))
            if text:
                # <li><div>x</div></li>: the div owns the text, but it's
                # still a list item.
                in_li = any(s[0] == "li" for s in self.stack)
                self.blocks.append(("li" if in_li else t, text))
            if t == tag:
                break

    def handle_data(self, data):
        self._text(data)

    def _text(self, s):
        (self.stack[-1][1] if self.stack else self.loose).append(s)


def html_blocks(src):
    src = re.sub(r'<span[^>]*class="gap-placeholder"[^>]*>.*?</span>', GAP, src or "", flags=re.S)
    p = _BlockParser()
    p.feed(src)
    p.close()
    blocks = p.blocks
    loose = clean("".join(p.loose))
    if loose:
        blocks.append(("text", loose))
    return blocks


def html_text(src):
    return " ".join(text for _, text in html_blocks(src))


# ---------------------------------------------------------------------------
# Passages
# ---------------------------------------------------------------------------

_LABEL_RE = re.compile(r"^([A-Z])\.\s+")


def convert_paragraphs(part):
    """Builds paragraphs from the part's `vocabs` tree: one paragraph per
    non-empty level-1 block, sentences joined with a space. Empty padding
    blocks and a block repeating the passage title are dropped; standfirst
    and footnote blocks are kept as unlabelled paragraphs since they're part
    of the printed passage.

    Also returns, for convert_evidence, a map from each kept block's index
    in `vocabs` to (its paragraph index, its cleaned sentences)."""
    kept = []
    for bi, block in enumerate(part.get("vocabs") or []):
        sentences = [clean(c.get("value", "")) for c in block.get("children") or []]
        text = clean(" ".join(sentences))
        if text and text != clean(part.get("title")):
            kept.append((bi, text, sentences))
    blocks = [text for _, text, _ in kept]
    if not blocks:
        raise ConvertError(f"part {part.get('passage')!r} has no passage text — export it with vocabs populated")

    # Section letters ("A. Around 25 million...") are inlined into the text.
    # Only treat them as labels when they run A, B, C... in order, so a
    # sentence that happens to start with "I. " isn't mistaken for one.
    labelled = [(i, m.group(1)) for i, b in enumerate(blocks) if (m := _LABEL_RE.match(b))]
    letters = [l for _, l in labelled]
    use_labels = len(letters) >= 2 and letters == [chr(ord("A") + k) for k in range(len(letters))]

    paragraphs = []
    block_map = {}
    label_at = dict(labelled) if use_labels else {}
    for i, (bi, text, sentences) in enumerate(kept):
        label = label_at.get(i, "")
        if label:
            text = _LABEL_RE.sub("", text, count=1)
        paragraphs.append({"label": label, "text": text})
        block_map[bi] = (i, sentences)
    return paragraphs, block_map


# ---------------------------------------------------------------------------
# Explanations and evidence
# ---------------------------------------------------------------------------


def convert_explanation(src):
    """YouPass's explanation HTML as plain text: one line per block, list
    items bulleted."""
    return "\n".join(f"• {text}" if tag == "li" else text for tag, text in html_blocks(src))


def locate_ranges(locate):
    """A question's paragraph_ranges. YouPass puts them directly on the
    question, or — for a choose-TWO question — one set per key under
    "0", "1", ..."""
    if not isinstance(locate, dict):
        return []
    if "paragraph_ranges" in locate:
        return locate["paragraph_ranges"] or []
    return [r for v in locate.values() if isinstance(v, dict) for r in v.get("paragraph_ranges") or []]


def convert_evidence(q, paragraphs, block_map):
    """Turns locate_info's word ranges into {paragraph, quote} evidence.
    Every quote is checked against its paragraph the way the backend will
    (whitespace / quote style folded — clean() already does both); one that
    doesn't match is dropped with a warning rather than failing the test."""
    evidence = []
    for rng in locate_ranges(q.get("locate_info")):
        start, end = rng.get("start") or {}, rng.get("end") or {}
        bi = (start.get("paragraph") or 0) - 1
        if bi not in block_map:
            continue
        pi, sentences = block_map[bi]
        # A range running into the next block is cut at the end of this one.
        same_block = end.get("paragraph") == start.get("paragraph")
        last_sentence = end.get("sentence", len(sentences)) if same_block else len(sentences)
        words = []
        for k in range(start.get("sentence", 1), last_sentence + 1):
            if not 1 <= k <= len(sentences):
                continue
            w = sentences[k - 1].split()
            lo = start.get("index", 1) - 1 if k == start.get("sentence") else 0
            hi = end.get("index", len(w)) if same_block and k == last_sentence else len(w)
            words += w[max(lo, 0):hi]
        quote = " ".join(words)
        label = paragraphs[pi]["label"]
        if label and quote.startswith(f"{label}. "):
            quote = quote[len(label) + 2 :]
        if quote and quote in paragraphs[pi]["text"]:
            if {"paragraph": pi, "quote": quote} not in evidence:
                evidence.append({"paragraph": pi, "quote": quote})
        else:
            print(f"warning: question {q.get('order')}: dropped evidence that doesn't match the passage", file=sys.stderr)
    return evidence


def attach_explanations(passage, sets, block_map):
    """Adds explanation / evidence from the raw YouPass questions to the
    converted ones, matched by question_order."""
    raw_by_order = {q["order"]: q for s in sets for q in s["questions"]}
    for group in passage["question_groups"]:
        for q in group["questions"]:
            raw = raw_by_order.get(q["question_order"])
            if not raw:
                continue
            explanation = convert_explanation(raw.get("explain"))
            if explanation:
                q["explanation"] = explanation
            evidence = convert_evidence(raw, passage["paragraphs"], block_map)
            if evidence:
                q["evidence"] = evidence


# ---------------------------------------------------------------------------
# Question groups
# ---------------------------------------------------------------------------


def group_type(qset):
    kinds = {q.get("question_type") for q in qset["questions"]}
    if len(kinds) != 1:
        raise ConvertError(f"set {qset.get('title')!r} mixes question types {sorted(kinds)}")
    kind = kinds.pop()
    if kind not in TYPE_MAP:
        raise ConvertError(
            f"set {qset.get('title')!r}: unsupported YouPass question_type {kind!r} — add it to TYPE_MAP"
        )
    qtype = TYPE_MAP[kind]
    if qtype == "summary-completion":
        qtype = fill_group_type(qset)
    return qtype


def fill_group_type(qset):
    """YouPass labels note completion SUMMARY_COMPLETION too. The instructions
    say which it is ("Complete the notes/summary below"); only when they
    don't, fall back to layout — notes are usually a bullet list, though
    YouPass sometimes bullets a summary's paragraphs too."""
    instructions = html_text(qset.get("description")).lower()
    if "complete the notes" in instructions:
        return "note-completion"
    if "complete the summary" in instructions:
        return "summary-completion"
    return "note-completion" if "<li" in (qset.get("content") or "") else "summary-completion"


def options(raw):
    return [{"id": clean(o["option"]), "text": clean(o.get("text"))} for o in raw or []]


def key_answer(q):
    ans = clean(q.get("correct_answer"))
    if not ans:
        raise ConvertError(f"question {q['order']} has no correct_answer")
    return ans


def fill_answers(q):
    accepted = []
    for a in q.get("correct_answers") or [q.get("correct_answer")]:
        a = clean(a)
        if a and a.lower() not in (x.lower() for x in accepted):
            accepted.append(a)
    if not accepted:
        raise ConvertError(f"question {q['order']} has no correct_answers")
    return {"answer": accepted[0], "accepted_answers": accepted}


def word_limit(instructions):
    """"NO MORE THAN TWO WORDS" / "ONE WORD ONLY" -> 2 / 1; 0 if not stated."""
    m = re.search(r"\b(ONE|TWO|THREE|FOUR)\s+WORDS?\b", instructions)
    return _NUMBER_WORDS[m.group(1)] if m else 0


def check_gaps(qset, qtype, text_parts, questions):
    n = sum(t.count(GAP) for t in text_parts)
    if n != len(questions):
        raise ConvertError(f"set {qset.get('title')!r} ({qtype}): {n} gaps but {len(questions)} questions")


def convert_group(qset, order):
    qtype = group_type(qset)
    qs = sorted(qset["questions"], key=lambda q: q["order"])
    group = {
        "group_order": order,
        "question_type": qtype,
        "instructions": html_text(qset.get("description")),
    }

    if qtype in ("true-false-not-given", "yes-no-not-given"):
        group["questions"] = [
            {"question_order": q["order"], "text": clean(q.get("text")), "answer": key_answer(q)} for q in qs
        ]

    elif qtype == "multiple-choice":
        group["questions"] = [
            {
                "question_order": q["order"],
                "text": clean(html_text(q.get("text"))),
                "options": options(q.get("options")),
                "answer": key_answer(q),
            }
            for q in qs
        ]

    elif qtype == "multiple-choice-multi":
        # YouPass stores "Questions 23-24: choose TWO" as one question with
        # order 23 and both keys — the same shape as ours, where the question
        # spans select_count numbers starting at question_order.
        answers = [[clean(a) for a in q.get("correct_answers") or []] for q in qs]
        select = qset.get("max_selections") or len(answers[0])
        for q, a in zip(qs, answers):
            if len(a) != select:
                raise ConvertError(f"question {q['order']}: {len(a)} keys but the set asks for {select}")
        group["select_count"] = select
        group["questions"] = [
            {
                "question_order": q["order"],
                "text": clean(html_text(q.get("text"))),
                "options": options(q.get("options")),
                "answer": a,
            }
            for q, a in zip(qs, answers)
        ]

    elif qtype in ("matching-headings", "matching-information", "matching-features", "matching-sentence-endings"):
        shared = options(qset.get("options"))
        if qtype == "matching-information":
            # YouPass lists bare letters ("A"); spell them out for the dropdown.
            shared = [{"id": o["id"], "text": f"Paragraph {o['id']}" if o["text"] == o["id"] else o["text"]} for o in shared]
        group["shared_options"] = shared
        # allow_reuse is only a UI hint for these two; each heading and each
        # sentence ending is used once.
        if qtype in ("matching-information", "matching-features"):
            group["allow_reuse"] = bool(qset.get("allow_reuse"))
        group["questions"] = [
            {"question_order": q["order"], "text": clean(html_text(q.get("text"))), "answer": key_answer(q)} for q in qs
        ]

    elif qtype == "note-completion":
        blocks = html_blocks(qset.get("content"))
        title = ""
        if blocks and blocks[0][0] in ("h1", "h2", "h3") and GAP not in blocks[0][1]:
            title = blocks.pop(0)[1]
        items = [text for _, text in blocks]
        check_gaps(qset, qtype, items, qs)
        group["note_structure"] = {"title": title, "items": items}
        group["questions"] = [{"question_order": q["order"], **fill_answers(q)} for q in qs]

    elif qtype == "sentence-completion":
        # One sentence per block, each with exactly one gap. Our shape keeps
        # the sentence on the question itself with "__" marking the blank.
        sentences = [text for _, text in html_blocks(qset.get("content")) if GAP in text]
        check_gaps(qset, qtype, sentences, qs)
        if len(sentences) != len(qs):
            raise ConvertError(f"set {qset.get('title')!r}: expected one gap per sentence")
        limit = word_limit(group["instructions"])
        if limit:
            group["word_limit"] = limit
        group["questions"] = [
            {"question_order": q["order"], "text": s.replace(GAP, "__"), **fill_answers(q)}
            for q, s in zip(qs, sentences)
        ]

    elif qtype == "summary-completion":
        summary = "\n".join(text for _, text in html_blocks(qset.get("content")))
        check_gaps(qset, qtype, [summary], qs)
        group["summary_text"] = summary
        word_bank = options(qset.get("options"))
        group["has_word_bank"] = bool(word_bank)
        if word_bank:
            # With a word bank the answer is a single key (A, B, ...), graded
            # by exact match rather than accepted_answers.
            group["word_bank"] = word_bank
            group["questions"] = [
                {"question_order": q["order"], "answer": key_answer(q) if q.get("correct_answer") else fill_answers(q)["answer"]}
                for q in qs
            ]
        else:
            group["questions"] = [{"question_order": q["order"], **fill_answers(q)} for q in qs]

    return group


# ---------------------------------------------------------------------------


def convert(raw, task_type, xp_gain, source, thumbnail_url, is_current, series=None):
    data = raw.get("data", raw)
    parts = sorted(data.get("parts") or [], key=lambda p: (p.get("passage") or 0, p.get("sort") or 0))
    if not parts:
        raise ConvertError("no parts found")

    passages = []
    for part in parts:
        sets = sorted(part.get("question_sets") or [], key=lambda s: s.get("sort") or 0)
        paragraphs, block_map = convert_paragraphs(part)
        passage = {
            "title": clean(part.get("title")),
            "paragraphs": paragraphs,
            "question_groups": [convert_group(s, i + 1) for i, s in enumerate(sets)],
        }
        attach_explanations(passage, sets, block_map)
        passages.append(passage)

    body = {
        "skill": "reading",
        "task_type": task_type,
    }
    if series:
        body["series"], body["volume"], body["test_number"] = series
    body |= {
        "source": source,
        "is_current": is_current,
        "xp_gain": xp_gain,
    }
    if thumbnail_url:
        body["thumbnail_url"] = thumbnail_url
    body["content_data"] = {"passages": passages}
    return body


def guess_series(data):
    """("cambridge", volume, test) from a title like "Orange 20 Reading -
    Test 1" (YouPass's name for Cambridge IELTS 20), else None."""
    m = re.search(r"(\d+)\D*?\bTest\s*(\d+)", data.get("title") or "", re.I)
    return ("cambridge", int(m.group(1)), int(m.group(2))) if m else None


def guess_task_type(path, n_parts):
    m = re.search(r"test[_-]?(\d+)", path.stem, re.I)
    if m:
        return f"test{m.group(1)}"
    return "test1" if n_parts > 1 else "passage1"


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("input", type=Path)
    ap.add_argument("-o", "--output", type=Path, help="default: stdout")
    ap.add_argument("--task-type", help="default: testN from the file name")
    ap.add_argument("--xp-gain", type=int, default=50)
    ap.add_argument("--source", default="youpass")
    ap.add_argument("--thumbnail-url", default="")
    ap.add_argument("--not-current", action="store_true", help="set is_current=false")
    ap.add_argument("--series", help="book series, e.g. cambridge (default: from the quiz title)")
    ap.add_argument("--volume", type=int, help="book volume, e.g. 20")
    ap.add_argument("--test-number", type=int, help="test number within the book")
    ap.add_argument("--no-series", action="store_true", help="don't place the test in a book")
    args = ap.parse_args()

    raw = json.loads(args.input.read_text(encoding="utf-8"))
    data = raw.get("data", raw)
    n_parts = len(data.get("parts") or [])
    series = None
    if not args.no_series:
        guessed = guess_series(data) or (None, None, None)
        series = (args.series or guessed[0], args.volume or guessed[1], args.test_number or guessed[2])
        if not all(series):
            sys.exit("error: couldn't tell the book from the title — pass --series/--volume/--test-number or --no-series")
    try:
        body = convert(
            raw,
            task_type=args.task_type or guess_task_type(args.input, n_parts),
            xp_gain=args.xp_gain,
            source=args.source,
            thumbnail_url=args.thumbnail_url,
            is_current=not args.not_current,
            series=series,
        )
    except ConvertError as e:
        sys.exit(f"error: {e}")

    out = json.dumps(body, ensure_ascii=False, indent=2) + "\n"
    if args.output:
        args.output.write_text(out, encoding="utf-8")
        n_q = sum(len(g["questions"]) for p in body["content_data"]["passages"] for g in p["question_groups"])
        print(f"wrote {args.output} ({len(body['content_data']['passages'])} passages, {n_q} questions)", file=sys.stderr)
    else:
        sys.stdout.write(out)


if __name__ == "__main__":
    main()

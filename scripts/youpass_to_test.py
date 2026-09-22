#!/usr/bin/env python3
"""Convert a YouPass reading or listening quiz export into a POST /api/tests
request body.

The input is the raw JSON the YouPass quiz API returns ({code, message,
data: {parts: [...]}}), fetched with ?included_vocabs=true — each part's
`vocabs` tree holds the reading passage, or the listening section's
transcript. The output matches the content_data shapes in
internal/feature/ielts_test/models.go and is meant to pass
validateContentData (autograde.go) as-is. The skill comes from the quiz
title ("Orange 10 Listening - Test 1").

Listening: each part becomes a section with its own recording
(https://cms.youpass.vn/assets/<part.file_id>) — YouPass has no
whole-test file — timed from 0; question listen_from (seconds into the
whole test) becomes a timestamp_hint within the section's file.

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

    python3 scripts/youpass_to_test.py IN.json [-o OUT.json]
        [--task-type test1] [--xp-gain 50] [--source youpass]
        [--thumbnail-url URL] [--not-current]
        [--series cambridge --volume 20 --test-number 1 | --no-series]
        [--image-dir internal/assets/diagrams] [--audio-dir media/audio]

The book (series / volume / test_number) is read from the quiz title
("Orange 20 Reading - Test 1" -> cambridge 20, test 1) unless given.

Diagram images point at YouPass's CMS unless --image-dir is given: then
they're downloaded there and referenced as /assets/<dir name>/<file>, which
the API serves out of internal/assets (rebuild the api image after).
Likewise --audio-dir for listening recordings, referenced as
/assets/<dir name>/<file_id>.mp3; docker-compose.yml mounts media/audio at
/assets/audio/.
"""

import argparse
import html
import json
import re
import sys
import urllib.request
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
    "MATCHING": "matching",
    "SENTENCE_COMPLETION": "sentence-completion",
    "SHORT_ANSWER": "short-answer",
    "MAP_DIAGRAM_LABEL": "diagram-label-completion",
}

# Where a YouPass type means something else in a listening test.
LISTENING_TYPE_MAP = {
    "MAP_DIAGRAM_LABEL": "map-plan-labelling",  # letters A-I on a map/plan
}

YOUPASS_ASSETS = "https://cms.youpass.vn/assets/"

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
    """Flattens HTML into (tag, text) blocks, in document order. A block's
    own text is emitted as a run whenever a child block starts or the block
    ends, so text around children keeps its place; text outside any block
    (YouPass fragments sometimes start mid-paragraph, "<br>x</p>") forms
    runs of its own. With br_breaks, <br> also ends a run — explanations use
    it as a line break."""

    def __init__(self, br_breaks=False):
        super().__init__(convert_charrefs=True)
        self.br_breaks = br_breaks
        self.stack = []  # [tag, text_parts]
        self.loose = []  # text outside any block element
        self.blocks = []
        self.depths = []  # per block: how many list items deep it sits

    def _flush(self, parts, tag, depth=0):
        text = clean("".join(parts))
        parts.clear()
        if text:
            # <li><div>x</div></li>: the div owns the text, but it's still a
            # list item.
            self.blocks.append(("li" if depth else tag, text))
            self.depths.append(depth)

    def _li_depth(self):
        return sum(1 for t, _ in self.stack if t == "li")

    def _flush_current(self):
        if self.stack:
            self._flush(self.stack[-1][1], self.stack[-1][0], self._li_depth())
        else:
            self._flush(self.loose, "text")

    def handle_starttag(self, tag, attrs):
        if tag == "br":
            if self.br_breaks:
                self._flush_current()
            else:
                self._text(" ")
        elif tag in BLOCK_TAGS:
            self._flush_current()
            self.stack.append([tag, []])

    def handle_endtag(self, tag):
        if tag not in BLOCK_TAGS:
            return
        if not any(t == tag for t, _ in self.stack):
            # A closing tag with no opener ends the loose run before it.
            if not self.stack:
                self._flush(self.loose, "text")
            return
        # Tolerate sloppy nesting: close up to the matching tag.
        while self.stack:
            t, parts = self.stack.pop()
            self._flush(parts, t, self._li_depth() + (t == "li"))
            if t == tag:
                break

    def handle_data(self, data):
        self._text(data)

    def _text(self, s):
        (self.stack[-1][1] if self.stack else self.loose).append(s)

    def close(self):
        super().close()
        while self.stack:
            t, parts = self.stack.pop()
            self._flush(parts, t, self._li_depth() + (t == "li"))
        self._flush(self.loose, "text")


def _parse(src, br_breaks=False):
    src = re.sub(r'<span[^>]*class="gap-placeholder"[^>]*>.*?</span>', GAP, src or "", flags=re.S)
    p = _BlockParser(br_breaks)
    p.feed(src)
    p.close()
    return p


def html_blocks(src, br_breaks=False):
    return _parse(src, br_breaks).blocks


def html_outline(src):
    """(tag, text, list depth) per block — html_blocks plus how deep in
    nested lists each block sits (0 outside any list)."""
    p = _parse(src)
    return [(tag, text, depth) for (tag, text), depth in zip(p.blocks, p.depths)]


def marked_lines(outline):
    """Structure lines with the light markup the frontend renders (see
    NoteStructure in models.go): list items become "- text", one leading
    tab per extra nesting level; h4-h6 become "## text" subheadings; the
    rest stay plain."""
    lines = []
    for tag, text, depth in outline:
        if depth:
            lines.append("\t" * (depth - 1) + "- " + text)
        elif tag in ("h4", "h5", "h6"):
            lines.append("## " + text)
        else:
            lines.append(text)
    return lines


def unmarked(line):
    """A marked line's text without its markup."""
    return re.sub(r"^(\t*- |## )", "", line)


def html_text(src):
    return " ".join(text for _, text in html_blocks(src))


# ---------------------------------------------------------------------------
# Passages
# ---------------------------------------------------------------------------

_LABEL_RE = re.compile(r"^([A-Z])\.\s+")


def convert_paragraphs(part, labels=True):
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
    use_labels = labels and len(letters) >= 2 and letters == [chr(ord("A") + k) for k in range(len(letters))]

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
    """YouPass's explanation HTML as plain text: one line per block or <br>
    line, list items bulleted (unless the text already starts with one)."""
    lines = []
    for tag, text in html_blocks(src, br_breaks=True):
        bullet = tag == "li" and not text.startswith("•")
        lines.append(f"• {text}" if bullet else text)
    return "\n".join(lines)


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


def attach_explanations(unit, sets, paragraphs, block_map, audio_offset=None, duration=None):
    """Adds explanation / evidence from the raw YouPass questions to the
    converted ones of a passage or section, matched by question_order;
    `paragraphs` is the passage text, or the section transcript. For a
    listening section (audio_offset set), listen_from — seconds into the
    whole test — becomes a timestamp_hint within the section's recording."""
    raw_by_order = {q["order"]: q for s in sets for q in s["questions"]}
    for group in unit["question_groups"]:
        for q in group["questions"]:
            raw = raw_by_order.get(q["question_order"])
            if not raw:
                continue
            explanation = convert_explanation(raw.get("explain"))
            if explanation:
                q["explanation"] = explanation
            evidence = convert_evidence(raw, paragraphs, block_map)
            if evidence:
                q["evidence"] = evidence
            heard = raw.get("listen_from")
            if audio_offset is not None and heard is not None and 0 <= heard - audio_offset <= duration:
                q["timestamp_hint"] = heard - audio_offset


# ---------------------------------------------------------------------------
# Question groups
# ---------------------------------------------------------------------------


def group_type(qset, skill):
    kinds = {q.get("question_type") for q in qset["questions"]}
    if len(kinds) != 1:
        raise ConvertError(f"set {qset.get('title')!r} mixes question types {sorted(kinds)}")
    kind = kinds.pop()
    if kind not in TYPE_MAP:
        raise ConvertError(
            f"set {qset.get('title')!r}: unsupported YouPass question_type {kind!r} — add it to TYPE_MAP"
        )
    qtype = (LISTENING_TYPE_MAP if skill == "listening" else {}).get(kind) or TYPE_MAP[kind]
    if qtype == "summary-completion":
        qtype = fill_group_type(qset) if skill == "reading" else listening_fill_type(qset)
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


def listening_fill_type(qset):
    """Listening gap-fills all come as SUMMARY_COMPLETION. A real table (a
    row with 2+ cells) is table completion — Cambridge also lays notes and
    forms out in one-column tables; otherwise the instructions name it,
    and notes are the default ("Complete the notes below" is by far the
    most common, and some sets have no "Complete the ..." line at all)."""
    if max((len(table_row_cells(r)) for r in table_rows(qset.get("content"))), default=0) >= 2:
        return "table-completion"
    instructions = html_text(qset.get("description")).lower()
    if "flow" in instructions:
        return "flow-chart-completion"
    if "complete the form" in instructions:
        return "form-completion"
    if "complete the sentences" in instructions:
        return "sentence-completion"
    return "note-completion"


def table_rows(src):
    return re.findall(r"<tr\b.*?</tr>", src or "", flags=re.S | re.I)


def table_row_cells(row):
    """A row's cells, each as marked lines joined by "\\n" — a cell can
    hold a bulleted list."""
    cells = re.findall(r"<t[dh]\b[^>]*>(.*?)</t[dh]>", row, flags=re.S | re.I)
    return ["\n".join(marked_lines(html_outline(c))) for c in cells]


def convert_table(src):
    """A table's rows as cell texts ({{gap}} kept). The first row becomes
    the column headers unless it has a gap itself."""
    rows = [table_row_cells(r) for r in table_rows(src)]
    rows = [r for r in rows if any(r)]
    if rows and GAP not in "".join(rows[0]):
        columns = [" ".join(unmarked(l) for l in c.split("\n")) for c in rows[0]]
        return {"columns": columns, "rows": rows[1:]}
    return {"columns": [""] * max((len(r) for r in rows), default=1), "rows": rows}


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


# Set by --image-dir / --audio-dir: where diagram images and listening
# recordings are saved instead of hotlinked.
IMAGE_DIR = None
AUDIO_DIR = None

_IMAGE_EXT = {"image/avif": ".avif", "image/webp": ".webp", "image/png": ".png", "image/jpeg": ".jpg", "image/gif": ".gif"}


def localize_image(src):
    """Downloads a diagram into IMAGE_DIR (named after the CMS asset id)
    and returns its /assets/ URL; returns src untouched without IMAGE_DIR.
    An image already downloaded isn't fetched again."""
    if IMAGE_DIR is None:
        return src
    asset_id = re.sub(r"[^A-Za-z0-9-]", "", src.split("?")[0].rstrip("/").rsplit("/", 1)[-1]) or "diagram"
    existing = sorted(IMAGE_DIR.glob(asset_id + ".*"))
    if existing:
        name = existing[0].name
    else:
        req = urllib.request.Request(src, headers={"User-Agent": "ielts-arena-import"})
        with urllib.request.urlopen(req, timeout=30) as resp:
            ctype = resp.headers.get_content_type()
            if not ctype.startswith("image/"):
                raise ConvertError(f"diagram {src} is {ctype}, not an image")
            name = asset_id + _IMAGE_EXT.get(ctype, ".img")
            IMAGE_DIR.mkdir(parents=True, exist_ok=True)
            (IMAGE_DIR / name).write_bytes(resp.read())
    return f"/assets/{IMAGE_DIR.name}/{name}"


def localize_audio(file_id):
    """A listening part's recording: its YouPass URL, or with AUDIO_DIR the
    local copy (downloaded unless already there) as an /assets/ URL."""
    src = YOUPASS_ASSETS + file_id
    if AUDIO_DIR is None:
        return src
    name = re.sub(r"[^A-Za-z0-9-]", "", file_id) + ".mp3"
    path = AUDIO_DIR / name
    if not path.exists() or path.stat().st_size == 0:
        req = urllib.request.Request(src, headers={"User-Agent": "ielts-arena-import"})
        with urllib.request.urlopen(req, timeout=120) as resp:
            ctype = resp.headers.get_content_type()
            if not ctype.startswith("audio/"):
                raise ConvertError(f"recording {src} is {ctype}, not audio")
            AUDIO_DIR.mkdir(parents=True, exist_ok=True)
            path.write_bytes(resp.read())
    return f"/assets/{AUDIO_DIR.name}/{name}"


def word_limit(instructions):
    """"NO MORE THAN TWO WORDS" / "ONE WORD ONLY" -> 2 / 1; 0 if not stated."""
    m = re.search(r"\b(ONE|TWO|THREE|FOUR)\s+WORDS?\b", instructions)
    return _NUMBER_WORDS[m.group(1)] if m else 0


def check_gaps(qset, qtype, text_parts, questions):
    n = sum(t.count(GAP) for t in text_parts)
    if n != len(questions):
        raise ConvertError(f"set {qset.get('title')!r} ({qtype}): {n} gaps but {len(questions)} questions")


def convert_group(qset, order, skill="reading"):
    qtype = group_type(qset, skill)
    qs = sorted(qset["questions"], key=lambda q: q["order"])
    group = {
        "group_order": order,
        "question_type": qtype,
        "instructions": html_text(qset.get("description")),
    }

    if not group["instructions"] and qtype == "multiple-choice":
        # YouPass leaves a few of these blank; Cambridge's wording follows
        # from the option letters.
        ids = [o["option"] for o in (qs[0].get("options") or [])]
        if len(ids) >= 2:
            group["instructions"] = f"Choose the correct letter, {', '.join(ids[:-1])} or {ids[-1]}."

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

    elif qtype in ("matching-headings", "matching-information", "matching-features", "matching-sentence-endings", "matching"):
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

    elif qtype == "map-plan-labelling":
        # The map is an <img> in the instructions (or the set's image) and
        # the options are its letters; each question names a place.
        desc = qset.get("description") or ""
        srcs = [html.unescape(s) for s in re.findall(r'<img[^>]*\ssrc="([^"]+)"', desc)]
        if not srcs and qset.get("image"):
            srcs = [YOUPASS_ASSETS + qset["image"]]
        if not srcs:
            raise ConvertError(f"set {qset.get('title')!r}: map labelling without a map image")
        group["map_image_url"] = localize_image(srcs[0])
        group["location_key"] = options(qset.get("options"))
        group["questions"] = [
            {"question_order": q["order"], "text": clean(html_text(q.get("text"))), "answer": key_answer(q)} for q in qs
        ]

    elif qtype == "table-completion":
        table = convert_table(qset.get("content"))
        check_gaps(qset, qtype, [c for r in table["rows"] for c in r], qs)
        group["table_structure"] = table
        group["questions"] = [{"question_order": q["order"], **fill_answers(q)} for q in qs]

    elif qtype in ("form-completion", "flow-chart-completion"):
        outline = html_outline(qset.get("content"))
        items = marked_lines(outline) if qtype == "form-completion" else [text for _, text, _ in outline]
        title = ""
        if qtype == "form-completion" and items and GAP not in items[0]:
            title = unmarked(items.pop(0))
        check_gaps(qset, qtype, items, qs)
        if qtype == "form-completion":
            group["form_structure"] = {"title": title, "fields": items}
        else:
            group["flow_structure"] = {"steps": items}
        group["questions"] = [{"question_order": q["order"], **fill_answers(q)} for q in qs]

    elif qtype == "note-completion":
        outline = html_outline(qset.get("content"))
        title = ""
        if outline and outline[0][0] in ("h1", "h2", "h3") and GAP not in outline[0][1]:
            title = outline.pop(0)[1]
        items = marked_lines(outline)
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

    elif qtype == "short-answer":
        # One Wh- question per block, its answer box a trailing gap.
        prompts = [text for _, text in html_blocks(qset.get("content")) if GAP in text]
        check_gaps(qset, qtype, prompts, qs)
        if len(prompts) != len(qs):
            raise ConvertError(f"set {qset.get('title')!r}: expected one gap per question")
        limit = word_limit(group["instructions"])
        if limit:
            group["word_limit"] = limit
        group["questions"] = [
            {"question_order": q["order"], "text": clean(p.replace(GAP, "")), **fill_answers(q)}
            for q, p in zip(qs, prompts)
        ]

    elif qtype == "diagram-label-completion":
        # The label numbers are printed on the diagram itself, so questions
        # carry no text; the images stay on YouPass's public CMS.
        content = qset.get("content") or ""
        images = [html.unescape(src) for src in re.findall(r'<img[^>]*\ssrc="([^"]+)"', content)]
        if not images:
            raise ConvertError(f"set {qset.get('title')!r}: diagram labelling without an image")
        check_gaps(qset, qtype, [html_text(content)], qs)
        images = [localize_image(src) for src in images]
        group["diagram_image_url"] = images[0]
        if len(images) > 1:
            group["diagram_image_urls"] = images[1:]
        limit = word_limit(group["instructions"])
        if limit:
            group["word_limit"] = limit
        group["questions"] = [{"question_order": q["order"], **fill_answers(q)} for q in qs]

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
    skill = "listening" if "listening" in (data.get("title") or "").lower() else "reading"
    parts = sorted(data.get("parts") or [], key=lambda p: (p.get("passage") or 0, p.get("sort") or 0))
    if not parts:
        raise ConvertError("no parts found")

    units = []
    for n, part in enumerate(parts, 1):
        sets = sorted(part.get("question_sets") or [], key=lambda s: s.get("sort") or 0)
        groups = [convert_group(s, i + 1, skill) for i, s in enumerate(sets)]
        if skill == "reading":
            paragraphs, block_map = convert_paragraphs(part)
            unit = {"title": clean(part.get("title")), "paragraphs": paragraphs, "question_groups": groups}
            attach_explanations(unit, sets, paragraphs, block_map)
        else:
            unit, transcript, block_map = convert_section(part, n, groups)
            attach_explanations(unit, sets, transcript, block_map, part["listen_from"], unit["section_end_time"])
        units.append(unit)

    body = {
        "skill": skill,
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
    body["content_data"] = {"passages": units} if skill == "reading" else {"sections": units}
    return body


def convert_section(part, n, groups):
    """A listening part as a section with its own recording, timed from 0
    to the part's length, and its transcript (the part's vocabs)."""
    if not part.get("file_id"):
        raise ConvertError(f"part {n} has no audio file")
    start, end = part.get("listen_from"), part.get("listen_to")
    if start is None or end is None or end <= start:
        raise ConvertError(f"part {n} has no usable listen_from/listen_to ({start}, {end})")
    transcript, block_map = convert_paragraphs(part, labels=False)
    section = {
        "title": f"Part {n}",
        "audio_url": localize_audio(part["file_id"]),
        "section_start_time": 0,
        "section_end_time": end - start,
        "transcript": transcript,
        "question_groups": groups,
    }
    return section, transcript, block_map


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
    ap.add_argument("--image-dir", type=Path, help="download diagram images here (e.g. internal/assets/diagrams) instead of hotlinking")
    ap.add_argument("--audio-dir", type=Path, help="download listening recordings here (e.g. media/audio) instead of hotlinking")
    args = ap.parse_args()
    global IMAGE_DIR, AUDIO_DIR
    IMAGE_DIR = args.image_dir
    AUDIO_DIR = args.audio_dir

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
        kind, units = next(iter(body["content_data"].items()))
        n_q = sum(len(g["questions"]) for u in units for g in u["question_groups"])
        print(f"wrote {args.output} ({len(units)} {kind}, {n_q} questions)", file=sys.stderr)
    else:
        sys.stdout.write(out)


if __name__ == "__main__":
    main()

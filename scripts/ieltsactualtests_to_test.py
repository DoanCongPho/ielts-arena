#!/usr/bin/env python3
"""Convert an ieltsactualtests.com reading or listening test page into a
POST /api/tests request body, filed under the "actual" series (real exams
recalled by candidates).

The site is a Next.js app: the whole test — passage / notes, questions,
options and the answer key — is in the page's RSC flight payload
(self.__next_f.push chunks). The rendered tree gives the text; a client
component's `items` prop gives, per question number, the site's
questionType, optionKeys and accepted answers.

What the page doesn't have, and how it's filled in:
  - Reading group instructions: written from the question type.
  - Listening section timings: the site has one recording per test, so
    its length (ffprobe) is split evenly across the parts.
  - Explanations / evidence: none.

The book fields come from the page title ("Reading 2026-09 Test 1" ->
series actual, volume 202609, test_number 1).

Stdlib only (plus ffprobe for listening). Usage:

    python3 scripts/ieltsactualtests_to_test.py PAGE.html -o OUT.json
        [--audio-dir media/audio]

With --audio-dir the recording is downloaded there and referenced as
/assets/audio/<file>; otherwise the site's CDN URL is used.
"""

import argparse
import json
import re
import subprocess
import sys
import urllib.request
from pathlib import Path

GAP = "{{gap}}"
_BLANK = re.compile(r"_{3,}|…{2,}|\.{4,}")
_QUOTES = str.maketrans({"‘": "'", "’": "'", "“": '"', "”": '"', "\xa0": " "})


class ConvertError(Exception):
    pass


def clean(s):
    return re.sub(r"\s+", " ", (s or "").translate(_QUOTES)).strip()


# ---------------------------------------------------------------------------
# RSC payload
# ---------------------------------------------------------------------------


def rsc_rows(html):
    """The flight payload as {row id: value}. Text rows (T<len>,) hold raw
    strings; everything else is JSON."""
    chunks = re.findall(r'self\.__next_f\.push\(\[1,(".*?")\]\)</script>', html, re.S)
    data = "".join(json.loads(c) for c in chunks)
    rows, i, n = {}, 0, len(data)
    row_id = re.compile(r"([0-9a-f]+):")
    while i < n:
        m = row_id.match(data, i)
        if not m:
            nl = data.find("\n", i)
            i = n if nl < 0 else nl + 1
            continue
        key, i = m.group(1), m.end()
        if data.startswith("T", i):
            comma = data.index(",", i)
            size = int(data[i + 1 : comma], 16)  # in UTF-8 bytes
            i = comma + 1
            text = data[i:].encode("utf-8")[:size].decode("utf-8", "ignore")
            rows[key] = text
            i += len(text)
            continue
        nl = data.find("\n", i)
        nl = n if nl < 0 else nl
        raw, i = data[i:nl], nl + 1
        try:
            rows[key] = json.loads(raw)
        except ValueError:
            rows[key] = raw
    return rows


def resolver(rows):
    memo = {}

    def resolve(v, depth=0):
        if depth > 300:
            return None
        if isinstance(v, str) and v.startswith("$"):
            if v == "$undefined":
                return None
            m = re.fullmatch(r"\$L?([0-9a-f]+)", v)
            if m and m.group(1) in rows:
                k = m.group(1)
                if k not in memo:
                    memo[k] = None  # cycle guard
                    memo[k] = resolve(rows[k], depth + 1)
                return memo[k]
            return v
        if isinstance(v, list):
            return [resolve(x, depth + 1) for x in v]
        if isinstance(v, dict):
            return {k: resolve(x, depth + 1) for k, x in v.items()}
        return v

    return resolve


def is_el(node):
    return isinstance(node, list) and len(node) == 4 and node[0] == "$" and isinstance(node[3], dict)


def walk(node):
    """Every element under node, in document order."""
    if is_el(node):
        yield node
        yield from walk(node[3].get("children"))
    elif isinstance(node, list):
        for x in node:
            yield from walk(x)


def text_of(node):
    if node is None or isinstance(node, bool):
        return ""
    if isinstance(node, (str, int, float)):
        return str(node)
    if is_el(node):
        return text_of(node[3].get("children"))
    if isinstance(node, list):
        return "".join(text_of(x) for x in node)
    return ""


def children(el):
    ch = el[3].get("children")
    out = []

    def flat(x):
        if is_el(x):
            out.append(x)
        elif isinstance(x, list):
            for y in x:
                flat(y)
        elif isinstance(x, str) and x.strip():
            out.append(x)

    flat(ch)
    return out


def cls(el):
    return el[3].get("className") or "" if is_el(el) else ""


# ---------------------------------------------------------------------------
# Page
# ---------------------------------------------------------------------------


def load_page(html):
    rows = rsc_rows(html)
    resolve = resolver(rows)
    sections, items, title = {}, None, None
    for value in rows.values():
        tree = resolve(value)
        for el in walk(tree):
            p = el[3]
            sid = p.get("id") or ""
            if el[1] == "section" and re.fullmatch(r"(passage|part)-\d+", sid) and sid not in sections:
                sections[sid] = el
            if items is None and isinstance(p.get("items"), list) and p.get("slug"):
                items = p["items"]
    m = re.search(r"<title>([^<]*)</title>", html)
    title = m.group(1) if m else ""
    audio = re.search(r'https://cdn\.ieltsactualtests\.com/listening/[^"\\]+\.mp3', html)
    if not items:
        raise ConvertError("no answer key (items) in the page")
    ordered = sorted(sections.values(), key=lambda e: int(e[3]["id"].split("-")[1]))
    return ordered, {i["number"]: i for i in items}, title, audio.group(0) if audio else None


def _qid(el):
    m = re.fullmatch(r"q(\d+)", (el[3].get("id") or "") if is_el(el) else "")
    return int(m.group(1)) if m else None


def _line_with_gaps(el):
    """A box line's text, with each inline question span ("<span id=q7>7</span>
    _____") turned into {{gap}}; returns (text, [question numbers])."""
    parts, nums = [], []

    def go(node):
        if is_el(node):
            n = _qid(node)
            if n is not None and node[1] == "span":
                nums.append(n)
                parts.append("\x00")
                return
            go(node[3].get("children"))
        elif isinstance(node, list):
            for x in node:
                go(x)
        elif isinstance(node, (str, int, float)) and not isinstance(node, bool):
            parts.append(str(node))

    go(el)
    text = clean("".join(parts))
    text = re.sub(r"\x00\s*(?:_{3,}|…{2,}|\.{4,})?", GAP, text)
    return text, nums


def parse_section(sec):
    """Flattens a section into events in document order:
    ("title", text), ("instr", text), ("img", src, alt), ("passage", paragraphs),
    ("box", box id, inline?, lines), ("q", n, stem, options, box id).
    A box line is (text with {{gap}}s, [question numbers], heading?)."""
    events = []
    box_counter = [0]

    def options_of(el):
        for ul in walk(el):
            if ul[1] != "ul":
                continue
            opts = []
            for li in children(ul):
                if not is_el(li):
                    continue
                parts = children(li)
                key = clean(text_of(parts[0])).rstrip(".") if parts else ""
                txt = clean("".join(text_of(x) for x in parts[1:]))
                if key:
                    opts.append({"id": key, "text": txt})
            return opts
        return []

    def stem_of(el):
        for p in walk(el):
            if p[1] == "p":
                # Some pages repeat the number: "20. The fence took..."
                return re.sub(r"^\d+(\s*[-–]\s*\d+)?\.\s+", "", clean(text_of(p)))
        return ""

    def visit(el):
        if not is_el(el):
            return
        tag, p, c = el[1], el[3], cls(el)
        if tag == "h2":
            events.append(("title", clean(text_of(el))))
            return
        if tag == "p" and "text-sm font-medium" in c:
            events.append(("instr", clean(text_of(el))))
            return
        if tag == "img":
            events.append(("img", p.get("src"), p.get("alt") or ""))
            return
        if tag == "li" and _qid(el) is not None:
            events.append(("q", _qid(el), stem_of(el), options_of(el), None))
            return
        if tag == "div" and "rounded-lg" in c and "bg-muted" in c:
            kids = [k for k in children(el) if is_el(k)]
            if not any(_qid(x) is not None for k in kids for x in walk(k)):
                events.append(("passage", passage_paragraphs(kids)))
                return
            box_counter[0] += 1
            box = box_counter[0]
            inline = not any(_qid(k) is not None for k in kids)
            lines, qs, seen = [], [], set()
            for k in kids:
                kc = cls(k)
                if "h-3" in kc.split() or not clean(text_of(k)):
                    continue
                n = _qid(k)
                if n is not None and n in seen:
                    # A stray repeat of an earlier question's id: plain text.
                    t = clean(text_of(k))
                    lines.append((re.sub(r"^\d*\.\s*", "", t), [], False))
                    continue
                if n is not None:
                    seen.add(n)
                    # A whole-line question: the number span renders as ".".
                    t = re.sub(r"^\d*\.\s*", "", clean(text_of(k)))
                    stem = t
                    t = _BLANK.sub(GAP, t, count=1)
                    t = _BLANK.sub("___", t)
                    if GAP not in t:
                        t = f"{t} {GAP}"
                    lines.append((t, [n], False))
                    qs.append((n, stem))
                else:
                    t, nums = _line_with_gaps(k)
                    heading = not nums and "font-semibold" in kc
                    lines.append((t, nums, heading))
                    qs.extend((n, "") for n in nums)
            events.append(("box", box, inline, lines))
            for n, stem in qs:
                events.append(("q", n, stem, [], box))
            return
        for k in children(el):
            visit(k)

    visit(sec)
    return events


def passage_paragraphs(kids):
    paragraphs, label = [], ""
    for k in kids:
        t = clean(text_of(k))
        if not t or "h-3" in cls(k).split():
            continue
        if "font-semibold" in cls(k) and re.fullmatch(r"(Paragraph |Section )?[A-Z]", t):
            label = t[-1]
            continue
        m = re.match(r"^\(?([A-Z])\)\s+", t) or re.match(r"^([A-Z])[.:]\s+(?=[A-Z])", t)
        if m and not label:
            label, t = m.group(1), t[m.end():]
        paragraphs.append({"label": label, "text": t})
        label = ""
    return paragraphs


# ---------------------------------------------------------------------------
# Questions
# ---------------------------------------------------------------------------

TFNG = {"TRUE", "FALSE", "NOT GIVEN"}
YNNG = {"YES", "NO", "NOT GIVEN"}
SHORT = {"T": "TRUE", "F": "FALSE", "NG": "NOT GIVEN", "Y": "YES", "N": "NO"}
LETTER_TYPES = {"matching", "map_labelling", "multiple_choice", "matching_information",
                "matching_features", "matching_headings", "heading_matching"}

READING_INSTR = {
    "true-false-not-given": "Do the following statements agree with the information given in the passage? Write TRUE, FALSE or NOT GIVEN.",
    "yes-no-not-given": "Do the following statements agree with the views of the writer? Write YES, NO or NOT GIVEN.",
    "multiple-choice": "Choose the correct letter.",
    "multiple-choice-multi": "Choose TWO letters.",
    "matching-headings": "Choose the correct heading for each paragraph from the list of headings below.",
    "matching-information": "Which paragraph contains the following information? You may use any letter more than once.",
    "matching-features": "Match each statement with the correct option. Write the correct letter.",
    "sentence-completion": "Complete the sentences below. Choose NO MORE THAN THREE WORDS from the passage for each answer.",
    "short-answer": "Answer the questions below. Choose NO MORE THAN THREE WORDS from the passage for each answer.",
    "note-completion": "Complete the notes below. Choose NO MORE THAN THREE WORDS from the passage for each answer.",
    "summary-completion": "Complete the summary below. Choose NO MORE THAN THREE WORDS from the passage for each answer.",
}
LISTENING_INSTR = {
    "multiple-choice": "Choose the correct letter, A, B or C.",
    "multiple-choice-multi": "Choose TWO letters.",
    "matching": "Choose the correct letter for each question.",
    "map-plan-labelling": "Label the map/plan below. Write the correct letter next to each question.",
    "sentence-completion": "Complete the sentences below. Write NO MORE THAN TWO WORDS AND/OR A NUMBER for each answer.",
    "note-completion": "Complete the notes below. Write NO MORE THAN TWO WORDS AND/OR A NUMBER for each answer.",
    "summary-completion": "Complete the summary below. Write NO MORE THAN TWO WORDS AND/OR A NUMBER for each answer.",
}


def inline_options(stem, keys):
    """'...becauseA. walkers...B. birds...' -> ('...because', options) when
    every key appears in order as "K. "."""
    pos, cuts = 0, []
    for k in keys:
        i = stem.find(f"{k}. ", pos)
        if i < 0:
            return stem, []
        cuts.append(i)
        pos = i + len(k) + 2
    opts = []
    for j, k in enumerate(keys):
        start = cuts[j] + len(k) + 2
        end = cuts[j + 1] if j + 1 < len(keys) else len(stem)
        opts.append({"id": k, "text": clean(stem[start:end])})
    return clean(stem[: cuts[0]]), opts


def is_letter(a):
    return bool(re.fullmatch(r"[A-Z]|[ivxlc]+", a))


def classify(skill, q, item, shared, has_image):
    """Our question_type, from what the question shows (its options and
    key) — the site's questionType is often wrong (a whole passage typed
    "tfng"), so it only breaks ties."""
    site = item.get("questionType") or ""
    keys = item.get("optionKeys") or []
    answers = item.get("answers") or []
    opts = q["options"] or shared
    texts = {o["text"].upper() for o in opts}
    # "F" is a letter option unless the site says the question is TFNG.
    short = site in ("tfng", "ynng") and not opts
    upper = {SHORT.get(a.upper(), a.upper()) if short else a.upper() for a in answers}
    letters = bool(answers) and all(is_letter(a) for a in answers) and not (short and upper <= TFNG | YNNG)
    if texts == TFNG or (not keys and upper and upper <= TFNG and (upper & {"TRUE", "FALSE"} or site == "tfng")):
        return "true-false-not-given"
    if texts == YNNG or (not keys and upper and upper <= YNNG and (upper & {"YES", "NO"} or site == "ynng")):
        return "yes-no-not-given"
    if letters and (keys or opts or site in LETTER_TYPES):
        if len(set(answers)) > 1:
            return "multiple-choice-multi"
        if all(re.fullmatch(r"[ivxlc]+", a) for a in answers):
            return "matching-headings"
        if q["box"] is not None and not q["options"]:
            return "box-letter"
        if q["options"] and not shared_run(q):
            return "multiple-choice"
        if skill == "listening" and (site == "map_labelling" or (not opts and has_image)):
            return "map-plan-labelling"
        if skill == "reading" and (site == "matching_information" or all(o["text"] == o["id"] for o in opts)):
            return "matching-information"
        return "matching" if skill == "listening" else "matching-features"
    if q["box"] is not None:
        return "box-gap"
    if skill == "reading" and not _BLANK.search(q["stem"]):
        return "short-answer"
    return "sentence-completion"


def shared_run(q):
    return q.get("_shared_run", False)


def answer_text(answers, opts):
    """TFNG / YNNG answers as words, whether the key gives words, short
    forms or the letter of a TRUE/FALSE/NOT GIVEN option."""
    a = answers[0]
    by_id = {o["id"]: o["text"].upper() for o in opts}
    if a in by_id and by_id[a] in TFNG | YNNG:
        return by_id[a]
    return SHORT.get(a.upper(), a.upper())


def _relabel(p):
    # passage_paragraphs only takes "A. " as a label before a capital.
    m = re.match(r"^([A-Z]|[ivx]+)\.\s+", p["text"])
    return {"label": m.group(1), "text": p["text"][m.end():]} if m else p


def option_box(paragraphs):
    """Options from a box of lettered lines (an untitled heading line is
    allowed first), or None when the box is something else."""
    paragraphs = [p if p["label"] else _relabel(p) for p in paragraphs]
    if paragraphs and not paragraphs[0]["label"]:
        paragraphs = paragraphs[1:]
    if len(paragraphs) < 2 or any(not re.fullmatch(r"[A-Z]|[ivx]+", p["label"]) or len(p["text"]) > 200 for p in paragraphs):
        return None
    return [{"id": p["label"], "text": p["text"]} for p in paragraphs]


def _norm(t):
    return re.sub(r"\s+", " ", _BLANK.sub("___", t)).strip()


def claim_box_line(boxes, n, stem):
    """Some pages print a blank as plain text in the box ("Rental for one
    week: 6 £_____") and list its question again under the box. Puts the
    gap back on that box line and returns the box, or None."""
    for box, (_, lines) in boxes.items():
        for i, (text, nums, heading) in enumerate(lines):
            if nums or heading or not _BLANK.search(text) or _norm(text) != _norm(stem):
                continue
            # Drop the printed question number before the blank, and the
            # next one's number left dangling at the end of the line.
            t = re.sub(rf"(^|\s){n}\s+(?=\S?\s*(?:_{{3,}}|…{{2,}}|\.{{4,}}))", r"\1", text)
            t = re.sub(rf"\s+{n + 1}$", "", t)
            lines[i] = (_BLANK.sub(GAP, t, count=1), [n], False)
            return box
    return None


def build_groups(skill, events, items):
    """Question groups for one passage / section, from its events."""
    questions, boxes = [], {}
    instr, image, has_image = None, None, False
    box_opts = None
    for ev in events:
        if ev[0] == "instr":
            instr = ev[1]
        elif ev[0] == "passage" and instr is not None:
            # A muted box after the instructions listing "A. ...", "B. ..."
            # is the option box the next questions answer from.
            opts = option_box(ev[1])
            if opts:
                box_opts = opts
        elif ev[0] == "img":
            image, has_image = ev[1], True
        elif ev[0] == "box":
            boxes[ev[1]] = (ev[2], ev[3])
        elif ev[0] == "q":
            opts, box = ev[3], ev[4]
            if box_opts and not opts and box is None:
                opts, box_opts = box_opts, None
            if box is None and not opts:
                box = claim_box_line(boxes, ev[1], ev[2])
            questions.append({"n": ev[1], "stem": ev[2], "options": opts, "box": box,
                              "instr": instr, "img": image})
    questions.sort(key=lambda q: q["n"])

    # Matching layouts print the options on the first question only; MCQs
    # print them on every question.
    for i, q in enumerate(questions):
        nxt = questions[i + 1] if i + 1 < len(questions) else None
        q["_shared_run"] = bool(q["options"]) and nxt is not None and not nxt["options"] and \
            (items.get(nxt["n"], {}).get("optionKeys") or []) == [o["id"] for o in q["options"]]

    shared, prev_keys = [], None
    for q in questions:
        item = items.get(q["n"]) or {}
        keys = item.get("optionKeys") or []
        if q["box"] is not None and keys and not q["options"]:
            stem, opts = inline_options(q["stem"], keys)
            if opts:
                q["stem"], q["options"] = stem, opts
        answers = set(item.get("answers") or [])
        if q["options"]:
            shared = q["options"]
        elif not (shared and answers <= {o["id"] for o in shared}) and keys != prev_keys:
            shared = []
        prev_keys = keys
        q["item"] = item
        q["type"] = classify(skill, q, item, shared, q["img"] is not None)
        q["shared"] = q["options"] or shared or [{"id": k, "text": k} for k in keys]

    # Gap questions in a box come out as note/summary groups, split where a
    # question of another kind sits in the box.
    groups = []
    for q in questions:
        t = q["type"]
        sig = (t, q["box"], q["instr"], None if t == "multiple-choice" else tuple(o["id"] for o in q["shared"]))
        if groups and groups[-1]["sig"] == sig:
            groups[-1]["qs"].append(q)
        else:
            groups.append({"sig": sig, "qs": [q]})
    return [make_group(skill, g["qs"], boxes) for g in groups]


def box_segment(box, members):
    """The box lines for a group of its questions: from just after the last
    line holding some other question before the group, to just before the
    next such line after it."""
    inline, lines = box
    idx = [i for i, (_, nums, _) in enumerate(lines) if set(nums) & members]
    first, last = idx[0], idx[-1]
    foreign = lambda i: lines[i][1] and not set(lines[i][1]) <= members
    lo = first
    while lo > 0 and not foreign(lo - 1):
        lo -= 1
    hi = last
    while hi + 1 < len(lines) and not foreign(hi + 1):
        hi += 1
    return inline, lines[lo : hi + 1]


def make_group(skill, qs, boxes):
    qtype = qs[0]["type"]
    shared = qs[0]["shared"]
    members = {q["n"] for q in qs}
    group = {}

    if qtype == "map-plan-labelling" and not qs[0]["img"]:
        qtype = "matching"  # the page has no map for it
    if qtype == "box-letter" and not shared:
        # A labelling task whose diagram and word box the page doesn't
        # show: answer by letter, from the range the instructions give.
        qtype = "matching" if skill == "listening" else "matching-features"
        m = re.search(r"\b([A-Z])\s*[–-]\s*([A-Z])\b", qs[0]["instr"] or "")
        top = m.group(2) if m else max(a for q in qs for a in q["item"].get("answers") or ["A"])
        shared = [{"id": chr(c), "text": chr(c)} for c in range(ord("A"), ord(top) + 1)]
        for q in qs:
            q["stem"] = f"Label {q['n']}"
    if qtype in ("box-gap", "box-letter"):
        inline, seg = box_segment(boxes[qs[0]["box"]], members)
        rendered = []
        for text, nums, heading in seg:
            if nums and not set(nums) <= members:
                continue
            rendered.append(("## " + text) if heading else text)
        word_bank = qtype == "box-letter"
        if inline or word_bank:
            qtype = "summary-completion"
            group["summary_text"] = "\n".join(rendered)
            if word_bank:
                group["has_word_bank"] = True
                group["word_bank"] = shared
        else:
            qtype = "note-completion"
            group["note_structure"] = {"items": rendered}

    instructions = qs[0]["instr"] or (LISTENING_INSTR if skill == "listening" else READING_INSTR).get(qtype) or "Answer the questions below."
    group = {"question_type": qtype, "instructions": instructions, **group, "questions": []}

    if qtype in ("matching-headings", "matching-information", "matching-features", "matching"):
        opts = shared
        if qtype == "matching-information" and all(o["text"] == o["id"] for o in opts):
            opts = [{"id": o["id"], "text": f"Paragraph {o['id']}"} for o in opts]
        if not opts:
            top = max(a for q in qs for a in q["item"].get("answers") or ["A"] if len(a) == 1)
            opts = [{"id": chr(c), "text": chr(c)} for c in range(ord("A"), ord(top) + 1)]
        group["shared_options"] = opts
        if qtype in ("matching-information", "matching-features"):
            group["allow_reuse"] = True
    if qtype == "map-plan-labelling":
        group["map_image_url"] = qs[0]["img"]
        keys = [o["id"] for o in shared] if shared else []
        if not keys:
            top = max(max(q["item"].get("answers") or ["A"]) for q in qs)
            keys = [chr(c) for c in range(ord("A"), max(ord(top), ord("G")) + 1)]
        group["location_key"] = [{"id": k, "text": k} for k in keys]
    if qtype == "multiple-choice-multi":
        group["select_count"] = 2

    skip = set()
    for q in qs:
        if q["n"] in skip:
            continue
        answers = [clean(a) for a in q["item"].get("answers") or []]
        if not answers:
            raise ConvertError(f"question {q['n']} has no answer")
        entry = {"question_order": q["n"]}
        if qtype in ("true-false-not-given", "yes-no-not-given"):
            entry["text"] = q["stem"]
            entry["answer"] = answer_text(answers, q["options"] or shared)
        elif qtype == "multiple-choice":
            entry["text"] = q["stem"]
            entry["options"] = q["options"] or shared
            entry["answer"] = answers[0]
        elif qtype == "multiple-choice-multi":
            entry["text"] = q["stem"]
            entry["options"] = q["options"] or shared
            entry["answer"] = sorted(set(answers))[:2]
            # The site lists a choose-TWO pair either once or once per
            # number with the same key; one question spans both numbers.
            skip.add(q["n"] + 1)
        elif qtype in ("matching-headings", "matching-information", "matching-features", "matching", "map-plan-labelling"):
            entry["text"] = _BLANK.sub("", q["stem"]).strip(" :–-") or f"Question {q['n']}"
            entry["answer"] = answers[0]
        elif qtype == "summary-completion" and group.get("has_word_bank"):
            entry["answer"] = answers[0]
        elif qtype in ("note-completion", "summary-completion"):
            entry["answer"] = answers[0]
            entry["accepted_answers"] = answers
        else:
            stem = q["stem"]
            if qtype == "sentence-completion":
                stem = _BLANK.sub("__", stem, count=1)
                stem = _BLANK.sub("___", stem)
                if "__" not in stem:
                    stem = f"{stem} __"
            entry["text"] = stem
            entry["answer"] = answers[0]
            entry["accepted_answers"] = answers
        group["questions"].append(entry)
    return group


# ---------------------------------------------------------------------------
# Test
# ---------------------------------------------------------------------------


def section_title(t, skill):
    t = re.sub(r"^(Reading Passage \d+|Passage \d+|Part \d+|Section \d+|Questions? \d+\s*[-–]\s*\d+)\s*[:–—-]\s*", "", t)
    return t


def renumber(content, skill):
    """question_order must run 1..N; a recalled test sometimes skips a
    number. Close the gaps, keeping spans."""
    units = content["passages"] if skill == "reading" else content["sections"]
    n = 1
    for u in units:
        for g in u["question_groups"]:
            g_span = g.get("select_count", 1) if g["question_type"] == "multiple-choice-multi" else 1
            for q in g["questions"]:
                q["question_order"] = n
                n += g_span
    return n - 1


def audio_seconds(path_or_url):
    out = subprocess.run(
        ["ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "csv=p=0", path_or_url],
        capture_output=True, text=True, check=True,
    )
    return float(out.stdout.strip())


def convert(html, skill, slug, audio_dir=None):
    sections, items, title, audio = load_page(html)
    m = re.search(r"(Reading|Listening) (\d{4})-(\d{2}) Test (\d+)", title)
    if not m:
        raise ConvertError(f"can't read the test name from title {title!r}")
    volume, test_number = int(m.group(2) + m.group(3)), int(m.group(4))

    # Some pages put each question group in a section of its own
    # ("Questions 1-6: ...") after the passage section; fold those back
    # into the passage before them.
    merged = []
    for sec in sections:
        events = parse_section(sec)
        name = next((e[1] for e in events if e[0] == "title"), "")
        if skill == "reading" and merged and re.match(r"Questions?\s+\d", name):
            merged[-1][1].extend(e for e in events if e[0] != "title")
        else:
            merged.append((name, events))

    units = []
    for name, events in merged:
        groups = build_groups(skill, events, items)
        for i, g in enumerate(groups, 1):
            g["group_order"] = i
        if skill == "reading":
            paragraphs = next((e[1] for e in events if e[0] == "passage"), None)
            if not paragraphs:
                raise ConvertError(f"{name!r}: no passage text")
            units.append({"title": section_title(name, skill), "paragraphs": paragraphs, "question_groups": groups})
        else:
            units.append({"title": section_title(name, skill), "question_groups": groups})

    content = {"passages": units} if skill == "reading" else {"sections": units}
    renumber(content, skill)

    if skill == "listening":
        if not audio:
            raise ConvertError("no recording on the page")
        if audio_dir:
            dest = Path(audio_dir) / f"iat-{slug}.mp3"
            if not dest.exists():
                req = urllib.request.Request(audio, headers={"User-Agent": "Mozilla/5.0"})
                with urllib.request.urlopen(req) as resp:
                    dest.write_bytes(resp.read())
            seconds = audio_seconds(str(dest))
            content["audio_url"] = f"/assets/audio/{dest.name}"
        else:
            seconds = audio_seconds(audio)
            content["audio_url"] = audio
        # No part timings on the site: split the recording evenly.
        per = seconds / len(units)
        for i, u in enumerate(units):
            u["section_start_time"] = round(i * per, 1)
            u["section_end_time"] = round(seconds if i == len(units) - 1 else (i + 1) * per, 1)
        ordered = ["audio_url", "sections"]
        content = {k: content[k] for k in ordered}
        for u in units:
            for k in ("question_groups",):
                u[k] = u.pop(k)

    return {
        "skill": skill,
        "task_type": f"test{test_number}",
        "series": "actual",
        "volume": volume,
        "test_number": test_number,
        "source": "ieltsactualtests",
        "is_current": True,
        "xp_gain": 50,
        "content_data": content,
    }


def apply_fixes(body, fixes):
    """Hand fixes for what a page gets too wrong to convert: keys
    "<section>.<group>" (1-based) map to group fields to set, null to
    drop; keys starting with "_" are notes."""
    c = body["content_data"]
    units = c.get("passages") or c.get("sections")
    for key, fields in fixes.items():
        if key.startswith("_"):
            continue
        u, g = (int(x) - 1 for x in key.split("."))
        group = units[u]["question_groups"][g]
        for k, v in fields.items():
            if v is None:
                group.pop(k, None)
            else:
                group[k] = v


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("page", help="saved page, named <skill>_<slug>.html")
    ap.add_argument("-o", "--output")
    ap.add_argument("--audio-dir")
    ap.add_argument("--fixes", help="directory of hand fixes, <skill>_<slug>.json each (see apply_fixes)")
    args = ap.parse_args()
    name = Path(args.page).stem
    skill, _, slug = name.partition("_")
    if skill not in ("reading", "listening"):
        sys.exit(f"{args.page}: name the file <reading|listening>_<slug>.html")
    try:
        body = convert(Path(args.page).read_text(), skill, slug, args.audio_dir)
    except ConvertError as e:
        sys.exit(f"{args.page}: {e}")
    fix = Path(args.fixes or "", f"{name}.json")
    if args.fixes and fix.exists():
        apply_fixes(body, json.loads(fix.read_text()))
    out = json.dumps(body, ensure_ascii=False, indent=1)
    if args.output:
        Path(args.output).write_text(out + "\n")
    else:
        print(out)


if __name__ == "__main__":
    main()

#!/usr/bin/env python3
"""Convert IELTS Mate reading / listening rooms into POST /api/tests request bodies.

IELTS Mate serves a reading room at GET /api/study-room/room/<roomId>; its
`content` is the YouPass quiz shape (parts[].vocabs / question_sets /
locate_info / explain), one passage per room. This script joins a test's
three passage rooms into one test and hands it to youpass_to_test.convert.

Listening rooms are one section each, with the section's recording
(audio_url), its transcript as one string and questions' audio_time_range;
a test's four sections are joined the same way.

Input layout (as downloaded per volume):

    tests_bank/ieltsmate/vol-10/test-01/passage-{1,2,3}.json
    tests_bank/ieltsmate/listening/vol-10/test-01/section-{1,2,3,4}.json

Output: one body per test, OUT_DIR/vol-10_test-01.json, filed under series
"vol" (IELTS Recent Actual Tests, "VOL 10"), volume 10, test_number 1.

Stdlib only. Usage:

    python3 scripts/ieltsmate_to_test.py tests_bank/ieltsmate -o tests_bank/ieltsmate/tests
        [--image-dir internal/assets/diagrams]
    python3 scripts/ieltsmate_to_test.py tests_bank/ieltsmate/listening --skill listening
        -o tests_bank/ieltsmate/listening-tests --image-dir internal/assets/diagrams --audio-dir media/audio

(ffprobe measures a recording's length when the room doesn't give it.)
"""

import argparse
import json
import math
import re
import subprocess
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import youpass_to_test as yp  # noqa: E402

SERIES = "vol"

# ---------------------------------------------------------------------------
# Where IELTS Mate differs from YouPass
# ---------------------------------------------------------------------------
#
# 1. Gap-fill questions come as FILL_IN_THE_BLANK (and one diagram set as
#    DIAGRAM_LABEL_COMPLETION) for every layout; the set's instructions say
#    which layout it is.
# 2. Fill-in keys are often one string holding the book's variants:
#    "colours / colors", "(the) bony carapace".
# 3. locate_info is {paragraph: "<1-based vocabs block>", source_excerpt,
#    evidence, status}, not YouPass's word ranges.


def fill_type(qset):
    """Our question_type for an IELTS Mate gap-fill set."""
    instr = yp.clean(yp.html_text(qset.get("description"))).lower()
    content = qset.get("content") or ""
    has_table = max(
        (len(yp.table_row_cells(r)) for r in yp.table_rows(content) if "gap-placeholder" in r), default=0
    ) >= 2
    has_image = "<img" in content
    if has_image and ("label the" in instr or "diagram" in instr):
        return "diagram-label-completion"
    if "flow" in instr and "chart" in instr:
        return "flow-chart-completion"
    gap_blocks = [t for _, t in yp.html_blocks(re.sub(
        r'<span[^>]*class="gap-placeholder"[^>]*>.*?</span>', yp.GAP, content, flags=re.S)) if yp.GAP in t]
    one_per_block = len(gap_blocks) == len(qset["questions"]) and all(t.count(yp.GAP) == 1 for t in gap_blocks)
    # The instructions name the layout, but IELTS Mate sometimes lays the
    # content out differently ("Complete the table" over plain notes, three
    # "sentences" run together in one paragraph): follow the content then.
    if has_table:
        return "table-completion"
    if "complete the table" in instr or "complete the notes" in instr:
        return "note-completion"
    if "complete the sentence" in instr or "complete each sentence" in instr:
        return "sentence-completion" if one_per_block else "summary-completion"
    if "answer the question" in instr:
        return "short-answer" if one_per_block else "note-completion"
    if "summary" in instr:
        return "summary-completion"
    # No "Complete the ..." line: one gap per block reads as sentences.
    if one_per_block:
        return "sentence-completion"
    return "note-completion" if "<li" in content else "summary-completion"


def listening_fill_type(qset):
    """Our question_type for an IELTS Mate listening gap-fill set. Listening
    has no short-answer or diagram-label type: "Answer the questions" reads
    as sentences, a labelled diagram with typed answers as notes."""
    instr = yp.clean(yp.html_text(qset.get("description"))).lower()
    content = qset.get("content") or ""
    has_table = max(
        (len(yp.table_row_cells(r)) for r in yp.table_rows(content) if "gap-placeholder" in r), default=0
    ) >= 2
    gap_blocks = [t for _, t in yp.html_blocks(re.sub(
        r'<span[^>]*class="gap-placeholder"[^>]*>.*?</span>', yp.GAP, content, flags=re.S)) if yp.GAP in t]
    one_per_block = len(gap_blocks) == len(qset["questions"]) and all(t.count(yp.GAP) == 1 for t in gap_blocks)
    if has_table:
        return "table-completion"
    if "flow" in instr and "chart" in instr:
        return "flow-chart-completion"
    if "complete the form" in instr:
        return "form-completion"
    if "sentence" in instr or "answer the question" in instr:
        return "sentence-completion" if one_per_block else "note-completion"
    if "summary" in instr:
        return "summary-completion"
    return "note-completion"


_youpass_group_type = yp.group_type


def group_type(qset, skill):
    if qset.get("_arena_type"):
        return qset["_arena_type"]
    qtype = _youpass_group_type(qset, skill)
    # IELTS Mate tags some sentence-ending sets MATCHING_INFO.
    instr = yp.clean(yp.html_text(qset.get("description"))).lower()
    if qtype == "matching-information" and "correct ending" in instr:
        return "matching-sentence-endings"
    # Listening has one matching type.
    if skill == "listening" and qtype.startswith("matching"):
        return "matching"
    return qtype


def expand_variants(answer):
    """ "colours / colors" -> both; "(the) soil" -> "soil", "the soil"."""
    out = []
    for alt in re.split(r"\s*/\s*", answer or ""):
        alt = alt.strip()
        if not alt:
            continue
        if "(" in alt:
            without = re.sub(r"\s+", " ", re.sub(r"\([^)]*\)", "", alt)).strip()
            with_ = re.sub(r"\s+", " ", re.sub(r"[()]", "", alt)).strip()
            out += [without, with_]
        else:
            out.append(alt)
    seen, uniq = set(), []
    for a in out:
        if a and a.lower() not in seen:
            seen.add(a.lower())
            uniq.append(a)
    return uniq


def normalize_answers(q):
    if q.get("question_type") not in ("FILL_IN_THE_BLANK", "SUMMARY_COMPLETION", "DIAGRAM_LABEL_COMPLETION"):
        return
    keys = q.get("correct_answers") or [q.get("correct_answer")]
    expanded = [v for k in keys if k for v in expand_variants(yp.clean(k))]
    if expanded:
        q["correct_answers"] = expanded


def _norm(s):
    return re.sub(r"\s+", " ", yp.clean(s)).strip()


def convert_evidence(q, paragraphs, block_map):
    """{paragraph, quote} from locate_info.source_excerpt: the excerpt (or,
    when it was trimmed with "...", its longest piece) as it appears in the
    paragraph locate_info names, else in any paragraph."""
    li = q.get("locate_info")
    if not isinstance(li, dict) or li.get("status") == "verified_absence_not_given":
        return []
    excerpt = _norm(li.get("source_excerpt"))
    if not excerpt:
        return []
    pieces = sorted((p.strip(" ,;.\"'") for p in re.split(r"\s*(?:\.\.\.|…)\s*", excerpt)), key=len, reverse=True)
    pieces = [p for p in pieces if len(p) >= 12]
    order = list(range(len(paragraphs)))
    try:
        bi = int(li.get("paragraph")) - 1
        if bi in block_map:
            pi = block_map[bi][0]
            order.remove(pi)
            order.insert(0, pi)
    except (TypeError, ValueError):
        pass
    for piece in pieces:
        for pi in order:
            text = _norm(paragraphs[pi]["text"])
            idx = text.lower().find(piece.lower())
            if idx >= 0:
                return [{"paragraph": pi, "quote": text[idx: idx + len(piece)]}]
    return []


_META_LINE = re.compile(
    r"^•?\s*(normalizedQuestionType|targetType|requiredAnswerCount|wordNumberMode|mustUseSourceWords"
    r"|answerCountValid|wordLimitValid|grammarValid|spellingAndFormValid|maxWordsPerAnswer"
    r"|maxNumbersPerAnswer)\s*:"
)
_META_LABEL = re.compile(r"^(•?\s*)(connection|preservedModifiers)\s*:\s*")
_ANSWER_LINE = re.compile(r"^•?\s*\d{6,}\s*→\s*(.+)$")
_youpass_explanation = yp.convert_explanation


def convert_explanation(src):
    """YouPass's conversion, minus the checklist fields IELTS Mate's
    generator leaves in (normalizedQuestionType, grammarValid, ...) and its
    internal question ids ("890002014 → F" becomes "Đáp án: F")."""
    lines = []
    for line in _youpass_explanation(src).split("\n"):
        if _META_LINE.match(line):
            continue
        m = _ANSWER_LINE.match(line)
        if m:
            line = f"Đáp án: {m.group(1).strip()}"
        line = _META_LABEL.sub(r"\1", line)
        if line.strip():
            lines.append(line)
    return "\n".join(lines)


yp.group_type = group_type
yp.convert_evidence = convert_evidence
yp.convert_explanation = convert_explanation


def merge_choose_many(qset):
    """IELTS Mate keeps a "Choose TWO" item's options on the set and lists it
    as one question per number (20 and 21, both keyed ['A', 'D']); ours is
    one question at the first number carrying both keys and the options."""
    qs = sorted(qset["questions"], key=lambda q: q["order"])
    if all(q.get("options") for q in qs):
        return
    select = int(qset.get("max_selections") or len(qs[0].get("correct_answers") or []) or 2)
    merged = []
    for i in range(0, len(qs), select):
        chunk = qs[i: i + select]
        keys = {tuple(sorted(q.get("correct_answers") or [])) for q in chunk}
        orders = [q["order"] for q in chunk]
        if len(keys) != 1 or len(chunk) != select or orders != list(range(orders[0], orders[0] + select)):
            raise yp.ConvertError(f"set {qset.get('title')!r}: can't pair choose-{select} questions {orders}")
        first = dict(chunk[0])
        first["options"] = first.get("options") or qset.get("options")
        first["correct_answers"] = sorted(keys.pop())
        merged.append(first)
    qset["questions"] = merged


_BLOCK_HTML = re.compile(r"<(p|h[1-6]|div|li)\b", re.I)


def split_html_vocabs(part):
    """Most IELTS Mate passages hold one whole paragraph of HTML per child
    (<p data-letter="A"><strong>A.</strong> ...</p>), sometimes the whole
    passage under a single block, where YouPass has one sentence of plain
    text per child. Give every HTML paragraph a block of its own, so
    convert_paragraphs makes one paragraph each and reads the A., B. labels;
    plain-text blocks are left as they are."""
    blocks = []  # (letter from data-letter or "", block)
    for block in part.get("vocabs") or []:
        children = block.get("children") or []
        if not any(_BLOCK_HTML.search(c.get("value") or "") for c in children):
            blocks.append(("", block))
            continue
        for c in children:
            for attrs, inner in _top_level_blocks(c.get("value") or ""):
                m = re.search(r'data-letter="([A-Z])"', attrs)
                for i, (_, text) in enumerate(yp.html_blocks(inner)):
                    letter = m.group(1) if m and i == 0 else ""
                    blocks.append((letter, {"value": "", "children": [{"value": text}]}))
    part["vocabs"] = label_paragraphs(blocks)


def _top_level_blocks(src):
    """(attributes, inner HTML) of each top-level <p>; text outside any <p>
    comes back with empty attributes."""
    out, pos = [], 0
    for m in re.finditer(r"<p\b([^>]*)>(.*?)</p>", src, flags=re.S | re.I):
        if src[pos:m.start()].strip():
            out.append(("", src[pos:m.start()]))
        out.append((m.group(1), m.group(2)))
        pos = m.end()
    if src[pos:].strip():
        out.append(("", src[pos:]))
    return out


# "A. Text", "A Text", or a letter run into an opening quote ("D“Even ...").
_LEAD_LETTER = re.compile(r"^([A-Z])(?:\.?\s+(?=\S)|(?=[\"“‘']))")


def label_paragraphs(blocks):
    """Writes each section letter as "A. " at the start of its paragraph,
    which is the form convert_paragraphs reads labels from. The letter comes
    from data-letter, else from a leading "A " / "A." — but only for letters
    running A, B, C... in order, so an article ("A pantomime is...") isn't
    taken for a label."""
    def text_of(b):
        return yp.clean(" ".join(c.get("value") or "" for c in b.get("children") or []))

    marks, expected = {}, "A"
    for i, (letter, b) in enumerate(blocks):
        m = _LEAD_LETTER.match(text_of(b))
        found = letter or (m.group(1) if m else "")
        if found == expected:
            marks[i] = found
            expected = chr(ord(expected) + 1)
    if len(marks) < 2:
        marks = {}
    out = []
    for i, (_, b) in enumerate(blocks):
        children = [dict(c) for c in b.get("children") or []]
        if i in marks and children:
            first = yp.clean(children[0].get("value"))
            letter = marks[i]
            m = _LEAD_LETTER.match(first)
            if m and m.group(1) == letter:
                first = first[m.end():]
            children[0]["value"] = f"{letter}. {first}"
        elif children:
            # Not a label: make sure convert_paragraphs won't read one either.
            first = yp.clean(children[0].get("value"))
            if re.match(r"^[A-Z]\.\s", first):
                children[0]["value"] = first.replace(".", "", 1) if first[2:3] == " " else first
        out.append({**b, "children": children})
    return out


def prepare(part, skill="reading"):
    if skill == "listening":
        return prepare_section(part)
    # A few titles are HTML: "<p>Title</p><p>Subtitle</p>". The subtitle
    # becomes the passage's opening (unlabelled) paragraph.
    # Plain titles put the subtitle on the next line instead.
    title_blocks = [t for line in re.split(r"\r?\n", part.get("title") or "")
                    for _, t in yp.html_blocks(line)]
    if title_blocks:
        part["title"] = title_blocks[0]
        if len(title_blocks) > 1:
            part["vocabs"] = [{"value": "", "children": [{"value": " ".join(title_blocks[1:])}]}] + list(part.get("vocabs") or [])
    split_html_vocabs(part)
    for qset in part.get("question_sets") or []:
        if any(q.get("question_type") == "MULTIPLE_CHOICE_MANY" for q in qset["questions"]):
            merge_choose_many(qset)
        kinds = {q.get("question_type") for q in qset["questions"]}
        if kinds & {"FILL_IN_THE_BLANK", "DIAGRAM_LABEL_COMPLETION", "SUMMARY_COMPLETION"}:
            if qset.get("options"):
                # A word bank also drawn as a table in the content: the
                # options already carry it.
                qset["content"] = re.sub(
                    r"<table\b.*?</table>",
                    lambda m: m.group(0) if "gap-placeholder" in m.group(0) else "",
                    qset.get("content") or "",
                    flags=re.S | re.I,
                )
            qset["_arena_type"] = fill_type(qset)
        for q in qset["questions"]:
            normalize_answers(q)
    return part


def mmss(t):
    """ "03:05" / "1:02:03" -> seconds."""
    secs = 0.0
    for piece in str(t).split(":"):
        secs = secs * 60 + float(piece)
    return secs


def prepare_section(part):
    """A listening section in YouPass's shape: the transcription string as
    one block per line, questions' audio_time_range as listen_from, a map's
    image in the set's instructions."""
    lines = [yp.clean(l) for l in re.split(r"\r?\n", part.get("transcription") or "")]
    part["vocabs"] = [{"value": "", "children": [{"value": l}]} for l in lines if l]
    part["listen_from"] = 0
    for qset in part.get("question_sets") or []:
        kinds = {q.get("question_type") for q in qset["questions"]}
        if "MULTIPLE_CHOICE_MANY" in kinds:
            merge_choose_many(qset)
        if kinds & {"FILL_BLANK", "FILL_IN_THE_BLANK", "SUMMARY_COMPLETION"}:
            qset["_arena_type"] = listening_fill_type(qset)
        if "MAP_DIAGRAM_LABEL" in kinds and "<img" not in (qset.get("description") or ""):
            imgs = re.findall(r'<img[^>]*\ssrc="([^"]+)"', qset.get("content") or "")
            src = imgs[0] if imgs else (qset.get("image") or "")
            if src.startswith("http"):
                qset["description"] = (qset.get("description") or "") + f'<img src="{src}">'
                qset["image"] = None
        for q in qset["questions"]:
            normalize_answers(q)
            rng = (q.get("locate_info") or {}).get("audio_time_range") or {}
            if rng.get("start"):
                q["listen_from"] = mmss(rng["start"])
    return part


AUDIO_DIR = None


def localize_audio(url):
    """The section's recording, downloaded into AUDIO_DIR (unless already
    there) and served as /assets/<dir>/<file>; the IELTS Mate URL without
    AUDIO_DIR. Returns (url, local path or None)."""
    if AUDIO_DIR is None:
        return url, None
    name = re.sub(r"[^A-Za-z0-9.-]", "", url.split("?")[0].rsplit("/", 1)[-1])
    if not name.endswith(".mp3"):
        name += ".mp3"
    path = AUDIO_DIR / name
    if not path.exists() or path.stat().st_size == 0:
        req = yp.urllib.request.Request(url, headers={"User-Agent": "ielts-arena-import"})
        with yp.urllib.request.urlopen(req, timeout=300) as resp:
            ctype = resp.headers.get_content_type()
            if not ctype.startswith("audio/"):
                raise yp.ConvertError(f"recording {url} is {ctype}, not audio")
            AUDIO_DIR.mkdir(parents=True, exist_ok=True)
            tmp = path.with_suffix(".part")
            tmp.write_bytes(resp.read())
            tmp.rename(path)
    return f"/assets/{AUDIO_DIR.name}/{name}", path


def duration(path):
    out = subprocess.run(
        ["ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "csv=p=0", str(path)],
        capture_output=True, text=True, check=True,
    )
    return round(float(out.stdout.strip()), 3)


def convert_section(part, n, groups):
    if not part.get("audio_url"):
        raise yp.ConvertError(f"section {n} has no audio_url")
    url, path = localize_audio(part["audio_url"])
    end = part.get("listen_to")
    if not end:
        if path is None:
            raise yp.ConvertError(f"section {n}: no listen_to — pass --audio-dir so its length can be measured")
        end = duration(path)
    transcript, block_map = yp.convert_paragraphs(part, labels=False)
    section = {
        "title": f"Part {n}",
        "audio_url": url,
        "section_start_time": 0,
        # Whole seconds, rounded up: the player shows the summed length as a
        # countdown, and a fraction there renders as 39:17.121000000000095.
        "section_end_time": math.ceil(float(end)),
        "transcript": transcript,
        "question_groups": groups,
    }
    return section, transcript, block_map


yp.convert_section = convert_section


def load_test(test_dir, skill):
    unit, count = ("section", 4) if skill == "listening" else ("passage", 3)
    parts, title = [], None
    for n in range(1, count + 1):
        room = json.loads((test_dir / f"{unit}-{n}.json").read_text(encoding="utf-8"))
        content = room["content"]
        if len(content.get("parts") or []) != 1:
            raise yp.ConvertError(f"{test_dir.name}/{unit}-{n}: expected one part")
        part = prepare(dict(content["parts"][0]), skill)
        part["passage"] = n  # convert() orders parts by this
        parts.append(part)
        title = title or content.get("testSet") or re.sub(r"\s+(PASSAGE|SECTION)\s+\d+$", "", content.get("title") or "")
    return {"data": {"title": f"{title} {skill.capitalize()}", "parts": parts}}


def main():
    global AUDIO_DIR
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("root", type=Path, help="folder holding vol-XX/test-YY/{passage,section}-N.json")
    ap.add_argument("-o", "--out-dir", type=Path, required=True)
    ap.add_argument("--skill", choices=("reading", "listening"), default="reading")
    ap.add_argument("--image-dir", type=Path, help="download diagram / map images here instead of hotlinking")
    ap.add_argument("--audio-dir", type=Path, help="download listening recordings here instead of hotlinking")
    args = ap.parse_args()
    yp.IMAGE_DIR = args.image_dir
    AUDIO_DIR = args.audio_dir
    args.out_dir.mkdir(parents=True, exist_ok=True)

    ok = failed = 0
    for test_dir in sorted(args.root.glob("vol-*/test-*")):
        vol = int(re.search(r"vol-(\d+)", test_dir.parent.name).group(1))
        num = int(re.search(r"test-(\d+)", test_dir.name).group(1))
        name = f"vol-{vol:02d}_test-{num:02d}.json"
        try:
            body = yp.convert(
                load_test(test_dir, args.skill),
                task_type=f"test{num}",
                xp_gain=50,
                source="ieltsmate",
                thumbnail_url="",
                is_current=True,
                series=(SERIES, vol, num),
            )
        except (yp.ConvertError, KeyError, ValueError, FileNotFoundError) as e:
            failed += 1
            print(f"FAILED {name}: {e}", file=sys.stderr)
            continue
        (args.out_dir / name).write_text(json.dumps(body, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
        ok += 1
    print(f"converted {ok}, failed {failed}")


if __name__ == "__main__":
    main()

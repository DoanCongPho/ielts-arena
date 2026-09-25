#!/usr/bin/env python3
"""Fetch YouPass speaking topics and turn them into POST /api/tests bodies.

Two steps, so the crawl (which needs a logged-in session) is separate from
the conversion (which can be re-run and tweaked offline):

    # 1. Crawl — needs a YouPass session, taken from the browser's request
    #    headers: authorization (without "Bearer ") and x-device-id.
    YOUPASS_TOKEN=... YOUPASS_DEVICE_ID=... \\
        python3 scripts/youpass_speaking.py fetch tests_bank/youpass/speaking/raw.json

    # 2. Convert — one CreateTestRequest JSON per test into OUT_DIR.
    python3 scripts/youpass_speaking.py build tests_bank/youpass/speaking/raw.json \\
        tests_bank/youpass/speaking/tests

    # 3. Load them (see import_tests.sh): api import-test FILE

YouPass shape, as far as this script relies on it:

    GET /v1/speaking-topics?speaking_part_type=N&page=P
      data.topics[]
        .id, .title                     Part 1: "Work or Studies";
                                        Part 2: a quarter's set, "[T09-12/26] New topics";
                                        Part 3: the cue card it follows, "Building you enjoy visiting…"
        .tags[].code                    SPEAKING_NEW_TOPICS / SPEAKING_ESSENTIAL_TOPICS …
        .quizzes[]                      one question each
          .id, .sort, .questions[0].title
    GET /v1/quizzes/{id}                (Part 2 only: the cue card's bullets)
      data.questions[0].description     "<p>You should say:</p><ul><li>…</li>…<li>And explain…</li></ul>"

Output tests (content_data as in internal/feature/ielts_test/speaking_content.go):

    part1  one per Part 1 topic, split into sets of at most PART1_SET questions
    part2  one per cue card
    part3  one per Part 3 topic, split into sets of at most PART3_SET questions
    full   one per cue card that has a Part 3 topic: two Part 1 topics
           (rotating through the common ones) + the card + its Part 3

Stdlib only.
"""

import argparse
import html
import json
import os
import re
import sys
import time
import urllib.error
import urllib.request
from html.parser import HTMLParser
from pathlib import Path

API = "https://api.youpass.vn/v1"
PAGE_SIZE = 50

PART1_SET = 6  # questions per Part 1 practice test
PART1_FULL = 4  # questions per Part 1 topic inside a full test
PART3_SET = 6  # questions per Part 3 practice test (validation allows 3-8)
MIN_SET = 3

XP = {"full": 50, "part1": 15, "part2": 20, "part3": 15}

# As in the real exam, a full test's Part 1 opens with an introductory topic
# (work or studies, hometown, home), which YouPass tags as essential, then
# moves to an everyday one; both rotate so tests don't repeat a pair.
ESSENTIAL_TAG = "SPEAKING_ESSENTIAL_TOPICS"


# --- fetch -------------------------------------------------------------------


def get(path: str) -> dict:
    token = os.environ.get("YOUPASS_TOKEN", "").removeprefix("Bearer ").strip()
    if not token:
        sys.exit("set YOUPASS_TOKEN (the authorization header of a logged-in youpass.vn request)")
    req = urllib.request.Request(
        f"{API}/{path}",
        headers={
            "accept": "application/json, text/plain, */*",
            "authorization": f"Bearer {token}",
            "x-device-id": os.environ.get("YOUPASS_DEVICE_ID", ""),
            "origin": "https://youpass.vn",
            "referer": "https://youpass.vn/",
            "user-agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 "
            "(KHTML, like Gecko) Chrome/151.0.0.0 Safari/537.36",
        },
    )
    for attempt in range(4):
        try:
            with urllib.request.urlopen(req, timeout=30) as r:
                return json.load(r)
        except urllib.error.HTTPError as e:
            if e.code == 401:
                sys.exit("YouPass rejected the token (401): it has expired — copy a fresh one from the browser")
            if attempt == 3:
                raise
        except urllib.error.URLError:
            if attempt == 3:
                raise
        time.sleep(2 * (attempt + 1))
    raise RuntimeError("unreachable")


def fetch_topics(part: int) -> list:
    topics, page = [], 1
    while True:
        data = get(f"speaking-topics?page_size={PAGE_SIZE}&page={page}&status=published&speaking_part_type={part}")["data"]
        batch = data.get("topics") or []
        topics += batch
        print(f"  part {part}: {len(topics)}/{data.get('total')} topics", file=sys.stderr)
        if not batch or len(topics) >= (data.get("total") or 0):
            return topics
        page += 1
        time.sleep(0.3)  # be polite


def cmd_fetch(args) -> None:
    raw = {"fetched_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()), "parts": {}}
    for part in (1, 2, 3):
        raw["parts"][str(part)] = fetch_topics(part)

    # Part 2 lists only the cue card's first line; its bullets are in the
    # quiz detail.
    cards = [q for t in raw["parts"]["2"] for q in t.get("quizzes") or []]
    raw["part2_details"] = {}
    for i, q in enumerate(cards, 1):
        detail = get(f"quizzes/{q['id']}")["data"]
        qs = detail.get("questions") or []
        raw["part2_details"][str(q["id"])] = {"title": detail.get("title"), "description": qs[0].get("description") if qs else ""}
        if i % 25 == 0 or i == len(cards):
            print(f"  cue cards: {i}/{len(cards)}", file=sys.stderr)
        time.sleep(0.2)

    out = Path(args.out)
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(json.dumps(raw, ensure_ascii=False, indent=1))
    print(f"wrote {out}", file=sys.stderr)


# --- build -------------------------------------------------------------------


class ListItems(HTMLParser):
    """Collects the text of each <li> (the cue card's bullets)."""

    def __init__(self):
        super().__init__()
        self.items, self._buf, self._in = [], [], 0

    def handle_starttag(self, tag, attrs):
        if tag == "li":
            self._in += 1
            self._buf = []

    def handle_endtag(self, tag):
        if tag == "li" and self._in:
            self._in -= 1
            self.items.append(clean(" ".join(self._buf)))

    def handle_data(self, data):
        if self._in:
            self._buf.append(data)


def clean(s: str) -> str:
    return re.sub(r"\s+", " ", html.unescape(s or "")).strip()


def sentence(s: str) -> str:
    s = clean(s)
    return s if not s or s[-1] in ".?!" else s + "."


def cue_card(title: str, description: str):
    """(topic, bullets, explain) from a YouPass card, or None if it isn't
    a proper IELTS cue card."""
    p = ListItems()
    p.feed(description or "")
    items = [i for i in p.items if i]
    if not items:
        return None
    explain = ""
    if re.match(r"(?i)^and\b", items[-1]):
        explain = items.pop()
    elif len(items) >= 4 and re.match(r"(?i)^explain\b", items[-1]):
        # "Explain why…" written without the "and".
        explain = "and " + items.pop()
    elif len(items) >= 4 and re.match(r"(?i)^(why|how|what|whether)\b", items[-1]):
        # A fourth point that is really the explanation: "why you will…".
        explain = "and explain " + items.pop()
    bullets = [b.rstrip(".") for b in items][:4]
    if len(bullets) < 3 or not explain:
        return None
    explain = explain[0].lower() + explain[1:]
    return sentence(title), bullets, sentence(explain)


def questions(quizzes: list) -> list:
    out = []
    for q in sorted(quizzes or [], key=lambda q: (q.get("sort") or 0, q["id"])):
        text = clean((q.get("questions") or [{}])[0].get("title") or q.get("title"))
        if text and text not in out:
            out.append(text if text[-1] in ".?!" else text + "?")
    return [t[:300] for t in out]


def chunks(items: list, size: int) -> list:
    """Split into sets of at most `size`, folding a short tail into the
    previous set so none is below MIN_SET."""
    sets = [items[i : i + size] for i in range(0, len(items), size)]
    if len(sets) > 1 and len(sets[-1]) < MIN_SET:
        tail = sets.pop()
        sets[-1] += tail
    return [s for s in sets if len(s) >= MIN_SET]


def subject(card_topic: str) -> str:
    """"Describe a building you enjoy visiting (e.g. …)." -> comparable key."""
    t = re.sub(r"\(.*?\)", "", card_topic.lower())
    t = re.sub(r"^(describe|talk about|tell me about)\s+", "", t.strip())
    t = re.sub(r"^(a|an|the|your)\s+", "", t)
    return re.sub(r"[^a-z ]", "", t).strip()


STOPWORDS = set("a an the and or of to in on at for with your you that it is are was were be "
                "have has had do did from by as about which who what when where why how".split())


# Keyword stems too broad to link a card to a theme on their own.
GENERIC = {"frien", "perso", "peopl", "place", "thing", "someo", "somet", "time", "event", "day"}


def keywords(text: str) -> set:
    return {w[:5] for w in re.findall(r"[a-z]+", text.lower()) if w not in STOPWORDS and len(w) > 2}


def match_part3(card_topic: str, themes: list):
    """The Part 3 topic that follows a cue card. Newer YouPass sets name it
    after the card; older ones by a theme ("achievement") whose keywords
    all appear in the card. Prefers the most specific (most keywords)."""
    key = subject(card_topic)
    card_words = keywords(key)
    best = None
    for theme, qs in themes:
        if subject(theme) == key:
            return qs
        words = keywords(theme)
        if not words or not words <= card_words:
            continue
        # A broad one-word theme ("Friends") only fits a card that is about
        # little else: "a friend from your childhood" yes, "a friend who
        # learned something without a teacher" no.
        if words <= GENERIC and len(card_words) > 3:
            continue
        if best is None or len(words) > best[0]:
            best = (len(words), qs)
    return best[1] if best else None


def test(key: str, title: str, content: dict, mode: str, tags: list) -> dict:
    return {
        "key": key,
        "request": {
            "skill": "speaking",
            "task_type": mode,
            "content_data": {"title": title[:100], **content},
            "source": "youpass",
            "is_current": True,
            "xp_gain": XP[mode],
        },
        "tags": tags,
    }


def cmd_build(args) -> None:
    raw = json.loads(Path(args.raw).read_text())
    parts = raw["parts"]
    out = []

    # Part 1
    part1_topics = []
    for t in parts["1"]:
        qs = questions(t.get("quizzes"))
        name = clean(t["title"])
        tags = [g.get("code") for g in t.get("tags") or []]
        if len(qs) >= MIN_SET:
            part1_topics.append((name, qs, ESSENTIAL_TAG in tags))
        sets = chunks(qs, PART1_SET)
        for i, s in enumerate(sets, 1):
            label = f"{name} ({i}/{len(sets)})" if len(sets) > 1 else name
            content = {"part1": {"topics": [{"topic": name, "questions": [{"text": q} for q in s]}]}}
            out.append(test(f"p1:{t['id']}:{i}", f"Part 1 · {label}", content, "part1", tags))

    # Part 3, kept by theme to pair with cue cards below.
    part3_themes = []
    for t in parts["3"]:
        qs = questions(t.get("quizzes"))
        theme = clean(t["title"])
        tags = [g.get("code") for g in t.get("tags") or []]
        if len(qs) >= MIN_SET:
            part3_themes.append((theme, qs))
        for i, s in enumerate(chunks(qs, PART3_SET), 1):
            content = {"part3": {"theme": re.sub(r"\s*\(.*?\)", "", theme), "questions": [{"text": q} for q in s]}}
            out.append(test(f"p3:{t['id']}:{i}", f"Part 3 · {theme}", content, "part3", tags))

    # Part 2, and full tests where the card has a Part 3.
    openers = [p for p in part1_topics if p[2]] or part1_topics
    everyday = [p for p in part1_topics if not p[2]] or part1_topics
    rotation, skipped, full = 0, 0, 0
    for t in parts["2"]:
        tags = [g.get("code") for g in t.get("tags") or []]
        for q in t.get("quizzes") or []:
            detail = raw["part2_details"].get(str(q["id"])) or {}
            card = cue_card(detail.get("title") or q.get("title"), detail.get("description"))
            if card is None:
                skipped += 1
                continue
            topic, bullets, explain = card
            part2 = {"topic": topic, "bullets": bullets, "explain": explain}
            out.append(test(f"p2:{q['id']}", f"Part 2 · {topic}", {"part2": part2}, "part2", tags))

            p3 = match_part3(topic, part3_themes)
            if p3 and len(p3) >= MIN_SET and openers and everyday:
                a, b = openers[rotation % len(openers)], everyday[rotation % len(everyday)]
                rotation += 1
                content = {
                    "part1": {"topics": [
                        {"topic": a[0], "questions": [{"text": x} for x in a[1][:PART1_FULL]]},
                        {"topic": b[0], "questions": [{"text": x} for x in b[1][:PART1_FULL]]},
                    ]},
                    "part2": part2,
                    "part3": {"questions": [{"text": x} for x in p3[:PART3_SET]]},
                }
                out.append(test(f"full:{q['id']}", f"Full test · {topic}", content, "full", tags))
                full += 1

    out_dir = Path(args.out_dir)
    out_dir.mkdir(parents=True, exist_ok=True)
    for old in out_dir.glob("*.json"):
        old.unlink()
    for t in out:
        name = re.sub(r"[^a-z0-9]+", "-", t["key"].lower()).strip("-") + ".json"
        (out_dir / name).write_text(json.dumps(t["request"], ensure_ascii=False, indent=1))
    counts = {m: sum(1 for t in out if t["request"]["task_type"] == m) for m in XP}
    print(f"wrote {len(out)} tests to {out_dir}: {counts}; {skipped} cue cards skipped (not 3-4 bullets + explain)", file=sys.stderr)


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = ap.add_subparsers(dest="cmd", required=True)
    f = sub.add_parser("fetch", help="crawl YouPass speaking topics into a raw JSON file")
    f.add_argument("out")
    b = sub.add_parser("build", help="convert the raw JSON into one test JSON per file")
    b.add_argument("raw")
    b.add_argument("out_dir")
    args = ap.parse_args()
    {"fetch": cmd_fetch, "build": cmd_build}[args.cmd](args)


if __name__ == "__main__":
    main()

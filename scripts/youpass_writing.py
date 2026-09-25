#!/usr/bin/env python3
"""Fetch YouPass writing tests and turn them into POST /api/tests bodies.

    # 1. Crawl — needs a YouPass session (see youpass_speaking.py).
    YOUPASS_TOKEN=... YOUPASS_DEVICE_ID=... \\
        python3 scripts/youpass_writing.py fetch tests_bank/youpass/writing/raw.json

    # 2. Convert — one CreateTestRequest JSON per task into OUT_DIR, with
    #    Task 1 charts downloaded into IMAGE_DIR and served from /assets/.
    python3 scripts/youpass_writing.py build tests_bank/youpass/writing/raw.json \\
        tests_bank/youpass/writing/tests --image-dir internal/assets/diagrams

YouPass shape, as far as this script relies on it:

    GET /v1/quizzes?types=7             the writing test bank
      data.items[].id, .title           "[C14T2] - Export earnings"
    GET /v1/quizzes/{id}
      data.writing_task_type            1 or 2
      data.questions[0]
        .title                          the task prompt as plain text
        .content_writing                the same as HTML, sometimes longer (Task 2
                                        prompts can have several paragraphs)
        .writing_graph_image            Task 1 chart: a cms.youpass.vn asset id

Output (content_data as WritingContent in internal/feature/ielts_test/models.go):
    {"prompt": "...", "image_url": "/assets/diagrams/<id>.png"}

A "[C14T2]" title is Cambridge 14 Test 2 and gets series/volume/test_number,
like the reading and listening imports. Stdlib only.
"""

import argparse
import html
import json
import re
import sys
import time
import urllib.request
from html.parser import HTMLParser
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from youpass_speaking import get  # noqa: E402  (same session handling)

PAGE_SIZE = 50
CMS = "https://cms.youpass.vn/assets"
XP = {"task1": 30, "task2": 40}
MIN_WORDS = {"task1": 150, "task2": 250}


def cmd_fetch(args) -> None:
    items, page = [], 1
    while True:
        data = get(f"quizzes?page_size={PAGE_SIZE}&page={page}&status=published&types=7")["data"]
        batch = data.get("items") or []
        items += batch
        print(f"  list: {len(items)}/{data.get('total')}", file=sys.stderr)
        if not batch or len(items) >= (data.get("total") or 0):
            break
        page += 1
        time.sleep(0.3)

    details = {}
    for i, it in enumerate(items, 1):
        details[str(it["id"])] = get(f"quizzes/{it['id']}")["data"]
        if i % 50 == 0 or i == len(items):
            print(f"  details: {i}/{len(items)}", file=sys.stderr)
        time.sleep(0.2)

    out = Path(args.out)
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(json.dumps({"fetched_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()), "quizzes": details}, ensure_ascii=False))
    print(f"wrote {out}", file=sys.stderr)


class Text(HTMLParser):
    """HTML to plain paragraphs (the attempt page renders the prompt as
    Markdown, so paragraphs are separated by a blank line)."""

    BLOCK = {"p", "div", "br", "li", "h1", "h2", "h3", "h4", "tr"}

    def __init__(self):
        super().__init__()
        self.parts = []

    def handle_starttag(self, tag, attrs):
        if tag in self.BLOCK:
            self.parts.append("\n")

    def handle_endtag(self, tag):
        if tag in self.BLOCK:
            self.parts.append("\n")

    def handle_data(self, data):
        self.parts.append(data)

    def paragraphs(self):
        text = html.unescape("".join(self.parts)).replace("\xa0", " ")
        paras = [re.sub(r"[ \t]+", " ", p).strip() for p in text.split("\n")]
        return [p for p in paras if p]


def prompt_text(q: dict, task: str) -> str:
    p = Text()
    p.feed(q.get("content_writing") or "")
    paras = p.paragraphs() or [html.unescape(q.get("title") or "").strip()]
    text = "\n\n".join(paras)
    # The official wording ends with the length requirement; YouPass often
    # leaves it off.
    if not re.search(r"(?i)write at least \d+ words", text):
        text += f"\n\nWrite at least {MIN_WORDS[task]} words."
    return text


def series_of(title: str):
    m = re.match(r"\s*\[C(\d+)T(\d+)\]", title or "")
    return ("cambridge", int(m.group(1)), int(m.group(2))) if m else None


def download(asset_id: str, image_dir: Path) -> str:
    image_dir.mkdir(parents=True, exist_ok=True)
    existing = list(image_dir.glob(asset_id + ".*"))
    if existing:
        return existing[0].name
    with urllib.request.urlopen(f"{CMS}/{asset_id}", timeout=60) as r:
        ext = {"image/png": ".png", "image/jpeg": ".jpg", "image/webp": ".webp", "image/gif": ".gif"}.get(
            r.headers.get_content_type(), ".png")
        name = asset_id + ext
        (image_dir / name).write_bytes(r.read())
    return name


def cmd_build(args) -> None:
    raw = json.loads(Path(args.raw).read_text())
    out_dir = Path(args.out_dir)
    out_dir.mkdir(parents=True, exist_ok=True)
    for old in out_dir.glob("*.json"):
        old.unlink()
    image_dir = Path(args.image_dir) if args.image_dir else None

    written, skipped, seen = {"task1": 0, "task2": 0}, [], set()
    for qid, d in raw["quizzes"].items():
        qs = d.get("questions") or []
        wtt = d.get("writing_task_type")
        if not qs or wtt not in (1, 2):
            skipped.append((qid, "no question or task type"))
            continue
        task = f"task{wtt}"
        q = qs[0]
        prompt = prompt_text(q, task)
        if len(prompt) < 40 or prompt in seen:
            skipped.append((qid, "empty or duplicate prompt"))
            continue
        seen.add(prompt)

        image_url = ""
        asset = q.get("writing_graph_image")
        if not asset:
            m = re.search(r"cms\.youpass\.vn/assets/([0-9a-f-]{36})", q.get("content_writing") or "")
            asset = m.group(1) if m else ""
        if task == "task1":
            if not asset:
                skipped.append((qid, "Task 1 without a chart"))
                continue
            if image_dir:
                image_url = f"/assets/{image_dir.name}/{download(asset, image_dir)}"
            else:
                image_url = f"{CMS}/{asset}"

        req = {
            "skill": "writing",
            "task_type": task,
            "content_data": {"prompt": prompt, **({"image_url": image_url} if image_url else {})},
            "source": "youpass",
            "is_current": True,
            "xp_gain": XP[task],
        }
        if s := series_of(d.get("title")):
            req["series"], req["volume"], req["test_number"] = s
        (out_dir / f"{task}-{qid}.json").write_text(json.dumps(req, ensure_ascii=False, indent=1))
        written[task] += 1

    print(f"wrote {sum(written.values())} tests to {out_dir}: {written}; skipped {len(skipped)}", file=sys.stderr)
    reasons = {}
    for _, why in skipped:
        reasons[why] = reasons.get(why, 0) + 1
    if reasons:
        print(f"  skipped: {reasons}", file=sys.stderr)


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = ap.add_subparsers(dest="cmd", required=True)
    f = sub.add_parser("fetch")
    f.add_argument("out")
    b = sub.add_parser("build")
    b.add_argument("raw")
    b.add_argument("out_dir")
    b.add_argument("--image-dir", help="download Task 1 charts here and serve them from /assets/<dir name>/")
    args = ap.parse_args()
    {"fetch": cmd_fetch, "build": cmd_build}[args.cmd](args)


if __name__ == "__main__":
    main()

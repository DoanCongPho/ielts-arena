"""Download every IELTS Mate room of one test set (e.g. "VOL 10"): its
passage (reading) or section (listening) rooms, as returned by
GET /api/study-room/room/<roomId>. ieltsmate_to_test.py converts them.

The listing needs a signed-in IELTS Mate account's bearer token (copy it
from a request in the browser's network tab); keep it in a file outside the
repo. "FULL TEST" rooms are skipped: they return passage / section 1 only.

usage: python3 -I scripts/ieltsmate_fetch.py TOKEN_FILE OUT_DIR [testSet] [reading|listening]
  e.g. python3 -I scripts/ieltsmate_fetch.py ~/ieltsmate.token tests_bank/ieltsmate/vol-10 "VOL 10"
"""
import json, re, sys, time, urllib.parse, urllib.request
from pathlib import Path

API = "https://api.ieltsmate.vn/api"
token = Path(sys.argv[1]).read_text().strip()
out = Path(sys.argv[2])
test_set = sys.argv[3] if len(sys.argv) > 3 else "VOL 10"
SKILL = sys.argv[4] if len(sys.argv) > 4 else "reading"
UNIT = "section" if SKILL == "listening" else "passage"
HEADERS = {
    "accept": "application/json",
    "authorization": f"Bearer {token}",
    "origin": "https://www.ieltsmate.vn",
    "referer": "https://www.ieltsmate.vn/",
    "user-agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 "
                  "(KHTML, like Gecko) Chrome/157.0.0.0 Safari/537.36",
}


def get(path):
    req = urllib.request.Request(API + path, headers=HEADERS)
    with urllib.request.urlopen(req, timeout=60) as r:
        return json.load(r)


rooms, page = [], 1
while True:
    q = urllib.parse.urlencode({
        "page": page, "limit": 24, "type": SKILL, "contentType": "academic",
        "status": "waiting", "testSet": test_set, "progressStatus": "all",
    })
    d = get(f"/study-room?{q}")
    rooms += d["data"]
    if page >= d["totalPages"]:
        break
    page += 1
    time.sleep(0.5)

# The testSet filter is a prefix match ("VOL 1" would also catch "VOL 10"), so keep exact sets only.
pat = re.compile(rf"^{re.escape(test_set)} TEST \d+$")
rooms = [r for r in rooms if pat.match(r.get("testSet") or "")]
out.mkdir(parents=True, exist_ok=True)
(out / "rooms.json").write_text(json.dumps(rooms, ensure_ascii=False, indent=2))

index = []
for r in sorted(rooms, key=lambda r: r["title"]):
    m = re.search(r"TEST (\d+) (?:PASSAGE|SECTION) (\d+)", r["title"])
    if not m:
        # "READING (FULL TEST)" rooms return the same content as passage 1, so skip them.
        print(f"skip {r['title']}")
        continue
    test_no, passage = int(m.group(1)), m.group(2)
    name = f"{UNIT}-{passage}.json"
    dest = out / f"test-{test_no:02d}" / name
    dest.parent.mkdir(exist_ok=True)
    data = get(f"/study-room/room/{r['id']}")
    dest.write_text(json.dumps(data, ensure_ascii=False, indent=2))
    parts = data.get("content", {}).get("parts", [])
    nq = sum(len(qs.get("questions", [])) for p in parts for qs in p.get("question_sets", []))
    index.append({"title": r["title"], "roomId": r["id"], "questionId": r["questionId"],
                  "file": str(dest.relative_to(out)), "parts": len(parts), "questions": nq})
    print(f"{r['title']:<40} parts={len(parts)} questions={nq} -> {dest.relative_to(out)}")
    time.sleep(0.5)

(out / "index.json").write_text(json.dumps(index, ensure_ascii=False, indent=2))
print(f"\n{len(index)} rooms saved to {out}")

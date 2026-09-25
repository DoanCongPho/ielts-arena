#!/usr/bin/env bash
# Imports every test JSON in a directory into the running stack, e.g. the
# output of youpass_speaking.py / youpass_writing.py:
#
#   scripts/import_tests.sh tests_bank/youpass/speaking/tests
#
# Each file's created test id is recorded in DIR/.imported, so running it
# again replaces those tests (import-test --replace) instead of creating
# duplicates. Set API_CMD to import some other way than through the
# compose stack's api container, e.g. API_CMD="go run ./cmd/api".
set -euo pipefail
dir="${1:?usage: scripts/import_tests.sh DIR}"
api="${API_CMD:-docker compose exec -T api ./api}"
state="$dir/.imported"
touch "$state"

created=0 replaced=0 failed=0
for f in "$dir"/*.json; do
  name="$(basename "$f")"
  id="$(awk -v n="$name" '$1 == n { print $2 }' "$state")"
  if [ -n "$id" ]; then
    if out="$($api import-test --replace "$id" - < "$f" 2>&1)"; then
      replaced=$((replaced + 1))
      continue
    fi
    # The test was deleted since; create it again below.
    grep -v "^$name " "$state" > "$state.tmp" && mv "$state.tmp" "$state"
  fi
  if out="$($api import-test - < "$f" 2>&1)"; then
    new_id="$(sed -n 's/^created test \([0-9]*\):.*/\1/p' <<< "$out")"
    echo "$name $new_id" >> "$state"
    created=$((created + 1))
  else
    failed=$((failed + 1))
    echo "FAILED $name: $out" >&2
  fi
done
echo "created $created, replaced $replaced, failed $failed"

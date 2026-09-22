#!/usr/bin/env bash
# Mirrors test media into an S3-compatible bucket (Backblaze B2 or Cloudflare
# R2), laid out like internal/assets so ASSETS_BASE_URL can stand in for
# /assets:
#   media/audio/*            -> <bucket>/audio/*
#   internal/assets/diagrams -> <bucket>/diagrams/*
#
# Needs rclone (brew install rclone) and, in .env.prod (or the environment):
#   S3_PROVIDER           Other (Backblaze B2) | Cloudflare (R2)
#   S3_ENDPOINT           B2: https://s3.<region>.backblazeb2.com
#                         R2: https://<account id>.r2.cloudflarestorage.com
#   S3_ACCESS_KEY_ID      B2: application keyID
#   S3_SECRET_ACCESS_KEY  B2: application key
#   S3_BUCKET
#
# Re-running only uploads what changed.
set -euo pipefail
cd "$(dirname "$0")/.."

if [[ -f .env.prod ]]; then
  set -a; source .env.prod; set +a
fi
: "${S3_ENDPOINT:?}" "${S3_ACCESS_KEY_ID:?}" "${S3_SECRET_ACCESS_KEY:?}" "${S3_BUCKET:?}"

# Remote "dest" configured from env, so no rclone.conf is written.
export RCLONE_CONFIG_DEST_TYPE=s3
export RCLONE_CONFIG_DEST_PROVIDER="${S3_PROVIDER:-Other}"
export RCLONE_CONFIG_DEST_ACCESS_KEY_ID="$S3_ACCESS_KEY_ID"
export RCLONE_CONFIG_DEST_SECRET_ACCESS_KEY="$S3_SECRET_ACCESS_KEY"
export RCLONE_CONFIG_DEST_ENDPOINT="$S3_ENDPOINT"
export RCLONE_CONFIG_DEST_NO_CHECK_BUCKET=true

rclone copy media/audio "dest:${S3_BUCKET}/audio" --progress --transfers 8
rclone copy internal/assets/diagrams "dest:${S3_BUCKET}/diagrams" --progress

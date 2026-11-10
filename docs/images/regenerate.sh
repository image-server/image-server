#!/bin/sh
# Run against a local server: sh docs/images/regenerate.sh [http://localhost:7000]
set -eu

asset_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
server_url=${1:-http://localhost:7000}
response=$(curl --fail --silent --show-error \
  -H 'Content-Type: image/jpeg' \
  --data-binary "@$asset_dir/sample.jpg" \
  "${server_url%/}/demo")
printf '%s\n' "$response"
hash=$(printf '%s' "$response" | python3 -c 'import json, sys; print(json.load(sys.stdin)["hash"])')
image_path=$(printf '%s' "$hash" | sed 's|^\(...\)\(...\)\(...\)|\1/\2/\3/|')
base="${server_url%/}/demo/$image_path"

for filename in w480.jpg x240.jpg 480x240.jpg w480-q80.webp c0.250_0.250_0.750_0.750-w480.jpg; do
  curl --fail --silent --show-error "$base/$filename" -o "$asset_dir/$filename"
  printf 'Saved %s\n' "$filename"
done

#!/bin/bash
EC2_HOST=18.206.160.161

set -euo pipefail
cd "$(dirname "$0")"

IMAGE=lerthsk/lab1
SSH_KEY="$HOME/.ssh/tda596-aws.pem"

changes=$(git status --porcelain --untracked-files=all)
if [[ -n "$changes" ]]; then
    printf 'Warning: commit all changes before deploying.\n%s\n' "$changes" >&2
    exit 1
fi
commit=$(git rev-parse HEAD)
tag=$(git rev-parse --short "$commit")

metadata=$(mktemp)
trap 'rm -f "$metadata"' EXIT

# Build the committed snapshot, excluding ignored files and any later edits.
git -C .. archive "$commit:Lab1" | docker buildx build --platform linux/arm64 \
    -t "$IMAGE:$tag" --metadata-file "$metadata" --push -

# Deploy exactly what this build pushed, even if the tag changes afterward.
digest=$(jq -er '.["containerimage.digest"]' "$metadata")
image_ref="$IMAGE@$digest"

ssh -i "$SSH_KEY" -o BatchMode=yes "ec2-user@$EC2_HOST" \
    -o StrictHostKeyChecking=accept-new                 \
    bash -s -- "$image_ref" <<'REMOTE'
set -euo pipefail
sudo -n docker pull "$1"
sudo -n docker rm -f lab1 || true
sudo -n docker run -d --name lab1 --restart unless-stopped \
    -p 80:8080 -e LOG_LEVEL=debug "$1"
test "$(sudo -n docker inspect --format '{{.Image}}' lab1)" = \
    "$(sudo -n docker image inspect --format '{{.Id}}' "$1")"
test "$(sudo -n docker inspect --format '{{.State.Running}}' lab1)" = true
REMOTE

url="http://$EC2_HOST/index.html"
curl -fsS --retry 5 --retry-connrefused --retry-delay 1 \
    --connect-timeout 3 --max-time 5 "$url" >/dev/null
printf 'Deployed %s:%s (%s)\nTest URL: %s\n' "$IMAGE" "$tag" "$digest" "$url"

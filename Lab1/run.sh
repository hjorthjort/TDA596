#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")"

IMAGE=lerthsk/lab1
EC2_HOST=54.144.165.163
SSH_KEY="$HOME/.ssh/tda596-aws.pem"

metadata=$(mktemp)
trap 'rm -f "$metadata"' EXIT

docker buildx build --platform linux/arm64 \
    -t "$IMAGE:latest" --metadata-file "$metadata" --push .

# Deploy exactly what this build pushed, even if latest changes afterward.
digest=$(jq -er '.["containerimage.digest"]' "$metadata")
image_ref="$IMAGE@$digest"

ssh -i "$SSH_KEY" -o BatchMode=yes "ec2-user@$EC2_HOST" \
    bash -s -- "$image_ref" <<'REMOTE'
set -euo pipefail
sudo -n docker pull "$1"
sudo -n docker rm -f lab1 || true
sudo -n docker run -d --name lab1 --restart unless-stopped \
    -p 80:8080 -e LOG_LEVEL=debug "$1"
REMOTE

url="http://$EC2_HOST/index.html"
curl -fsS --retry 5 --retry-connrefused --retry-delay 1 \
    --connect-timeout 3 --max-time 5 "$url" >/dev/null
printf 'Deployed %s\nTest URL: %s\n' "$image_ref" "$url"

#!/bin/bash
EC2_HOST=18.206.160.161

set -euo pipefail
cd "$(dirname "$0")"

if [[ -n "$(git status --porcelain --untracked-files=all)" ]]; then
    echo 'Warning: commit all changes before deploying.' >&2
    exit 1
fi
commit=$(git rev-parse --short HEAD)
IMAGE="lerthsk/lab1:$commit"

# Build and run all tests. Always rerun the Dockerfile's test stage.
git -C .. archive "$commit:Lab1" | docker buildx build --platform linux/arm64 \
    --no-cache-filter test --load -t "$IMAGE" -

# Start locally and check HTTP. Print logs and remove it when the script exits.
container=$(docker run -d -p 127.0.0.1::8080 -e LOG_LEVEL=debug "$IMAGE")
trap 'docker logs "$container"; docker rm -f "$container" >/dev/null' EXIT
address=$(docker port "$container" 8080/tcp)
curl -fsS --retry 5 --retry-connrefused --retry-delay 1 \
    --max-time 5 "http://$address/index.html" >/dev/null
test "$(docker inspect -f '{{.State.Running}}' "$container")" = true

docker push "$IMAGE"
image_ref=$(docker image inspect -f '{{index .RepoDigests 0}}' "$IMAGE")

ssh -i "$HOME/.ssh/tda596-aws.pem" -o BatchMode=yes -o StrictHostKeyChecking=accept-new \
    "ec2-user@$EC2_HOST" bash -s -- "$image_ref" <<'REMOTE'
set -euo pipefail
sudo -n docker pull "$1"
sudo -n docker rm -f lab1 || true
sudo -n docker run -d --name lab1 --restart unless-stopped \
    -p 80:8080 -e LOG_LEVEL=debug "$1"
test "$(sudo -n docker inspect --format '{{.State.Running}}' lab1)" = true
REMOTE

url="http://$EC2_HOST/index.html"
curl -fsS --retry 5 --retry-connrefused --retry-delay 1 \
    --max-time 5 "$url" >/dev/null
printf 'Deployed %s\nTest URL: %s\n' "$IMAGE" "$url"

#!/bin/bash
set -exuo pipefail

if [ $# -eq 0 ] || [ ! -d "${1:-}" ]; then
    echo "Error: \"${1:-}\" is not a directory." >&2
    exit 1
fi

commit=$(git rev-parse --short HEAD)
git archive --format=zip --output="submission_$1_$commit.zip" HEAD "$1"

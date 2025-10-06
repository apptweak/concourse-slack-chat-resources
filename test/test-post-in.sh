#! /bin/bash

set -euo pipefail

run_case() {
  local req=$1
  echo "==> Testing with $req"
  cat "$req" | docker run --rm -i \
    -v "$(pwd)/post/in:/tmp/resource" ghcr.io/apptweak/slack-post-resource /opt/resource/in /tmp/resource > /tmp/out.json
  echo "out: $(cat /tmp/out.json)"
  echo -n "timestamp file: "; cat post/in/timestamp; echo
}

run_case post/in/test-request-version-string.json
run_case post/in/test-request-version-object.json

echo "All cases passed"

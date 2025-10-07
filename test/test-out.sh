#! /bin/bash

set -e # Exit on error
set -u # Exit on unset variable
set -o pipefail # Exit on pipe error
set -x # DEBUG

type=$1
request=$2

# Help Usage
#  ./test/test-out.sh post post/out/test-request-upload.json [image|local]
#  ./test/test-out.sh post post/out/test-request-reactions.json [image|local]
#  ./test/test-out.sh post post/out/test-request-reactions-thread.json [image|local]
#
# Env (optional):
#   SLACK_TOKEN         -> overrides .source.token in request JSON
#   SLACK_CHANNEL_ID    -> overrides .source.channel_id in request JSON
#   THREAD_TS           -> sets/overrides .params.message.thread_ts (for thread tests)
#   RUN_LOCAL=1         -> force local execution (builds ./bin/out and runs it)

# Auto-load environment from .env if present (expects lines like KEY=value)
if [[ -f .env ]]; then
    set -a
    # shellcheck disable=SC1091
    source .env
    set +a
fi

type=${1:-}
request=${2:-}
image_arg=${3:-}
RUN_LOCAL=${RUN_LOCAL:-}

if [[ -z ${type} || -z ${request} ]]; then
    echo "Required arguments: <resource type> <request file> [image]"
    exit 1
fi

# Pick image: prefer explicit arg, then local dev image if present, otherwise GHCR latest
if [[ "${image_arg}" == "local" ]]; then
    RUN_LOCAL=1
fi

if [[ -z "${RUN_LOCAL}" ]]; then
    if [[ -n ${image_arg} && "${image_arg}" != "local" ]]; then
        IMAGE="${image_arg}"
    else
        LOCAL_IMAGE="local/slack-${type}-resource:dev"
        if docker image inspect "${LOCAL_IMAGE}" >/dev/null 2>&1; then
            IMAGE="${LOCAL_IMAGE}"
        else
            IMAGE="ghcr.io/apptweak/slack-${type}-resource:latest"
        fi
    fi
fi

TMP_REQ=""
cleanup() { [[ -n "${TMP_REQ}" && -f "${TMP_REQ}" ]] && rm -f "${TMP_REQ}"; }
trap cleanup EXIT

# If env overrides provided, rewrite request via jq
if [[ -n "${SLACK_TOKEN:-}" || -n "${SLACK_CHANNEL_ID:-}" || -n "${THREAD_TS:-}" ]]; then
    if ! command -v jq >/dev/null 2>&1; then
        echo "jq is required to inject env overrides into the request JSON"
        exit 1
    fi
    FILTER='.'
    if [[ -n "${SLACK_TOKEN:-}" ]]; then
        FILTER+=" | .source.token=env.SLACK_TOKEN"
    fi
    if [[ -n "${SLACK_CHANNEL_ID:-}" ]]; then
        FILTER+=" | .source.channel_id=env.SLACK_CHANNEL_ID"
    fi
    if [[ -n "${THREAD_TS:-}" ]]; then
        FILTER+=" | .params.message.thread_ts=env.THREAD_TS"
    fi
    TMP_REQ=$(mktemp)
    jq "${FILTER}" "${request}" > "${TMP_REQ}"
    REQUEST_FILE="${TMP_REQ}"
else
    REQUEST_FILE="${request}"
fi

if [[ -n "${RUN_LOCAL}" ]]; then
    # Local execution: build and run the Go binary directly, using the repo's source dir as the resource dir
    BIN_DIR="$(pwd)/bin"
    mkdir -p "${BIN_DIR}"
    echo "Building local binary ./bin/out ..."
    go build -o "${BIN_DIR}/out" ./post/out
    SRC_DIR="$(pwd)/${type}/out"
    echo "Running locally against source dir: ${SRC_DIR}"
    cat "${REQUEST_FILE}" | "${BIN_DIR}/out" "${SRC_DIR}"
else
    cat "${REQUEST_FILE}" | docker run --rm -i \
        -e BUILD_NAME=mybuild \
        -e BUILD_JOB_NAME=myjob \
        -e BUILD_PIPELINE_NAME=mypipe \
        -e BUILD_TEAM_NAME=myteam \
        -e ATC_EXTERNAL_URL="https://example.com" \
        --platform linux/amd64 \
        -v "$(pwd)/${type}/out:/tmp/resource" "${IMAGE}" /opt/resource/out /tmp/resource
fi

#! /bin/bash

# Help Usage
#  ` ./test/test-out.sh post post/out/test-request-upload.json`

type=$1
request=$2

if [[ -z $type || -z $request ]]; then
    echo "Required arguments: <resource type> <request file>"
    exit 1
fi

cat "$request" | docker run --rm -i \
-e BUILD_NAME=mybuild \
-e BUILD_JOB_NAME=myjob \
-e BUILD_PIPELINE_NAME=mypipe \
-e BUILD_TEAM_NAME=myteam \
-e ATC_EXTERNAL_URL="https://example.com" \
--platform linux/amd64 \
-v "$(pwd)/$type/out:/tmp/resource" apptweakci/slack-$type-resource:upload-file /opt/resource/out /tmp/resource

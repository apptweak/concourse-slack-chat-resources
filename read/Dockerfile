FROM golang:1.13.15-alpine as build-env
  RUN apk add --no-cache git && \
      go get github.com/slack-go/slack
  COPY ./utils /go/src/github.com/apptweak/slack-chat-resource/utils
  RUN go build -o /assets/utils github.com/apptweak/slack-chat-resource/utils

FROM build-env as build-read
  COPY ./read/check /go/src/github.com/apptweak/slack-chat-resource/read/check/
  COPY ./read/in /go/src/github.com/apptweak/slack-chat-resource/read/in/
  COPY ./read/out /go/src/github.com/apptweak/slack-chat-resource/read/out/
  RUN go build -o /assets/check github.com/apptweak/slack-chat-resource/read/check &&\
      go build -o /assets/in github.com/apptweak/slack-chat-resource/read/in &&\
      go build -o /assets/out github.com/apptweak/slack-chat-resource/read/out

FROM alpine
  ARG VERSION
  ARG VCS_REF
  ARG BUILD_DATE
  LABEL org.opencontainers.image.source=https://github.com/apptweak/concourse-slack-chat-resources
  LABEL org.opencontainers.image.url=https://github.com/apptweak/concourse-slack-chat-resources
  LABEL org.opencontainers.image.documentation=https://github.com/apptweak/concourse-slack-chat-resources#readme
  LABEL org.opencontainers.image.title=slack-read-resource
  LABEL org.opencontainers.image.description="Concourse resource to read Slack messages"
  LABEL org.opencontainers.image.vendor=AppTweak
  LABEL org.opencontainers.image.version=$VERSION
  LABEL org.opencontainers.image.revision=$VCS_REF
  LABEL org.opencontainers.image.created=$BUILD_DATE
  RUN apk add --no-cache ca-certificates
  COPY --from=build-read /assets /opt/resource

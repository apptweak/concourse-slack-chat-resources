FROM golang:1.13.15-alpine as build-env
  RUN apk add --no-cache git && \
    go get github.com/slack-go/slack
  COPY ./utils /go/src/github.com/apptweak/slack-chat-resource/utils
  RUN go build -o /assets/utils github.com/apptweak/slack-chat-resource/utils

FROM build-env as build-post
  COPY ./post/check /go/src/github.com/apptweak/slack-chat-resource/post/check/
  COPY ./post/in /go/src/github.com/apptweak/slack-chat-resource/post/in/
  COPY ./post/out /go/src/github.com/apptweak/slack-chat-resource/post/out/
  RUN go build -o /assets/check github.com/apptweak/slack-chat-resource/post/check && \
      go build -o /assets/in github.com/apptweak/slack-chat-resource/post/in &&\
      go build -o /assets/out github.com/apptweak/slack-chat-resource/post/out

FROM alpine
  RUN apk add --no-cache ca-certificates
  COPY --from=build-post /assets /opt/resource

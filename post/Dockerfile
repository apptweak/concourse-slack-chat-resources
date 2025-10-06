FROM golang:1.22-alpine AS build-post
  WORKDIR /src
  COPY go.mod go.sum ./
  RUN go mod download
  COPY . .
  RUN CGO_ENABLED=0 go build -o /assets/check ./post/check && \
      CGO_ENABLED=0 go build -o /assets/in ./post/in && \
      CGO_ENABLED=0 go build -o /assets/out ./post/out

FROM alpine:3.22
  ARG VERSION
  ARG VCS_REF
  ARG BUILD_DATE
  LABEL org.opencontainers.image.source=https://github.com/apptweak/concourse-slack-chat-resources
  LABEL org.opencontainers.image.url=https://github.com/apptweak/concourse-slack-chat-resources
  LABEL org.opencontainers.image.documentation=https://github.com/apptweak/concourse-slack-chat-resources#readme
  LABEL org.opencontainers.image.title=concourse-slack-post-resource
  LABEL org.opencontainers.image.description="Concourse resource to post/update Slack messages and upload files"
  LABEL org.opencontainers.image.vendor=AppTweak
  LABEL org.opencontainers.image.version=$VERSION
  LABEL org.opencontainers.image.revision=$VCS_REF
  LABEL org.opencontainers.image.created=$BUILD_DATE
  RUN apk add --no-cache ca-certificates
  COPY --from=build-post /assets /opt/resource

FROM golang:1.22-alpine AS build-env
  WORKDIR /src
  COPY go.mod go.sum ./
  RUN go mod download
  COPY . .
  RUN CGO_ENABLED=0 go build -o /assets/check ./read/check && \
      CGO_ENABLED=0 go build -o /assets/in ./read/in && \
      CGO_ENABLED=0 go build -o /assets/out ./read/out

  # build artifacts are in /assets in this stage

FROM alpine:3.22
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
  COPY --from=build-env /assets /opt/resource

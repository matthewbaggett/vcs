# The vcs image. The build is static. The final image holds git, because vcs calls git.
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build
ARG VERSION=dev
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /vcs ./cmd/vcs

FROM alpine:3.22
RUN apk add --no-cache git ca-certificates openssh-client
COPY --from=build /vcs /usr/local/bin/vcs
ENTRYPOINT ["vcs"]

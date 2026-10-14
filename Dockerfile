# Shared Go build environment for both binaries. Keeping module downloads here
# lets the server and CLI targets reuse the same dependency cache.
FROM golang:1.27.2-alpine3.24 AS build-base
RUN apk --no-cache add git gcc bash

# copy the file instead of add we use copy so caching layer could do it better
WORKDIR /src

COPY go.mod /src
COPY go.sum /src
RUN go mod download

# Install swag for swagger docs generation (pinned to a specific version for reproducibility).
# CGO_ENABLED=0: swag is pure Go — no C headers needed, and Alpine lacks musl-dev by default.
RUN CGO_ENABLED=0 go install github.com/swaggo/swag/cmd/swag@v1.16.6

COPY . /src

RUN cat /dev/null > "/src/assets/build_commit_hash.txt"
RUN git rev-list -1 HEAD >> "/src/assets/build_commit_hash.txt"

RUN cat /dev/null > "/src/assets/build_time.txt"
RUN BUILD_TIME=$(date +"%s") && echo "$BUILD_TIME" >> "/src/assets/build_time.txt"

# Shared minimal runtime image. The final server and cli targets add their own
# binary and entrypoint on top of this stage.
FROM alpine:3.24 AS runtime
LABEL maintainer="Yusuf Syaifudin <yusuf.syaifudin@gmail.com>"
LABEL org.opencontainers.image.authors="Yusuf Syaifudin <yusuf.syaifudin@gmail.com>"
RUN apk update && apk --no-cache add curl bash which wget openssh
WORKDIR /app

# Server target: generate Swagger files, then compile cmd/server/main.go.
FROM build-base AS build-server
RUN swag init \
      --parseDependency \
      --parseInternal \
      -g transport/restapi/doc.go \
      -o docs/swagger
RUN CGO_ENABLED=0 go build -o /src/server.bin /src/cmd/server/main.go

# Select with `docker build --target server`; this image runs the HTTP service.
FROM runtime AS server
COPY --from=build-server /src/server.bin /app/server.bin
ENTRYPOINT ["/app/server.bin"]

# CLI target: compile cmd/cli/main.go without running Swagger generation.
FROM build-base AS build-cli
RUN CGO_ENABLED=0 go build -o /src/cli.bin /src/cmd/cli/main.go

# Select with `docker build --target cli`; this image runs the standalone CLI.
FROM runtime AS cli
COPY --from=build-cli /src/cli.bin /app/cli.bin
ENTRYPOINT ["/app/cli.bin"]

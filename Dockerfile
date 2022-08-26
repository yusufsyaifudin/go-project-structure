# build stage
FROM golang:alpine AS build-env
RUN apk --no-cache add git gcc bash

# copy the file instead of add we use copy so caching layer could do it better
WORKDIR /src

COPY go.mod /src
COPY go.sum /src
RUN go mod download

COPY . /src

RUN cat /dev/null > "/src/assets/build_commit_hash.txt"
RUN git rev-list -1 HEAD >> "/src/assets/build_commit_hash.txt"

RUN cat /dev/null > "/src/assets/build_time.txt"
RUN BUILD_TIME=$(date +"%s") && echo "$BUILD_TIME" >> "/src/assets/build_time.txt"

# server
RUN env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server.bin /src/cmd/server/main.go

# final stage build using pure alpine image
FROM alpine:3.16
LABEL maintainer="Yusuf"

RUN apk update
RUN apk --no-cache add curl bash which wget openssh

# copy golang binary
WORKDIR /app
COPY --from=build-env /src/server.bin /app/server.bin


CMD [ "/app/server.bin" ]

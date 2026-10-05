# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM golang:1.25.13-alpine AS build
WORKDIR /src/server
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/cmd ./cmd
COPY server/internal ./internal
COPY tools /src/tools
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w -X github.com/Kook-Dohyun/To-Do-Connect/internal/app.buildVersion=${VERSION}" -o /out/todo-connect ./cmd/todo-connect
RUN go run ../tools/license-notices/main.go /out/todo-connect /out/THIRD_PARTY_NOTICES.txt
COPY LICENSE /out/LICENSE

# Developer verification only. No compiler or test fixture enters the runtime image.
FROM build AS test
RUN apk add --no-cache build-base
RUN CGO_ENABLED=1 go test -race ./... -count=1 && go vet ./... && go -C ../tools test ./... && go -C ../tools vet ./...

FROM alpine:3.23 AS runtime
RUN apk add --no-cache ca-certificates \
    && addgroup -S -g 10001 todo-connect \
    && adduser -S -D -H -u 10001 -G todo-connect todo-connect \
    && mkdir /data /keys \
    && chown 10001:10001 /data /keys \
    && chmod 700 /data /keys
COPY --from=build /out/todo-connect /usr/local/bin/todo-connect
COPY --from=build /out/LICENSE /out/THIRD_PARTY_NOTICES.txt /usr/share/licenses/todo-connect/
ENV TODO_CONNECT_DATA_DIR=/data TODO_CONNECT_KEY_FILE=/keys/todo-connect.key
USER 10001:10001
WORKDIR /data
ENTRYPOINT ["/usr/local/bin/todo-connect"]
CMD ["serve"]

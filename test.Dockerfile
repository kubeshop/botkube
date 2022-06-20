# syntax = docker/dockerfile:1-experimental

FROM golang:1.18-alpine as builder

ARG TEST_NAME
ARG SOURCE_PATH="./tests/$TEST_NAME"

WORKDIR /botkube

# Use experimental frontend syntax to cache dependencies.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Use experimental frontend syntax to cache go build.
# Replace `COPY . .` with `--mount=target=.` to speed up as we do not need them to persist in the final image.
# https://github.com/moby/buildkit/blob/master/frontend/dockerfile/docs/syntax.md
RUN --mount=target=. \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOARCH=amd64 go test -c -tags=integration -o /bin/$TEST_NAME $SOURCE_PATH

FROM scratch as generic

ARG TEST_NAME

# Copy common CA certificates from Builder image (installed by default with ca-certificates package)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder /bin/$TEST_NAME /test

LABEL name="botkube" \
    test=$TEST_NAME

CMD ["/test", "-test.v"]

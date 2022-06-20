# syntax = docker/dockerfile:1-experimental

FROM alpine:3.15 as builder

FROM scratch as generic

ARG TEST_NAME

# Copy common CA certificates from Builder image (installed by default with ca-certificates package)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY $TEST_NAME /test

LABEL name="botkube" \
    test=$TEST_NAME

CMD ["/test", "-test.v"]

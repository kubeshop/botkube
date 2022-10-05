FROM alpine:3.15 as builder

FROM scratch as generic

ARG TEST_NAME

# Copy common CA certificates from Builder image (installed by default with ca-certificates package)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY $TEST_NAME /test

LABEL name="botkube" \
      test=$TEST_NAME \
      org.opencontainers.image.source="git@github.com:kubeshop/botkube.git" \
      org.opencontainers.image.title="Botkube E2 tests" \
      org.opencontainers.image.description="Botkube E2E tests which are run against Botkube installed on Kubernetes cluster and Slack API." \
      org.opencontainers.image.documentation="https://botkube.io" \
      org.opencontainers.image.licenses="MIT"

CMD ["/test", "-test.v"]

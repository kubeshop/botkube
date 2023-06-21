FROM alpine:3.15
ARG botkube_version="dev"
LABEL org.opencontainers.image.source="git@github.com:kubeshop/botkube.git" \
      org.opencontainers.image.title="Botkube" \
      org.opencontainers.image.version="${botkube_version}" \
      org.opencontainers.image.description="Botkube is a messaging bot for monitoring and debugging Kubernetes clusters" \
      org.opencontainers.image.documentation="https://docs.botkube.io" \
      org.opencontainers.image.licenses="MIT"

COPY botkube-agent /usr/local/bin/botkube-agent

# Create Non Privileged user
RUN addgroup --gid 1001 botkube && \
    adduser -S --uid 1001 --ingroup botkube botkube

# Run as Non Privileged user
USER botkube

ENTRYPOINT /usr/local/bin/botkube-agent

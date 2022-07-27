FROM alpine:3.15
ARG botkube_version="dev"
LABEL org.opencontainers.image.source="git@github.com:kubeshop/botkube.git" \
      org.opencontainers.image.title="BotKube" \
      org.opencontainers.image.version="${botkube_version}" \
      org.opencontainers.image.description="BotKube is a messaging bot for monitoring and debugging Kubernetes clusters" \
      org.opencontainers.image.documentation="https://botkube.io"

COPY botkube /usr/local/bin/botkube
# Download the latest kubectl in the appropriate architecture. Currently handles aarch64 (arm64) and x86_64 (amd64).
RUN MACH=$(uname -m); if [[ ${MACH} == "aarch64" ]]; then ARCH=arm64; \
    elif [[ ${MACH} == "x86_64" ]]; then ARCH=amd64; \
    elif [[ ${MACH} == "armv7l" ]]; then ARCH=arm; \
    else echo "Unsupported arch: ${MACH}"; ARCH=${MACH}; fi; \
    wget -O /usr/local/bin/kubectl "https://dl.k8s.io/release/$(wget -qO - https://dl.k8s.io/release/stable.txt)/bin/linux/${ARCH}/kubectl" && \
    chmod +x /usr/local/bin/kubectl

# Create Non Privileged user
RUN addgroup --gid 1001 botkube && \
    adduser -S --uid 1001 --ingroup botkube botkube

# Run as Non Privileged user
USER botkube

ENTRYPOINT /usr/local/bin/botkube

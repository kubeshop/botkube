FROM alpine:3.15
ARG botkube_version="dev"
LABEL name="botkube" \
    version="${botkube_version}" \
    release="${botkube_version}" \
    summary="BotKube is a messaging bot for monitoring and debugging Kubernetes clusters" \
    maintainer="Prasad Ghangal<prasad.ghangal@gmail.com>" \
    description="BotKube is a messaging bot for monitoring and debugging Kubernetes clusters"

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

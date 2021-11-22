# Copyright (c) 2019 InfraCloud Technologies
#
# Permission is hereby granted, free of charge, to any person obtaining a copy of
# this software and associated documentation files (the "Software"), to deal in
# the Software without restriction, including without limitation the rights to
# use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
# the Software, and to permit persons to whom the Software is furnished to do so,
# subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
# FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
# COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
# IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
# CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

FROM alpine:3.14.2
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

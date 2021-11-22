FROM alpine:3.14.3

ARG botkube_version="dev"

LABEL name="botkube-router" \
    version="${botkube_version}" \
    release="${botkube_version}" \
    maintainer="Lion916" \
    description="BotKube's routing and forwarding service"

COPY botkube-router /usr/local/bin/botkube-router

RUN addgroup --gid 1001 botkube && \
    adduser -S --uid 1001 --ingroup botkube botkube

USER botkube

ENTRYPOINT ["/usr/local/bin/botkube-router"]

CMD  ["web"]
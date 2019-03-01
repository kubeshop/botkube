# Development image
FROM golang:alpine AS BUILD-ENV

RUN mkdir -p /go/src/github.com/infracloudio/botkube/vendor && \
    mkdir -p /go/src/github.com/infracloudio/botkube/cmd && \
    mkdir -p /go/src/github.com/infracloudio/botkube/pkg

COPY vendor/ /go/src/github.com/infracloudio/botkube/vendor
COPY cmd/ /go/src/github.com/infracloudio/botkube/cmd
COPY pkg/ /go/src/github.com/infracloudio/botkube/pkg

RUN cd /go/src/github.com/infracloudio/botkube/cmd/botkube && \
    GOOS=linux GOARCH=amd64 go build -o /go/bin/botkube

ENV KUBE_LATEST_VERSION="v1.13.0"

RUN apk add --no-cache ca-certificates bash git \
    && wget -q https://storage.googleapis.com/kubernetes-release/release/${KUBE_LATEST_VERSION}/bin/linux/amd64/kubectl -O /usr/local/bin/kubectl \
    && chmod +x /usr/local/bin/kubectl

# Production image
FROM alpine

COPY --from=BUILD-ENV /go/bin/botkube /go/bin/botkube
COPY --from=BUILD-ENV /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=BUILD-ENV /usr/local/bin/kubectl /usr/local/bin/kubectl

ENTRYPOINT /go/bin/botkube

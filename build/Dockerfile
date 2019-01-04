FROM golang:1.11-alpine

RUN mkdir -p /go/src/app
WORKDIR /go/src/app

CMD ["/botkube"]

RUN mkdir -p /go/src/github.com/infracloudio/botkube/vendor && \
    mkdir -p /go/src/github.com/infracloudio/botkube/cmd && \
    mkdir -p /go/src/github.com/infracloudio/botkube/pkg

COPY vendor/ /go/src/github.com/infracloudio/botkube/vendor
COPY cmd/ /go/src/github.com/infracloudio/botkube/cmd
COPY pkg/ /go/src/github.com/infracloudio/botkube/pkg

RUN cd /go/src/github.com/infracloudio/botkube/cmd/botkube && \
    go build && \
    cp /go/src/github.com/infracloudio/botkube/cmd/botkube/botkube /botkube

# Install kubectl
ENV KUBE_LATEST_VERSION="v1.13.0"
RUN apk add --no-cache ca-certificates bash git \
    && wget -q https://storage.googleapis.com/kubernetes-release/release/${KUBE_LATEST_VERSION}/bin/linux/amd64/kubectl -O /usr/local/bin/kubectl \
    && chmod +x /usr/local/bin/kubectl \

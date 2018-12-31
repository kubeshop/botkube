FROM golang:1.11-alpine

RUN mkdir -p /go/src/app
WORKDIR /go/src/app

CMD ["/controller"]

RUN mkdir -p /go/src/github.com/infracloudio/kubeops/vendor && \
    mkdir -p /go/src/github.com/infracloudio/kubeops/cmd && \
    mkdir -p /go/src/github.com/infracloudio/kubeops/pkg

COPY vendor/ /go/src/github.com/infracloudio/kubeops/vendor
COPY cmd/ /go/src/github.com/infracloudio/kubeops/cmd
COPY pkg/ /go/src/github.com/infracloudio/kubeops/pkg

RUN cd /go/src/github.com/infracloudio/kubeops/cmd/controller && \
    go build && \
    cp /go/src/github.com/infracloudio/kubeops/cmd/controller/controller /controller

# Install kubectl
ENV KUBE_LATEST_VERSION="v1.13.0"
RUN apk add --no-cache ca-certificates bash git \
    && wget -q https://storage.googleapis.com/kubernetes-release/release/${KUBE_LATEST_VERSION}/bin/linux/amd64/kubectl -O /usr/local/bin/kubectl \
    && chmod +x /usr/local/bin/kubectl \

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

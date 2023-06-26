FROM golang:1.19 as builder
WORKDIR /usr/src/app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -o botkube-cloud-migration -tags build -ldflags '-w' .

FROM gcr.io/distroless/static:latest
COPY --from=builder /usr/src/app/botkube-cloud-migration ./app
CMD ["./app"]

# Example services

An example project that contains the following services:

- **meme** - generate random memes. It takes a random quote from quote service and displays it on a random image.
- **quote** - generate random quotes.

## Prerequisites

- [Go](https://go.dev/doc/install) at least 1.18
- [Docker](https://docs.docker.com/get-docker/)

## Local usage

To run the services locally, follow the steps:

1. Start the quote service:
   ```
   go run cmd/quote/main.go
   ```
   Site can be viewed at [http://localhost:8080/quote](http://localhost:8080/quote).

2. Start the meme service:
    ```
    env QUOTE_URL=http://localhost:8080 go run cmd/meme/main.go
    ```
   Site can be viewed at [http://localhost:9090/meme](http://localhost:9090/meme).

## Docker usage

### Build images

To build the Docker images, run the following commands:

```bash
env GOOS=linux GOARCH=amd64 go build -o meme ./cmd/meme
docker build -t meme:1.0.0 -f build/Dockerfile-meme .

env GOOS=linux GOARCH=amd64 go build -o quote ./cmd/quote
docker build -t quote:1.0.0 -f build/Dockerfile-quote .
```

### Run the images

To run the built Docker images, run the following commands:

```bash
docker network create testing

docker run --name quote -d  --network testing --network-alias quote -p 8080:8080 quote:1.0.0

docker run --name meme --network testing -e QUOTE_URL=http://quote:8080 -p 9090:9090 meme:1.0.0
```

Site can be viewed at [http://localhost:9090/meme](http://localhost:9090/meme)

To clean up, run:

```bash
docker rm -f meme quote
docker network rm testing
```

## Publishing

The images were published manually as:
- [`ghcr.io/kubeshop/botkube/examples/quote-service:v1.1.0`](https://github.com/orgs/kubeshop/packages/container/package/botkube%2Fexamples%2Fquote-service)
- [`ghcr.io/kubeshop/botkube/examples/meme-service:v1.1.0`](https://github.com/orgs/kubeshop/packages/container/package/botkube%2Fexamples%2Fmeme-service)

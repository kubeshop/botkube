# Meme & Quote

Example services contains following services:

- **meme** - generate random memes. It takes a random quote from quote service and displays it on a random image.
- **quote** - generate random quotes.

## Local

1. Start the quote service:
    ```
    go run cmd/quote/main.go
    ```
    Site can be viewed at [http://localhost:8080/quote](http://localhost:8080/quote)

2. Start the meme service
    ```
    env QUOTE_URL=http://localhost:8080 go run cmd/meme/main.go
    ```
    Site can be viewed at [http://localhost:9090/meme](http://localhost:9090/meme)

## Docker

### Build

```bash
env GOOS=linux GOARCH=amd64 go build -o meme ./cmd/meme
docker build -t meme:1.0.0 -f build/Dockerfile-meme .

env GOOS=linux GOARCH=amd64 go build -o quote ./cmd/quote
docker build -t quote:1.0.0 -f build/Dockerfile-quote .
```

### Testing

```bash
docker network create testing

docker run --name quote -d  --network testing --network-alias quote -p 8080:8080 quote:1.0.0

docker run --name meme --network testing -e QUOTE_URL=http://quote:8080 -p 9090:9090 meme:1.0.0
```

Site can be viewed at [http://localhost:9090/meme](http://localhost:9090/meme)

#### Cleanup

```bash
docker rm -f meme quote
docker network rm testing
```

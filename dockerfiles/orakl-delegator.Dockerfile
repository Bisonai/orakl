FROM golang:1.22.3-bullseye as builder
# https://stackoverflow.com/questions/75209245/golang-cross-compiling-with-cgo-inside-docker-image
RUN apt-get update && apt-get install -y curl g++-x86-64-linux-gnu libc6-dev-amd64-cross && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY node node

WORKDIR /app/node

# https://github.com/klaytn/klaytn/issues/197#issuecomment-612597933
RUN CGO_ENABLED=1 CGO_CFLAGS="-O -D__BLST_PORTABLE__" CGO_CFLAGS_ALLOW="-O -D__BLST_PORTABLE__" CC=x86_64-linux-gnu-gcc GOOS=linux GOARCH=amd64 go build -o delegatorbin -ldflags="-w -s" ./cmd/delegator/main.go
# debian:bullseye-slim
FROM debian@sha256:4b48997afc712259da850373fdbc60315316ee72213a4e77fc5a66032d790b2a

RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*

RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz && \
   mv ./migrate /usr/bin

WORKDIR /app

RUN mkdir /app/migrations

COPY --from=builder /app/node/migrations/delegator /app/migrations

COPY --from=builder /app/delegator/delegatorbin /usr/bin

COPY dockerfiles/start-go.sh .

RUN chmod +x start-go.sh

CMD ["./start-go.sh", "delegatorbin"]

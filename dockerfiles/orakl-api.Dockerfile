
FROM golang:1.22.3-bullseye as builder

RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY api api

WORKDIR /app/api

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o apibin -ldflags="-w -s" .

# debian:bullseye-slim
FROM debian@sha256:4b48997afc712259da850373fdbc60315316ee72213a4e77fc5a66032d790b2a

RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*

RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz && \
   mv ./migrate /usr/bin

WORKDIR /app

RUN mkdir /app/migrations

COPY --from=builder /app/api/migrations /app/migrations

COPY --from=builder /app/api/apibin /usr/bin

COPY dockerfiles/start-go.sh .

RUN chmod +x start-go.sh

CMD ["./start-go.sh", "apibin"]

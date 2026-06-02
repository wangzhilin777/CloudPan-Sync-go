FROM golang:1.26 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY web ./web

RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o /out/cloudpan-sync ./cmd/cloudpan-sync

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates sqlite3 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

ENV CLOUDPAN_ADDR=:8080
ENV CLOUDPAN_DATA_DIR=/data
ENV CLOUDPAN_DB_PATH=/data/cloudpan-sync.db
ENV CLOUDPAN_ENV=production
ENV CLOUDPAN_LOG_LEVEL=info

COPY --from=builder /out/cloudpan-sync /usr/local/bin/cloudpan-sync

VOLUME ["/data"]
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/cloudpan-sync"]

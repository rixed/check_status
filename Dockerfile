# ---- build stage ----
FROM debian:trixie-slim AS builder

RUN apt-get update \
 && apt-get install -y --no-install-recommends golang-go ca-certificates \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /build
COPY go.mod .
RUN go mod download
COPY *.go .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=mod -ldflags="-s -w" -o check_status .

# ---- runtime stage ----
FROM debian:trixie-slim

RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates postgresql-client \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /build/check_status .
COPY schema.sql .
COPY entrypoint.sh .
RUN chmod +x entrypoint.sh

EXPOSE 80
ENTRYPOINT ["/app/entrypoint.sh"]

FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git gcc musl-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o /seed ./cmd/seed
RUN CGO_ENABLED=1 go build -o /agent ./cmd/agent
RUN CGO_ENABLED=1 go build -o /founder ./cmd/founder

FROM alpine:3.20

RUN apk add --no-cache ca-certificates
RUN adduser -D -u 1000 musketeers

COPY --from=builder /seed /usr/local/bin/seed
COPY --from=builder /agent /usr/local/bin/agent
COPY --from=builder /founder /usr/local/bin/founder

USER musketeers
WORKDIR /data
VOLUME /data

EXPOSE 4001 8080

ENTRYPOINT ["seed"]
CMD ["-port", "4001", "-data", "/data"]

# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /app/lb ./cmd && \
    CGO_ENABLED=0 go build -o /app/dummy ./dummy_server

# Run stage
FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/lb /app/lb
COPY --from=builder /app/dummy /app/dummy
COPY config.docker.yaml /app/config.yaml

RUN mkdir -p /app/dump

EXPOSE 8080 3000 3001 3002 3003

CMD ["./lb"]

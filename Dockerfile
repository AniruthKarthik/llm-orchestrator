# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o server ./cmd/server/main.go

# Final stage
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/server .
COPY migrations ./migrations

EXPOSE 8080

CMD ["./server"]

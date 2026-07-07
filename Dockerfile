# syntax=docker/dockerfile:1

# Builder stage
FROM golang:1.26-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/server .

# Runtime stage
FROM alpine:3.20 AS runtime
WORKDIR /app

COPY --from=builder /out/server ./server
# JSON fixtures are not compiled, copy them alongside the binary
COPY internal/webhook/jsons ./internal/webhook/jsons
COPY internal/webhook/responses ./internal/webhook/responses

EXPOSE 3000 

CMD ["./server"]

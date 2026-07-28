FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install ca-certificates and git
RUN apk add --no-cache ca-certificates git

# Copy dependency files
COPY go.mod go.sum* ./
RUN go mod download

# Copy source files
COPY . .

# Ensure dependencies and go.sum are complete
RUN go mod tidy

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main .

# Production Runner Stage
FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/main .
COPY --from=builder /app/db ./db

EXPOSE 8080

CMD ["./main"]

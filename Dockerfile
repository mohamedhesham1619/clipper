# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY . .

RUN go mod download

RUN go build -o bin/clipper ./cmd/clipper

# Final stage
FROM alpine:latest

RUN apk add --no-cache yt-dlp ffmpeg

WORKDIR /clipper

COPY --from=builder /app/bin/clipper .

CMD ["./clipper"]
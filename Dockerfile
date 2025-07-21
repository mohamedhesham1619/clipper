# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY . .

RUN go mod download

RUN go build -o bin/clipper ./cmd/clipper

# Final stage
FROM alpine:latest

# Install Python3, pip, and ffmpeg
RUN apk add --no-cache python3 py3-pip ffmpeg

# Install the latest yt-dlp from PyPI (pip)
RUN pip install --upgrade yt-dlp --break-system-packages

WORKDIR /clipper

COPY --from=builder /app/bin/clipper .

CMD ["./clipper"]

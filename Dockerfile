# Build stage
FROM golang:1.22-alpine as builder

WORKDIR /app

COPY . .

RUN go mod download

RUN go build -o app .

# Final stage
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/app .

RUN mkdir -p temp

RUN apk add --no-cache yt-dlp ffmpeg

CMD ["./app"]
FROM golang:1.22-alpine as builder

WORKDIR /app

COPY . .

RUN go mod download

RUN go build -o app .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/app .

RUN mkdir -p /app/temp && chmod 777 /app/temp

RUN apk add --no-cache yt-dlp ffmpeg

EXPOSE 8080

CMD ["./app"]
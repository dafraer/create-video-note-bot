FROM ubuntu:latest

WORKDIR /app

COPY . .

RUN apt-get update && apt-get install -y golang ffmpeg

RUN go mod download

RUN go build -o task ./cmd

EXPOSE 8080

CMD ["sh", "-c", "./task ${TOKEN}"]
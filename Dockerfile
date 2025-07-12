FROM golang:latest

WORKDIR /app

COPY . .

RUN go mod download

RUN go build -o task ./cmd

EXPOSE 8080

CMD ["sh", "-c", "./task ${TOKEN} -w"]
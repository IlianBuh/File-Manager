FROM golang:1.24-alpine3.21

WORKDIR /app

ENV CONFIG_PATH=./config/config.yaml
COPY . .

RUN go mod tidy


CMD go run ./cmd/gateway/main.go
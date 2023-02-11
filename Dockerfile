FROM golang:alpine3.17

WORKDIR /app

COPY main.go main.go
COPY go.mod go.mod

RUN go mod tidy
RUN go build -o app

ENTRYPOINT ["/app/app"]
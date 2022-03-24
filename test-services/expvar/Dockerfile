FROM golang:1.18-alpine

CMD ["./expvar"]
EXPOSE 8080

WORKDIR /code
COPY ./server.go ./server.go

RUN go build -o expvar ./server.go

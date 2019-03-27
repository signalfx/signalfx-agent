FROM golang:1.12.1-alpine

CMD ["./expvar"]
EXPOSE 8080

WORKDIR /code
COPY ./server.go ./server.go

RUN go build -o expvar ./server.go

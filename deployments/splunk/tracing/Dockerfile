FROM golang:1.13-stretch

WORKDIR /go/src/app

COPY main.go .

RUN go get

RUN go build

CMD /go/src/app/app
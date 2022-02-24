FROM golang:1.17.7-stretch

WORKDIR /go/src/app

COPY main.go .

RUN go install

RUN go build

CMD /go/src/app/app
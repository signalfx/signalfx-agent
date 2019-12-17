FROM golang:1.13.4-alpine

CMD ["./fakek8s", "-httptest.serve=0.0.0.0:8443"]
EXPOSE 8443

RUN apk add git
WORKDIR /usr/src/signalfx-agent
COPY ./go.mod go.sum ./
COPY ./cmd/fakek8s/ ./cmd/fakek8s/
COPY ./pkg/neotest/k8s/testhelpers/fakek8s/ ./pkg/neotest/k8s/testhelpers/fakek8s/

RUN go build -o fakek8s ./cmd/fakek8s

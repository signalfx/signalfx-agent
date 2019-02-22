FROM golang:1.11-alpine

CMD ["./fakek8s", "-httptest.serve", "0.0.0.0:8443"]
EXPOSE 8443

WORKDIR /go/src/github.com/signalfx/signalfx-agent
COPY ./vendor ./vendor/
COPY ./cmd/fakek8s/ ./cmd/fakek8s/
COPY ./internal/neotest/k8s/testhelpers/fakek8s/ ./internal/neotest/k8s/testhelpers/fakek8s/

RUN go build -o fakek8s ./cmd/fakek8s

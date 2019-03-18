FROM golang:1.12.1-alpine

CMD ["./fakek8s", "-httptest.serve=0.0.0.0:8443"]
EXPOSE 8443

WORKDIR /usr/src/signalfx-agent
COPY ./vendor/ ./vendor/
COPY ./go.mod go.sum ./
COPY ./cmd/fakek8s/ ./cmd/fakek8s/
COPY ./internal/neotest/k8s/testhelpers/fakek8s/ ./internal/neotest/k8s/testhelpers/fakek8s/

RUN go build -mod vendor -o fakek8s ./cmd/fakek8s

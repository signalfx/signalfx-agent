FROM golang:1.12.1-alpine

CMD ["./fakek8s", "-httptest.serve=0.0.0.0:8443"]
EXPOSE 8443

RUN apk add git
WORKDIR /usr/src/signalfx-agent
COPY ./go.mod go.sum ./
COPY ./cmd/fakek8s/ ./cmd/fakek8s/
COPY ./internal/neotest/k8s/testhelpers/fakek8s/ ./internal/neotest/k8s/testhelpers/fakek8s/

RUN go build -o fakek8s ./cmd/fakek8s

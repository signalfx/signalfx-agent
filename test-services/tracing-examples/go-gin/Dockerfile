ARG GO_VERSION=1.12
FROM golang:$GO_VERSION

RUN go get -v github.com/signalfx/signalfx-go-tracing/tracing
RUN go get -v github.com/signalfx/tracing-examples/signalfx-tracing/signalfx-go-tracing/gin/player
RUN go get -v github.com/signalfx/tracing-examples/signalfx-tracing/signalfx-go-tracing/gin/server

COPY docker-entrypoint.sh /
RUN chmod a+x /docker-entrypoint.sh

WORKDIR /go/src/github.com/signalfx/tracing-examples/signalfx-tracing/signalfx-go-tracing/gin
ENTRYPOINT ["/docker-entrypoint.sh"]

#!/bin/bash -e

TRACINGENDPOINT=${TRACINGENDPOINT:-"http://localhost:9080/v1/trace"}
MONGOHOST=${MONGOHOST:-"localhost"}
MONGODRIVER=${MONGODRIVER:-"mongo"}

if [ $# -eq 0 ]; then
    sed -i "s|TracingEndpoint = .*|TracingEndpoint = \"$TRACINGENDPOINT\"|" server/server.go
    sed -i "s|MongoHost = .*|MongoHost = \"$MONGOHOST\"|" server/server.go
    sed -i "s|MongoDriver = .*|MongoDriver = \"$MONGODRIVER\"|" server/server.go
    go run ./server/server.go
else
    exec "$@"
fi

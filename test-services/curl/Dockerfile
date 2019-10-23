FROM alpine:3.8

RUN apk add --no-cache curl
# Expose something so observers will find this container
EXPOSE 8080

ENTRYPOINT ["/usr/bin/curl"]

FROM alpine:3.9

COPY socat /bin/socat

RUN apk add --no-cache ca-certificates

# Insert our fake certs to the system bundle so they are trusted
COPY certs/*.signalfx.com.* /
COPY certs/*.signalfx.com.* /usr/local/share/ca-certificates/
RUN update-ca-certificates

RUN echo "hosts: files dns" > /etc/nsswitch.conf

FROM ubuntu:20.04
ENV DEBIAN_FRONTEND=noninteractive
ADD https://github.com/just-containers/s6-overlay/releases/download/v1.21.8.0/s6-overlay-amd64.tar.gz /tmp/
RUN tar xzf /tmp/s6-overlay-amd64.tar.gz -C / --exclude='./bin' && tar xzf /tmp/s6-overlay-amd64.tar.gz -C /usr ./bin
RUN apt update && \
    apt install -y nginx && \
    apt install -y php-fpm && \
    rm -f /etc/nginx/sites-enabled/default
COPY status.conf /etc/nginx/sites-enabled/status.conf
COPY services.d /etc/services.d
RUN VERSION=$(find /usr/*bin/* -name 'php-fpm*' -type f -printf "%f" | sed 's/^php-fpm//') && \
    sed -i "s/{VERSION}/${VERSION}/g" /etc/nginx/sites-enabled/status.conf /etc/services.d/php-fpm/run && \
    sed -i 's/^;pm\.status/pm.status/' /etc/php/${VERSION}/fpm/pool.d/www.conf
ENTRYPOINT ["/init"]

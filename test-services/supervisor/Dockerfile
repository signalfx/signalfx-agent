FROM ubuntu:20.04
ENV DEBIAN_FRONTEND=noninteractive
RUN apt update && \
    apt install -y supervisor
COPY conf.d /etc/supervisor/conf.d
COPY long.sh /usr/local/bin/long.sh
RUN echo "[inet_http_server]" >> /etc/supervisor/supervisord.conf && \
    echo "port=0.0.0.0:9001" >> /etc/supervisor/supervisord.conf
CMD [ "bash", "-c", "/usr/bin/supervisord -n -c /etc/supervisor/supervisord.conf"]

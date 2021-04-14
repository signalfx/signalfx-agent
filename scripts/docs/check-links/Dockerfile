FROM ghcr.io/tcort/markdown-link-check:3.8.5

RUN apk add bash

VOLUME /usr/src/signalfx-agent
WORKDIR /usr/src/signalfx-agent

COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod a+x /docker-entrypoint.sh

ENV PATH=/src:$PATH

ENTRYPOINT ["/docker-entrypoint.sh"]

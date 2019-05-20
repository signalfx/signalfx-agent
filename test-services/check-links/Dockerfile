FROM node:8.16.0-alpine

RUN apk add bash

RUN npm install -g markdown-link-check

VOLUME /usr/src/signalfx-agent
WORKDIR /usr/src/signalfx-agent

COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod a+x /docker-entrypoint.sh

ENTRYPOINT ["/docker-entrypoint.sh"]

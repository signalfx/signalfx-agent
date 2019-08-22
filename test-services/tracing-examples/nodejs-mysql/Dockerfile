ARG NODE_VERSION=8
FROM node:$NODE_VERSION

WORKDIR /signalfx
RUN git clone https://github.com/signalfx/tracing-examples

WORKDIR /signalfx/tracing-examples/signalfx-tracing/signalfx-nodejs-tracing/mysql
RUN npm install

COPY docker-entrypoint.sh /
RUN chmod a+x /docker-entrypoint.sh

ENTRYPOINT ["/docker-entrypoint.sh"]

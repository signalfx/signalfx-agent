ARG NODE_VERSION=8
FROM node:$NODE_VERSION

WORKDIR /signalfx
RUN git clone https://github.com/signalfx/tracing-examples

WORKDIR /signalfx/tracing-examples/signalfx-tracing/signalfx-nodejs-tracing/express
RUN npm install

CMD npm start

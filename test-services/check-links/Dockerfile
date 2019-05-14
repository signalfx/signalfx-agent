FROM node:8.16.0-alpine

RUN apk add bash

#RUN apt-get update && \
#    apt-get install -y npm

RUN npm install -g markdown-link-check

COPY run.sh /usr/local/bin/run.sh
RUN chmod a+x /usr/local/bin/run.sh

VOLUME /usr/src/signalfx-agent

WORKDIR /usr/src/signalfx-agent

CMD ["/usr/local/bin/run.sh", "/usr/src/signalfx-agent"]

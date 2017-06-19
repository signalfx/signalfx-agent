FROM ubuntu:16.04

ENV DEBIAN_FRONTEND noninteractive
ENV SIGNALFX_CONFIGURE_MONITORING_COMMANDS /opt/signalfx-imaging/signalfx-agent-setup.sh
ENV SIGNALFX_LAUNCH_MONITORING_COMMANDS /usr/sbin/signalfx-agent
ENV LD_LIBRARY_PATH /usr/lib:/usr/lib/collectd:/usr/lib/jvm/java-8-openjdk-amd64/jre/lib/amd64/server

RUN sed -i -e '/^deb-src/d' /etc/apt/sources.list \
    && apt-get update \
    && apt-get install -y \
      software-properties-common \
      python-software-properties \
      curl \
      libcurl4-gnutls-dev \
      openjdk-8-jre-headless \
      wget \
      net-tools \
      libpython2.7 \
      libmysqlclient-dev \
      vim \
      python-pip \
    && apt-get install -y libcurl4-openssl-dev
RUN add-apt-repository -y ppa:signalfx/collectd-release \
    && apt-get update \
    && apt-get install -y --no-install-recommends collectd

LABEL agent.version="$signalfx_agent_version" \
      app="signalfx-agent"

# base
COPY *.sh /opt/signalfx-imaging/

RUN chmod +x /opt/signalfx-imaging/*.sh

# collectd-all
# collectd
RUN mkdir -p /etc/collectd/managed_config \
    && mkdir -p /etc/collectd/filtering_config

# collectd-cpu
# collectd-docker
ENV DOCKER_COLLECTD_PLUGIN_V=1.0.6
RUN mkdir -p /usr/share/collectd/docker-collectd-plugin \
    && curl -L "https://github.com/signalfx/docker-collectd-plugin/archive/v${DOCKER_COLLECTD_PLUGIN_V}.tar.gz" --output /usr/share/collectd/docker-collectd-plugin/v${DOCKER_COLLECTD_PLUGIN_V}.tar.gz \
    && tar -zxvf /usr/share/collectd/docker-collectd-plugin/v${DOCKER_COLLECTD_PLUGIN_V}.tar.gz -C /usr/share/collectd/docker-collectd-plugin/ \
    && cp /usr/share/collectd/docker-collectd-plugin/docker-collectd-plugin-${DOCKER_COLLECTD_PLUGIN_V}/* /usr/share/collectd/docker-collectd-plugin/ \
    && rm -f /usr/share/collectd/docker-collectd-plugin/v${DOCKER_COLLECTD_PLUGIN_V}.tar.gz \
    && rm -rf /usr/share/collectd/docker-collectd-plugin/docker-collectd-plugin-${DOCKER_COLLECTD_PLUGIN_V} \
    && pip install -r /usr/share/collectd/docker-collectd-plugin/requirements.txt

# collectd-elasticsearch
ENV ELASTICSEARCH_COLLECTD_PLUGIN_V=1.3.2
RUN mkdir -p /usr/share/collectd/elasticsearch-collectd-plugin \
    && curl -L "https://github.com/signalfx/collectd-elasticsearch/archive/v${ELASTICSEARCH_COLLECTD_PLUGIN_V}.tar.gz" --output /usr/share/collectd/elasticsearch-collectd-plugin/v${ELASTICSEARCH_COLLECTD_PLUGIN_V}.tar.gz \
    && tar -zxvf /usr/share/collectd/elasticsearch-collectd-plugin/v${ELASTICSEARCH_COLLECTD_PLUGIN_V}.tar.gz -C /usr/share/collectd/elasticsearch-collectd-plugin/ \
    && cp -r /usr/share/collectd/elasticsearch-collectd-plugin/collectd-elasticsearch-${ELASTICSEARCH_COLLECTD_PLUGIN_V}/* /usr/share/collectd/elasticsearch-collectd-plugin/ \
    && rm -f /usr/share/collectd/elasticsearch-collectd-plugin/v${ELASTICSEARCH_COLLECTD_PLUGIN_V}.tar.gz \
    && rm -rf /usr/share/collectd/elasticsearch-collectd-plugin/collectd-elasticsearch-${ELASTICSEARCH_COLLECTD_PLUGIN_V}

# collectd-java
RUN mkdir -p /usr/share/collectd/java \
    && echo "jmx_memory      value:GAUGE:0:U" > /usr/share/collectd/java/signalfx_types_db

# collectd-mesos
ENV MESOS_COLLECTD_PLUGIN_V=1.1.1
RUN mkdir -p /usr/share/collectd/mesos-collectd-plugin \
    && curl -L "https://github.com/signalfx/collectd-mesos/archive/v${MESOS_COLLECTD_PLUGIN_V}.tar.gz" --output /usr/share/collectd/mesos-collectd-plugin/v${MESOS_COLLECTD_PLUGIN_V}.tar.gz \
    && tar -zxvf /usr/share/collectd/mesos-collectd-plugin/v${MESOS_COLLECTD_PLUGIN_V}.tar.gz -C /usr/share/collectd/mesos-collectd-plugin/ \
    && cp /usr/share/collectd/mesos-collectd-plugin/collectd-mesos-${MESOS_COLLECTD_PLUGIN_V}/* /usr/share/collectd/mesos-collectd-plugin/ \
    && rm -f /usr/share/collectd/mesos-collectd-plugin/v${MESOS_COLLECTD_PLUGIN_V}.tar.gz \
    && rm -rf /usr/share/collectd/mesos-collectd-plugin/collectd-mesos-${MESOS_COLLECTD_PLUGIN_V}

# collectd-marathon
ENV MARATHON_COLLECTD_PLUGIN_V=1.0.1
RUN mkdir -p /usr/share/collectd/marathon-collectd-plugin \
    && curl -L "https://github.com/signalfx/collectd-marathon/archive/v${MARATHON_COLLECTD_PLUGIN_V}.tar.gz" --output /usr/share/collectd/marathon-collectd-plugin/v${MARATHON_COLLECTD_PLUGIN_V}.tar.gz \
    && tar -zxvf /usr/share/collectd/marathon-collectd-plugin/v${MARATHON_COLLECTD_PLUGIN_V}.tar.gz -C /usr/share/collectd/marathon-collectd-plugin/ \
    && cp /usr/share/collectd/marathon-collectd-plugin/collectd-marathon-${MARATHON_COLLECTD_PLUGIN_V}/* /usr/share/collectd/marathon-collectd-plugin/ \
    && rm -f /usr/share/collectd/marathon-collectd-plugin/v${MARATHON_COLLECTD_PLUGIN_V}.tar.gz \
    && rm -rf /usr/share/collectd/marathon-collectd-plugin/collectd-marathon-${MARATHON_COLLECTD_PLUGIN_V}

# collectd-mongodb
ENV MONGODB_COLLECTD_PLUGIN_V=1.1.2
ENV PYMONGO_V=3.1.1
RUN mkdir -p /usr/share/collectd/mongodb-collectd-plugin \
    && curl -L "https://github.com/signalfx/collectd-mongodb/archive/v${MONGODB_COLLECTD_PLUGIN_V}.tar.gz" --output /usr/share/collectd/mongodb-collectd-plugin/v${MONGODB_COLLECTD_PLUGIN_V}.tar.gz \
    && tar -zxvf /usr/share/collectd/mongodb-collectd-plugin/v${MONGODB_COLLECTD_PLUGIN_V}.tar.gz -C /usr/share/collectd/mongodb-collectd-plugin/ \
    && cp /usr/share/collectd/mongodb-collectd-plugin/collectd-mongodb-${MONGODB_COLLECTD_PLUGIN_V}/* /usr/share/collectd/mongodb-collectd-plugin/ \
    && rm -f /usr/share/collectd/mongodb-collectd-plugin/v${MONGODB_COLLECTD_PLUGIN_V}.tar.gz \
    && rm -rf /usr/share/collectd/mongodb-collectd-plugin/collectd-mongodb-${MONGODB_COLLECTD_PLUGIN_V} \
    && pip install pymongo==${PYMONGO_V}

# collectd-rabbitmq
ENV RABBITMQ_COLLECTD_PLUGIN_V=1.0.1
RUN mkdir -p /usr/share/collectd/rabbitmq-collectd-plugin \
    && curl -L "https://github.com/signalfx/collectd-rabbitmq/archive/v${RABBITMQ_COLLECTD_PLUGIN_V}.tar.gz" --output /usr/share/collectd/rabbitmq-collectd-plugin/v${RABBITMQ_COLLECTD_PLUGIN_V}.tar.gz \
    && tar -zxvf /usr/share/collectd/rabbitmq-collectd-plugin/v${RABBITMQ_COLLECTD_PLUGIN_V}.tar.gz -C /usr/share/collectd/rabbitmq-collectd-plugin/ \
    && cp -r /usr/share/collectd/rabbitmq-collectd-plugin/collectd-rabbitmq-${RABBITMQ_COLLECTD_PLUGIN_V}/* /usr/share/collectd/rabbitmq-collectd-plugin/ \
    && rm -f /usr/share/collectd/rabbitmq-collectd-plugin/v${RABBITMQ_COLLECTD_PLUGIN_V}.tar.gz \
    && rm -rf /usr/share/collectd/rabbitmq-collectd-plugin/collectd-rabbitmq-${RABBITMQ_COLLECTD_PLUGIN_V}

# collectd-redis
ENV REDIS_COLLECTD_PLUGIN_V=1.0.0
RUN mkdir -p /usr/share/collectd/redis-collectd-plugin \
    && curl -L "https://github.com/signalfx/redis-collectd-plugin/archive/v${REDIS_COLLECTD_PLUGIN_V}.tar.gz" --output /usr/share/collectd/redis-collectd-plugin/v${REDIS_COLLECTD_PLUGIN_V}.tar.gz \
    && tar -zxvf /usr/share/collectd/redis-collectd-plugin/v${REDIS_COLLECTD_PLUGIN_V}.tar.gz -C /usr/share/collectd/redis-collectd-plugin/ \
    && cp -r /usr/share/collectd/redis-collectd-plugin/redis-collectd-plugin-${REDIS_COLLECTD_PLUGIN_V}/* /usr/share/collectd/redis-collectd-plugin/ \
    && rm -f /usr/share/collectd/redis-collectd-plugin/v${REDIS_COLLECTD_PLUGIN_V}.tar.gz \
    && rm -rf /usr/share/collectd/redis-collectd-plugin/redis-collectd-plugin-${REDIS_COLLECTD_PLUGIN_V} \
    && rm -rf /usr/share/collectd/redis-collectd-plugin/{screenshots,redis.conf}

# signalfx-collectd-plugin
ENV SIGNALFX_COLLECTD_PLUGIN_V=0.0.31
RUN mkdir -p /usr/share/collectd/signalfx-collectd-plugin \
    && curl -L "https://github.com/signalfx/signalfx-collectd-plugin/archive/v${SIGNALFX_COLLECTD_PLUGIN_V}.tar.gz" --output /usr/share/collectd/signalfx-collectd-plugin/v${SIGNALFX_COLLECTD_PLUGIN_V}.tar.gz \
    && tar -zxvf /usr/share/collectd/signalfx-collectd-plugin/v${SIGNALFX_COLLECTD_PLUGIN_V}.tar.gz -C /usr/share/collectd/signalfx-collectd-plugin/ \
    && cp /usr/share/collectd/signalfx-collectd-plugin/signalfx-collectd-plugin-${SIGNALFX_COLLECTD_PLUGIN_V}/requirements.txt /usr/share/collectd/signalfx-collectd-plugin/ \
    && cp /usr/share/collectd/signalfx-collectd-plugin/signalfx-collectd-plugin-${SIGNALFX_COLLECTD_PLUGIN_V}/types.db.plugin /usr/share/collectd/signalfx-collectd-plugin/ \
    && cp /usr/share/collectd/signalfx-collectd-plugin/signalfx-collectd-plugin-${SIGNALFX_COLLECTD_PLUGIN_V}/src/* /usr/share/collectd/signalfx-collectd-plugin/ \
    && rm -f /usr/share/collectd/signalfx-collectd-plugin/v${SIGNALFX_COLLECTD_PLUGIN_V}.tar.gz \
    && rm -rf /usr/share/collectd/signalfx-collectd-plugin/signalfx-collectd-plugin-${SIGNALFX_COLLECTD_PLUGIN_V} \
    && pip install -r /usr/share/collectd/signalfx-collectd-plugin/requirements.txt

# collectd-zookeeper
ENV ZOOKEEPER_COLLECTD_PLUGIN_V=0.0.3
RUN mkdir -p /usr/share/collectd/zookeeper-collectd-plugin \
    && curl -L "https://github.com/signalfx/collectd-zookeeper/archive/v${ZOOKEEPER_COLLECTD_PLUGIN_V}.tar.gz" --output /usr/share/collectd/zookeeper-collectd-plugin/v${ZOOKEEPER_COLLECTD_PLUGIN_V}.tar.gz \
    && tar -zxvf /usr/share/collectd/zookeeper-collectd-plugin/v${ZOOKEEPER_COLLECTD_PLUGIN_V}.tar.gz -C /usr/share/collectd/zookeeper-collectd-plugin/ \
    && cp /usr/share/collectd/zookeeper-collectd-plugin/collectd-zookeeper-${ZOOKEEPER_COLLECTD_PLUGIN_V}/* /usr/share/collectd/zookeeper-collectd-plugin/ \
    && rm -f /usr/share/collectd/zookeeper-collectd-plugin/v${ZOOKEEPER_COLLECTD_PLUGIN_V}.tar.gz \
    && rm -rf /usr/share/collectd/zookeeper-collectd-plugin/collectd-zookeeper-${ZOOKEEPER_COLLECTD_PLUGIN_V}

# collectd-all-cleanup
RUN apt-get remove -y --purge build-essential software-properties-common python-software-properties python-pip \
    && apt-get autoremove -y --purge \
    && rm -rf /var/lib/apt/lists/*

# signalfx-agent
ARG signalfx_agent_version=1.0.0-beta-9

COPY etc /etc/signalfx/
COPY agent /usr/sbin/signalfx-agent
COPY libcollectd.so /usr/local/lib/collectd/libcollectd.so
COPY java.so /usr/lib/collectd/java.so
COPY mysql.so /usr/lib/collectd/mysql.so
COPY nginx.so /usr/lib/collectd/nginx.so
COPY python.so /usr/lib/collectd/python.so
COPY aggregation.so /usr/lib/collectd/aggregation.so
COPY generic-jmx.jar /usr/share/collectd/java/generic-jmx.jar

RUN chmod +x /usr/sbin/signalfx-agent \
    && rm -f /etc/collectd/collectd.conf \
    && rm -rf /etc/collectd/managed_config/* \
    && rm -rf /etc/collectd/filter_config/*

# default startup command
CMD ["/opt/signalfx-imaging/signalfx-agent.sh"]

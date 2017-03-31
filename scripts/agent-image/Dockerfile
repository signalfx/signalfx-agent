FROM ubuntu:16.04

ENV DEBIAN_FRONTEND noninteractive
ENV SIGNALFX_CONFIGURE_MONITORING_COMMANDS /opt/signalfx-imaging/signalfx-agent-setup.sh
ENV SIGNALFX_LAUNCH_MONITORING_COMMANDS /usr/sbin/signalfx-agent
ENV LD_LIBRARY_PATH /usr/lib:/usr/lib/collectd:/usr/lib/jvm/java-8-openjdk-amd64/jre/lib/amd64/server

RUN sed -i -e '/^deb-src/d' /etc/apt/sources.list \
    && apt-get update \
    && apt-get install -y software-properties-common \
    && apt-get install -y python-software-properties \
    && apt-get install -y dpkg \
    && apt-get install -y curl \
    && apt-get install -y libcurl4-gnutls-dev \
    && apt-get install -y libcurl4-openssl-dev \
    && apt-get install -y wget \
    && apt-get install -y net-tools \
    && apt-get install -y libpython2.7 \
    && apt-get install -y python-pip \
    && add-apt-repository -y ppa:signalfx/collectd-release \
    && add-apt-repository -y ppa:signalfx/collectd-plugin-release \
    && apt-get update \
    && apt-get install -y --no-install-recommends collectd \
    && apt-get install -y signalfx-collectd-plugin \
    && apt-get install -y openjdk-8-jre-headless

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
ENV DOCKER_COLLECTD_PLUGIN_V=1.0.3
RUN mkdir -p /usr/share/collectd/docker-collectd-plugin \
    && curl -L "https://github.com/signalfx/docker-collectd-plugin/archive/v${DOCKER_COLLECTD_PLUGIN_V}.tar.gz" --output /usr/share/collectd/docker-collectd-plugin/v${DOCKER_COLLECTD_PLUGIN_V}.tar.gz \
    && tar -zxvf /usr/share/collectd/docker-collectd-plugin/v${DOCKER_COLLECTD_PLUGIN_V}.tar.gz -C /usr/share/collectd/docker-collectd-plugin/ \
    && cp /usr/share/collectd/docker-collectd-plugin/docker-collectd-plugin-${DOCKER_COLLECTD_PLUGIN_V}/* /usr/share/collectd/docker-collectd-plugin/ \
    && rm -f /usr/share/collectd/docker-collectd-plugin/v${DOCKER_COLLECTD_PLUGIN_V}.tar.gz \
    && rm -rf /usr/share/collectd/docker-collectd-plugin/docker-collectd-plugin-${DOCKER_COLLECTD_PLUGIN_V} \
    && pip install -r /usr/share/collectd/docker-collectd-plugin/requirements.txt

# collectd-elasticsearch
ENV ELASTICSEARCH_COLLECTD_PLUGIN_V=1.2.0
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
ENV MESOS_AGENT_COLLECTD_PLUGIN_V=1.1.0
RUN mkdir -p /usr/share/collectd/mesos-collectd-plugin \
    && curl -L "https://github.com/signalfx/collectd-mesos/archive/v${MESOS_AGENT_COLLECTD_PLUGIN_V}.tar.gz" --output /usr/share/collectd/mesos-collectd-plugin/v${MESOS_AGENT_COLLECTD_PLUGIN_V}.tar.gz \
    && tar -zxvf /usr/share/collectd/mesos-collectd-plugin/v${MESOS_AGENT_COLLECTD_PLUGIN_V}.tar.gz -C /usr/share/collectd/mesos-collectd-plugin/ \
    && cp /usr/share/collectd/mesos-collectd-plugin/collectd-mesos-${MESOS_AGENT_COLLECTD_PLUGIN_V}/* /usr/share/collectd/mesos-collectd-plugin/ \
    && rm -f /usr/share/collectd/mesos-collectd-plugin/v${MESOS_AGENT_COLLECTD_PLUGIN_V}.tar.gz \
    && rm -rf /usr/share/collectd/mesos-collectd-plugin/collectd-mesos-${MESOS_AGENT_COLLECTD_PLUGIN_V}

# collectd-mesos-master
ENV MESOS_MASTER_COLLECTD_PLUGIN_V=1.1.0
RUN mkdir -p /usr/share/collectd/mesos-collectd-plugin \
    && curl -L "https://github.com/signalfx/collectd-mesos/archive/v${MESOS_MASTER_COLLECTD_PLUGIN_V}.tar.gz" --output /usr/share/collectd/mesos-collectd-plugin/v${MESOS_MASTER_COLLECTD_PLUGIN_V}.tar.gz \
    && tar -zxvf /usr/share/collectd/mesos-collectd-plugin/v${MESOS_MASTER_COLLECTD_PLUGIN_V}.tar.gz -C /usr/share/collectd/mesos-collectd-plugin/ \
    && cp /usr/share/collectd/mesos-collectd-plugin/collectd-mesos-${MESOS_MASTER_COLLECTD_PLUGIN_V}/* /usr/share/collectd/mesos-collectd-plugin/ \
    && rm -f /usr/share/collectd/mesos-collectd-plugin/v${MESOS_MASTER_COLLECTD_PLUGIN_V}.tar.gz \
    && rm -rf /usr/share/collectd/mesos-collectd-plugin/collectd-mesos-${MESOS_MASTER_COLLECTD_PLUGIN_V}

# collectd-mongodb
ENV MONGODB_COLLECTD_PLUGIN_V=1.1.1
ENV PYMONGO_V=3.1.1
RUN mkdir -p /usr/share/collectd/mongodb-collectd-plugin \
    && curl -L "https://github.com/signalfx/collectd-mongodb/archive/v${MONGODB_COLLECTD_PLUGIN_V}.tar.gz" --output /usr/share/collectd/mongodb-collectd-plugin/v${MONGODB_COLLECTD_PLUGIN_V}.tar.gz \
    && tar -zxvf /usr/share/collectd/mongodb-collectd-plugin/v${MONGODB_COLLECTD_PLUGIN_V}.tar.gz -C /usr/share/collectd/mongodb-collectd-plugin/ \
    && cp /usr/share/collectd/mongodb-collectd-plugin/collectd-mongodb-${MONGODB_COLLECTD_PLUGIN_V}/* /usr/share/collectd/mongodb-collectd-plugin/ \
    && rm -f /usr/share/collectd/mongodb-collectd-plugin/v${MONGODB_COLLECTD_PLUGIN_V}.tar.gz \
    && rm -rf /usr/share/collectd/mongodb-collectd-plugin/collectd-mongodb-${MONGODB_COLLECTD_PLUGIN_V} \
    && pip install pymongo==${PYMONGO_V}

# collectd-redis
RUN mkdir -p /usr/share/collectd/redis-collectd-plugin \
    && curl -L https://raw.githubusercontent.com/signalfx/redis-collectd-plugin/master/redis_info.py --output /usr/share/collectd/redis-collectd-plugin/redis_info.py

# collectd-zookeeper
ENV ZOOKEEPER_COLLECTD_PLUGIN_V=0.0.2
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

# Have signalfx-collectd-plugin share same location as the rest of the plugins.
RUN ln -s /opt/signalfx-collectd-plugin /usr/share/collectd

# signalfx-agent
ARG signalfx_agent_version=0.0.1-beta

COPY etc /etc/signalfx/
COPY agent /usr/sbin/signalfx-agent
COPY libcollectd.so /usr/local/lib/collectd/libcollectd.so
COPY java.so /usr/lib/collectd/java.so
COPY nginx.so /usr/lib/collectd/nginx.so
COPY python.so /usr/lib/collectd/python.so
COPY aggregation.so /usr/lib/collectd/aggregation.so

RUN chmod +x /usr/sbin/signalfx-agent \
    && rm -f /etc/collectd/collectd.conf \
    && rm -rf /etc/collectd/managed_config/* \
    && rm -rf /etc/collectd/filter_config/*

# default startup command
CMD ["/opt/signalfx-imaging/signalfx-agent.sh"]

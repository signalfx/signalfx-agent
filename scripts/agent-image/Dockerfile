FROM ubuntu:16.04

ENV DEBIAN_FRONTEND noninteractive
ENV SIGNALFX_CONFIGURE_MONITORING_COMMANDS /opt/signalfx-imaging/signalfx-agent-setup.sh
ENV SIGNALFX_LAUNCH_MONITORING_COMMANDS /usr/sbin/signalfx-agent

RUN sed -i -e '/^deb-src/d' /etc/apt/sources.list \
    && apt-get update \
    && apt-get install -y software-properties-common \
    && apt-get install -y python-software-properties \
    && apt-get install -y dpkg \
    && apt-get install -y curl \
    && apt-get install -y wget \
    && apt-get install -y net-tools \
    && add-apt-repository -y ppa:signalfx/collectd-release \
    && apt-get update \
    && apt-get install -y --no-install-recommends collectd \
    && apt-get install -y libpython2.7 \
    && apt-get install -y python-pip \
    && apt-get install -y python-pip \
    && add-apt-repository -y ppa:signalfx/collectd-plugin-release \
    && apt-get update \
    && apt-get install -y signalfx-collectd-plugin \
    && apt-get install -y openjdk-8-jre-headless

LABEL com.signalfx.agent.version="$signalfx_agent_version" \
      com.signalfx.plugin.docker.version="1.0.0" \
      com.signalfx.plugin.elasticsearch.version="1.2.0" \
      com.signalfx.plugin.mesos.version="1.1.0" \
      com.signalfx.plugin.mongodb.version="1.0.9" \
      com.signalfx.plugin.zookeeper.version="0.0.2" \
      java_version="1.8"

# base
COPY *.sh /opt/signalfx-imaging/

RUN chmod +x /opt/signalfx-imaging/*.sh

# collectd-all
# collectd
RUN mkdir -p /etc/collectd/managed_config \
    && mkdir -p /etc/collectd/filtering_config

# collectd-cpu
# collectd-docker
RUN mkdir -p /usr/share/collectd/docker-collectd-plugin \
    && curl -L "https://github.com/signalfx/docker-collectd-plugin/archive/v1.0.0.tar.gz" --output /usr/share/collectd/docker-collectd-plugin/v1.0.0.tar.gz \
    && tar -zxvf /usr/share/collectd/docker-collectd-plugin/v1.0.0.tar.gz -C /usr/share/collectd/docker-collectd-plugin/ \
    && cp /usr/share/collectd/docker-collectd-plugin/docker-collectd-plugin-1.0.0/* /usr/share/collectd/docker-collectd-plugin/ \
    && rm -f /usr/share/collectd/docker-collectd-plugin/v1.0.0.tar.gz \
    && rm -rf /usr/share/collectd/docker-collectd-plugin/docker-collectd-plugin-1.0.0 \
    && pip install -r /usr/share/collectd/docker-collectd-plugin/requirements.txt

# collectd-elasticsearch
RUN mkdir -p /usr/share/collectd/elasticsearch-collectd-plugin \
    && curl -L "https://github.com/signalfx/collectd-elasticsearch/archive/v1.2.0.tar.gz" --output /usr/share/collectd/elasticsearch-collectd-plugin/v1.2.0.tar.gz \
    && tar -zxvf /usr/share/collectd/elasticsearch-collectd-plugin/v1.2.0.tar.gz -C /usr/share/collectd/elasticsearch-collectd-plugin/ \
    && cp -r /usr/share/collectd/elasticsearch-collectd-plugin/collectd-elasticsearch-1.2.0/* /usr/share/collectd/elasticsearch-collectd-plugin/ \
    && rm -f /usr/share/collectd/elasticsearch-collectd-plugin/v1.2.0.tar.gz \
    && rm -rf /usr/share/collectd/elasticsearch-collectd-plugin/collectd-elasticsearch-1.2.0

# collectd-java
RUN mkdir -p /usr/share/collectd/java \
    && echo "jmx_memory      value:GAUGE:0:U" > /usr/share/collectd/java/signalfx_types_db

# collectd-mesos
RUN mkdir -p /usr/share/collectd/mesos-collectd-plugin \
    && curl -L "https://github.com/signalfx/collectd-mesos/archive/v1.1.0.tar.gz" --output /usr/share/collectd/mesos-collectd-plugin/v1.1.0.tar.gz \
    && tar -zxvf /usr/share/collectd/mesos-collectd-plugin/v1.1.0.tar.gz -C /usr/share/collectd/mesos-collectd-plugin/ \
    && cp /usr/share/collectd/mesos-collectd-plugin/collectd-mesos-1.1.0/* /usr/share/collectd/mesos-collectd-plugin/ \
    && rm -f /usr/share/collectd/mesos-collectd-plugin/v1.1.0.tar.gz \
    && rm -rf /usr/share/collectd/mesos-collectd-plugin/collectd-mesos-1.1.0

# collectd-mesos-master
RUN mkdir -p /usr/share/collectd/mesos-collectd-plugin \
    && curl -L "https://github.com/signalfx/collectd-mesos/archive/v1.1.0.tar.gz" --output /usr/share/collectd/mesos-collectd-plugin/v1.1.0.tar.gz \
    && tar -zxvf /usr/share/collectd/mesos-collectd-plugin/v1.1.0.tar.gz -C /usr/share/collectd/mesos-collectd-plugin/ \
    && cp /usr/share/collectd/mesos-collectd-plugin/collectd-mesos-1.1.0/* /usr/share/collectd/mesos-collectd-plugin/ \
    && rm -f /usr/share/collectd/mesos-collectd-plugin/v1.1.0.tar.gz \
    && rm -rf /usr/share/collectd/mesos-collectd-plugin/collectd-mesos-1.1.0

# collectd-mongodb
RUN mkdir -p /usr/share/collectd/mongodb-collectd-plugin \
    && curl -L "https://github.com/signalfx/collectd-mongodb/archive/v1.0.9.tar.gz" --output /usr/share/collectd/mongodb-collectd-plugin/v1.0.9.tar.gz \
    && tar -zxvf /usr/share/collectd/mongodb-collectd-plugin/v1.0.9.tar.gz -C /usr/share/collectd/mongodb-collectd-plugin/ \
    && cp /usr/share/collectd/mongodb-collectd-plugin/collectd-mongodb-1.0.9/* /usr/share/collectd/mongodb-collectd-plugin/ \
    && rm -f /usr/share/collectd/mongodb-collectd-plugin/v1.0.9.tar.gz \
    && rm -rf /usr/share/collectd/mongodb-collectd-plugin/collectd-mongodb-1.0.9 \
    && pip install pymongo==3.1.1

# collectd-redis
RUN mkdir -p /usr/share/collectd/redis-collectd-plugin \
    && curl -L https://raw.githubusercontent.com/signalfx/redis-collectd-plugin/master/redis_info.py --output /usr/share/collectd/redis-collectd-plugin/redis_info.py

# collectd-zookeeper
RUN mkdir -p /usr/share/collectd/zookeeper-collectd-plugin \
    && curl -L "https://github.com/signalfx/collectd-zookeeper/archive/v0.0.2.tar.gz" --output /usr/share/collectd/zookeeper-collectd-plugin/v0.0.2.tar.gz \
    && tar -zxvf /usr/share/collectd/zookeeper-collectd-plugin/v0.0.2.tar.gz -C /usr/share/collectd/zookeeper-collectd-plugin/ \
    && cp /usr/share/collectd/zookeeper-collectd-plugin/collectd-zookeeper-0.0.2/* /usr/share/collectd/zookeeper-collectd-plugin/ \
    && rm -f /usr/share/collectd/zookeeper-collectd-plugin/v0.0.2.tar.gz \
    && rm -rf /usr/share/collectd/zookeeper-collectd-plugin/collectd-zookeeper-0.0.2

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
COPY python.so /usr/lib/collectd/python.so
COPY aggregation.so /usr/lib/collectd/aggregation.so

RUN chmod +x /usr/sbin/signalfx-agent \
    && rm -f /etc/collectd/collectd.conf \
    && rm -rf /etc/collectd/managed_config/* \
    && rm -rf /etc/collectd/filter_config/*

# default startup command
CMD ["/opt/signalfx-imaging/signalfx-agent.sh"]

FROM docker.elastic.co/elasticsearch/elasticsearch:6.6.1

ENV discovery.type="single-node"
ENV ES_JAVA_OPTS="-Xms128m -Xmx128m"

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["eswrapper"]

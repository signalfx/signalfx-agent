FROM elasticsearch:2.0.2

ENV discovery.type="single-node"
ENV ES_JAVA_OPTS="-Xms128m -Xmx128m"

ENTRYPOINT ["bash", "/docker-entrypoint.sh", "elasticsearch"]

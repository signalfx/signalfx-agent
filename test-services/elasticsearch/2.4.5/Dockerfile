FROM elasticsearch:2.4.5

ENV discovery.type="single-node"
ENV ES_JAVA_OPTS="-Xms128m -Xmx128m"

ENTRYPOINT ["bash", "/docker-entrypoint.sh", "elasticsearch"]

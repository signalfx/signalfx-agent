if [[ -z "$KAFKA_ZOOKEEPER_CONNECT" ]]; then
    echo "ERROR: missing mandatory config: KAFKA_ZOOKEEPER_CONNECT"
    exit 1
fi
sleep 20
./opt/kafka_2.11-"$KAFKA_VERSION"/bin/kafka-topics.sh --create --zookeeper "$KAFKA_ZOOKEEPER_CONNECT" --replication-factor 1 --partitions 1 --topic sfx-employee
